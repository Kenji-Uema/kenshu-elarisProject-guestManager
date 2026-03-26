package transport

import (
	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/infra/mdb"
	"github.com/Kenji-Uema/guestManager/internal/infra/mq"
	redisc "github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/Kenji-Uema/guestManager/internal/port"
	transporthttp "github.com/Kenji-Uema/guestManager/internal/transport/http"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func NewGin(cfg config.AppConfig, services app.Services, clockClient port.ClockClient,
	mongoClient *mdb.Mdb, rabbitmqConnection *mq.RabbitMqConnection, redisClient *redisc.Redis) *gin.Engine {
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware(cfg.ServiceName))

	guestHandler := transporthttp.NewGuestHandler(services.Guest, clockClient)
	websocketHandler := websocket.NewWebsocket(
		services.Reception,
		services.Cleaning,
		clockClient,
		services.Notification,
		services.TimeEvents,
	)
	probeHandler := transporthttp.NewProbeHandler(mongoClient, rabbitmqConnection, redisClient)

	router.GET("/guest/:userId", guestHandler.GetGuest)
	router.GET("/guest/:userId/bookings", guestHandler.GetBookings)
	router.POST("/guest", guestHandler.AddGuest)
	router.PATCH("/guest/:userId", guestHandler.UpdateGuest)

	router.POST("/consume/:roomNumber/item/:itemName", nil)
	router.GET("/lodging/chat", websocketHandler.Handle)

	router.GET("/healthz", probeHandler.Heath)
	router.GET("/readyz", probeHandler.Ready)

	return router
}
