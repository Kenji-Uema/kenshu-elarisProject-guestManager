package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/infra/clock"
	"github.com/Kenji-Uema/guestManager/internal/infra/mdb"
	"github.com/Kenji-Uema/guestManager/internal/infra/mq"
	redisinfra "github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/Kenji-Uema/guestManager/internal/infra/telemetry"
	transporthttp "github.com/Kenji-Uema/guestManager/internal/transport/http"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	Router  *gin.Engine
	cleanup []func(context.Context) error
}

func BuildApp(ctx context.Context, cfg config.Configs) (*App, error) {
	cleanup := make([]func(context.Context) error, 0, 5)

	shutdownTelemetry, err := telemetry.Init(ctx, cfg.TelemetryConfig, cfg.AppConfig)
	if err != nil {
		return nil, fmt.Errorf("init telemetry: %w", err)
	}
	cleanup = append(cleanup, shutdownTelemetry)

	rabbitmqClient, err := mq.NewRabbitMqConnection(ctx, cfg.RabbitMqConfig)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("create rabbitmq connection: %w", err)
	}
	cleanup = append(cleanup, func(context.Context) error {
		return rabbitmqClient.Close()
	})

	cleaningPublisher, err := mq.NewRabbitmqProducer(rabbitmqClient, config.PublishConfig{})
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("create rabbitmq publisher: %w", err)
	}
	if err := cleaningPublisher.DeclareExchange(cfg.CleaningExchangeConfig); err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("declare cleaning exchange: %w", err)
	}
	cleanup = append(cleanup, func(context.Context) error {
		return cleaningPublisher.CloseChannel()
	})

	mongoDb, err := mdb.NewMongoDb(ctx, cfg.MongoConfig)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("init mongo: %w", err)
	}
	cleanup = append(cleanup, func(ctx context.Context) error {
		closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return mongoDb.Close(closeCtx)
	})

	redisClient, err := redisinfra.NewRedisClient(ctx, cfg.RedisConfig)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("init redis: %w", err)
	}
	cleanup = append(cleanup, func(context.Context) error {
		return redisClient.Close()
	})

	clockEmu, err := clock.NewClockEmu(cfg.ClockEmuConfig)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("create grpc clockEmu: %w", err)
	}
	cleanup = append(cleanup, func(context.Context) error {
		return clockEmu.Close()
	})

	guestRepo := mdb.NewGuestRepo(mongoDb.Database, cfg.GuestCollectionConfig)
	bookingRepo := mdb.NewBookingRepo(mongoDb.Database, cfg.BookingCollectionConfig)
	cottageRepo := mdb.NewCottageRepo(mongoDb.Database, cfg.CottageCollectionConfig)

	clockEventConsumer, err := mq.NewRabbitmqConsumer(rabbitmqClient, config.ConsumeConfig{})
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("create rabbitmq consumer: %w", err)
	}
	cleanup = append(cleanup, func(context.Context) error {
		return clockEventConsumer.CloseChannel()
	})

	cleaningService := app.NewCleaningService(cleaningPublisher)
	guestService := app.NewGuestService(guestRepo, bookingRepo)
	receptionService, err := app.NewReceptionService(guestService, cleaningService, cottageRepo, bookingRepo, clockEventConsumer, redisClient)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return nil, fmt.Errorf("create reception service: %w", err)
	}

	cleaningHandler := transporthttp.NewCleaningHandler(cleaningService)
	guestHandler := transporthttp.NewGuestHandler(guestService, clockEmu)
	wsServer := websocket.NewWebsocket(receptionService, cleaningService, clockEmu, nil, nil)
	probeHandler := transporthttp.NewProbeHandler(mongoDb, rabbitmqClient, redisClient)

	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware(cfg.AppConfig.ServiceName))
	registerRoutes(router, guestHandler, cleaningHandler, wsServer, probeHandler)

	return &App{Router: router, cleanup: cleanup}, nil
}

func (a *App) Close(ctx context.Context) error {
	return runCleanup(ctx, a.cleanup)
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

func registerRoutes(router *gin.Engine, guestHandler transporthttp.GuestHandler, cleaningHandler transporthttp.CleaningHandler,
	websocket websocket.Ws, probeHandler transporthttp.ProbeHandler) {
	router.GET("/guest/:userId", guestHandler.GetGuest)
	router.GET("/guest/:userId/bookings", guestHandler.GetBookings)
	router.POST("/guest", guestHandler.AddGuest)
	router.PATCH("/guest/:userId", guestHandler.UpdateGuest)

	router.POST("/clean/:roomNumber", cleaningHandler.CleanRoom)

	router.POST("/consume/:roomNumber/item/:itemName", nil)
	router.GET("/lodging/chat", websocket.Handle)

	router.GET("/healthz", probeHandler.Heath)
	router.GET("/readyz", probeHandler.Ready)
}
