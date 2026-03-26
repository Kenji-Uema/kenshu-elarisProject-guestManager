package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/infra/mdb"
	"github.com/Kenji-Uema/guestManager/internal/infra/mq"
	"github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/Kenji-Uema/guestManager/internal/port"
	goredis "github.com/redis/go-redis/v9"
)

type Mongo struct {
	Connection      *mdb.Mdb
	BookingRepo     port.BookingRepo
	GuestRepo       port.GuestRepo
	CottageRepo     port.CottageRepo
	ConnectionClose func(context.Context) error
}

type Rabbitmq struct {
	Connection         *mq.RabbitMqConnection
	CleaningPublisher  port.MqPublisher
	DayChangeConsumer  port.MqConsumer
	HourChangeConsumer port.MqConsumer
	ConnectionClose    func(context.Context) error
}

type Redis struct {
	Client *redis.Redis
	Close  func() error
}

func NewMongoDb(ctx context.Context, configs config.MongoConfig) (Mongo, error) {
	mongoDB, err := mdb.NewMongoDb(ctx, configs)
	if err != nil {
		return Mongo{}, err
	}

	cottageRepo := mdb.NewCottageRepo(mongoDB.Database, configs.Collections.Cottage)
	guestRepo := mdb.NewGuestRepo(mongoDB.Database, configs.Collections.Guest)
	bookingRepo := mdb.NewBookingRepo(mongoDB.Database, configs.Collections.Booking)
	return Mongo{
		Connection:      mongoDB,
		BookingRepo:     bookingRepo,
		GuestRepo:       guestRepo,
		CottageRepo:     cottageRepo,
		ConnectionClose: mongoDB.Close,
	}, nil
}

func NewRabbitmq(ctx context.Context, configs config.RabbitMqConfig) (Rabbitmq, error) {
	cleanup := make([]func(context.Context) error, 0, 4)

	rabbitConn, err := mq.NewRabbitMqConnection(ctx, configs)
	if err != nil {
		return Rabbitmq{}, err
	}
	cleanup = append(cleanup, func(context.Context) error {
		return rabbitConn.Close()
	})

	cleaningPublisher, err := mq.NewRabbitmqProducer(rabbitConn, configs.Producers.Cleaning.Publish)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, err
	}
	if err := cleaningPublisher.DeclareExchange(configs.Producers.Cleaning.Exchange); err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, fmt.Errorf("declare cleaning exchange: %w", err)
	}
	cleanup = append(cleanup, func(context.Context) error {
		return cleaningPublisher.CloseChannel()
	})

	dayChangeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.Consumers.DayChange.Consume)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, err
	}
	cleanup = append(cleanup, func(context.Context) error {
		return dayChangeConsumer.CloseChannel()
	})

	hourChangeConsumer, err := mq.NewRabbitmqConsumer(rabbitConn, configs.Consumers.HourChange.Consume)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, err
	}
	cleanup = append(cleanup, func(context.Context) error {
		return hourChangeConsumer.CloseChannel()
	})

	if err := declareAndBindConsumer(ctx, dayChangeConsumer, configs.Consumers.DayChange.Queue, configs.Consumers.DayChange.Binding); err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, err
	}
	if err := declareAndBindConsumer(ctx, hourChangeConsumer, configs.Consumers.HourChange.Queue, configs.Consumers.HourChange.Binding); err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, err
	}

	return Rabbitmq{
		Connection:         rabbitConn,
		CleaningPublisher:  cleaningPublisher,
		DayChangeConsumer:  dayChangeConsumer,
		HourChangeConsumer: hourChangeConsumer,
		ConnectionClose:    func(ctx context.Context) error { return runCleanup(ctx, cleanup) },
	}, nil
}

func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username:     string(cfg.Username),
		Password:     string(cfg.Password),
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	r := redis.NewRedisClient(client)
	if err := r.Ping(ctx); err != nil {
		_ = client.Close()
		return Redis{}, fmt.Errorf("redis ping failed for address %s:%d: %w", cfg.Host, cfg.Port, err)
	}

	return Redis{
		Client: r,
		Close:  r.Close,
	}, nil
}

func declareAndBindConsumer(ctx context.Context, consumer port.MqConsumer, queueCfg config.QueueConfig, bindingCfg config.BindingConfig) error {
	if err := consumer.DeclareQueue(ctx, queueCfg); err != nil {
		return err
	}

	return consumer.BindQueue(ctx, bindingCfg)
}

func runCleanup(ctx context.Context, cleanup []func(context.Context) error) error {
	var shutdownErr error
	for i := len(cleanup) - 1; i >= 0; i-- {
		if err := cleanup[i](ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}
	}
	return shutdownErr
}
