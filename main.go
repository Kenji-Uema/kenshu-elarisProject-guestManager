package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/infra/mdb"
	"github.com/Kenji-Uema/guestManager/internal/infra/mq"
	"github.com/Kenji-Uema/guestManager/internal/tooling/log"
	"github.com/Kenji-Uema/guestManager/internal/transport/grpc/clock"
	"github.com/Kenji-Uema/guestManager/internal/transport/http"
	"github.com/gin-gonic/gin"
)

func exitOnError(errMsg string, err error) {
	if err != nil {
		slog.Error(errMsg, "error", err)
		os.Exit(1)
	}
}

func main() {
	slog.SetDefault(log.NewJsonLogger())
	baseCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	configs, err := config.LoadConfigs()
	exitOnError("failed to load configs", err)

	slog.Info("starting guestManager")

	rabbitmqClient, err := mq.NewRabbitMqConnection(configs.RabbitMqConfig)
	exitOnError("failed to create rabbitmq connection", err)
	defer func() {
		if err := rabbitmqClient.Close(); err != nil {
			slog.Error("close rabbitmq connection", "error", err)
		}
	}()

	cleaningPublisher, err := mq.NewRabbitMqPublisher(rabbitmqClient, configs.CleaningExchangeConfig)
	exitOnError("failed to create rabbitmq publisher", err)
	defer func() {
		if err := cleaningPublisher.Close(); err != nil {
			slog.Error("close publisher", "error", err)
		}
	}()

	mongoDb, err := mdb.NewMongoDb(baseCtx, configs.MongoConfig)
	exitOnError("mongo init", err)
	defer func() {
		closeCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()
		if err := mongoDb.Close(closeCtx); err != nil {
			slog.Error("failed to close mongo connection", "error", err)
		}
	}()

	clockEmu, err := clock.NewClockEmu(configs.ClockEmuConfig)
	exitOnError("failed to create grpc clockEmu", err)
	defer clockEmu.Close()

	guestRepo := mdb.NewGuestRepo(mongoDb.Database, configs.GuestCollectionConfig)

	cleaningService := app.NewCleaningService(cleaningPublisher)
	guestService := app.NewGuestService(guestRepo)

	cleaningHandler := http.NewCleaningHandler(cleaningService)
	guestHandler := http.NewGuestHandler(guestService, clockEmu)
	probeHandler := http.NewProbeHandler(mongoDb, rabbitmqClient)

	router := gin.Default()

	router.GET("/guest/:userId", guestHandler.GetGuest)
	router.GET("/guest/:userId/bookings", nil)
	router.POST("/guest", guestHandler.AddGuest)
	router.PATCH("/guest/:userId", guestHandler.UpdateGuest)

	router.POST("/checkin/:userId/reservation/:reservationId", nil)
	router.POST("/checkout/:userId/reservation/:reservationId", nil)

	router.POST("/clean/:roomNumber", cleaningHandler.CleanRoom)

	router.POST("/consume/:roomNumber/item/:itemName", nil)

	router.GET("/healthz", probeHandler.Heath)
	router.GET("/readyz", probeHandler.Ready)

	err = router.Run(fmt.Sprintf("%s:%d", configs.AppConfig.Host, configs.AppConfig.Port))
	if err != nil {
		return
	}

	stop()
}
