package http

import (
	"net/http"

	"github.com/Kenji-Uema/guestManager/internal/infra/mdb"
	"github.com/Kenji-Uema/guestManager/internal/infra/mq"
	redisc "github.com/Kenji-Uema/guestManager/internal/infra/redis"
	"github.com/gin-gonic/gin"
)

type ProbeHandler interface {
	Heath(c *gin.Context)
	Ready(c *gin.Context)
}

type probeHandler struct {
	mongoClient        *mdb.Mdb
	rabbitmqConnection *mq.RabbitMqConnection
	redisClient        *redisc.Redis
}

func NewProbeHandler(mongoClient *mdb.Mdb, rabbitmqConnection *mq.RabbitMqConnection, redisClient *redisc.Redis) ProbeHandler {
	return &probeHandler{mongoClient: mongoClient, rabbitmqConnection: rabbitmqConnection, redisClient: redisClient}
}

func (p probeHandler) Heath(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (p probeHandler) Ready(c *gin.Context) {
	if err := p.mongoClient.Ping(); err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if !p.rabbitmqConnection.IsConnectionOpen() {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	if err := p.redisClient.Ping(c.Request.Context()); err != nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}

	c.Status(http.StatusOK)
}
