package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/infra"
	"github.com/Kenji-Uema/guestManager/internal/infra/clock"
	"github.com/Kenji-Uema/guestManager/internal/infra/logging"
	"github.com/Kenji-Uema/guestManager/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestManager/internal/transport"
)

const shutdownTimeout = 5 * time.Second

func exitOnError(ctx context.Context, errMsg string, err error) {
	if err != nil {
		slog.ErrorContext(ctx, errMsg, "error", err)
		os.Exit(1)
	}
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

func main() {
	slog.SetDefault(logging.NewLogger())
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cleanup := make([]func(context.Context) error, 0, 5)
	started := false
	defer func() {
		if started {
			return
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := runCleanup(shutdownCtx, cleanup); err != nil {
			slog.ErrorContext(ctx, "startup cleanup", "error", err)
		}
	}()

	configs, err := config.LoadConfigs()
	exitOnError(ctx, "failed to load configs", err)

	shutdownTelemetry, err := telemetry.Init(ctx, configs.TelemetryConfig, configs.AppConfig)
	exitOnError(ctx, "failed to init telemetry", err)
	cleanup = append(cleanup, shutdownTelemetry)

	mongoDb, err := infra.NewMongoDb(ctx, configs.MongoConfig)
	exitOnError(ctx, "failed to init mongo", err)
	cleanup = append(cleanup, mongoDb.ConnectionClose)

	rabbitmqInfra, err := infra.NewRabbitmq(ctx, configs.RabbitMqConfig)
	exitOnError(ctx, "failed to init rabbitmq", err)
	cleanup = append(cleanup, rabbitmqInfra.ConnectionClose)

	redis, err := infra.NewRedisClient(ctx, configs.RedisConfig)
	exitOnError(ctx, "failed to init redis", err)
	cleanup = append(cleanup, func(context.Context) error {
		return redis.Close()
	})

	clockSimulator, err := clock.NewClockEmu(configs.Services)
	exitOnError(ctx, "failed to create grpc clockEmu", err)
	if closer, ok := clockSimulator.(interface{ Close() error }); ok {
		cleanup = append(cleanup, func(context.Context) error {
			return closer.Close()
		})
	}

	services, err := app.NewServices(ctx, app.Dependencies{
		GuestRepo:             mongoDb.GuestRepo,
		BookingRepo:           mongoDb.BookingRepo,
		CottageRepo:           mongoDb.CottageRepo,
		CleaningPublisher:     rabbitmqInfra.CleaningPublisher,
		GuestCommunicationPub: rabbitmqInfra.GuestCommunicationPub,
		HourChangeConsumer:    rabbitmqInfra.HourChangeConsumer,
		DayChangeConsumer:     rabbitmqInfra.DayChangeConsumer,
		Cache:                 redis.Client,
	})
	exitOnError(ctx, "failed to init app services", err)

	router := transport.NewGin(
		configs.AppConfig,
		services,
		clockSimulator,
		mongoDb.Connection,
		rabbitmqInfra.Connection,
		redis.Client,
	)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", configs.AppConfig.Server.Host, configs.AppConfig.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: time.Duration(configs.AppConfig.Server.ReadHeaderTimeoutInSeconds) * time.Second,
		ReadTimeout:       time.Duration(configs.AppConfig.Server.ReadTimeoutInSeconds) * time.Second,
		WriteTimeout:      time.Duration(configs.AppConfig.Server.WriteTimeoutInSeconds) * time.Second,
		IdleTimeout:       time.Duration(configs.AppConfig.Server.IdleTimeoutInSeconds) * time.Second,
	}
	started = true

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "failed to shutdown http server", "error", err)
		}
	}()

	slog.InfoContext(ctx, "starting guestManager")
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		exitOnError(ctx, "failed to run http server", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := runCleanup(shutdownCtx, cleanup); err != nil {
		slog.ErrorContext(ctx, "shutdown resources", "error", err)
	}

	slog.InfoContext(ctx, "shutdown complete")
}
