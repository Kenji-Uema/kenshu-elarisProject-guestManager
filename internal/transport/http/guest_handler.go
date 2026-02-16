package http

import (
	"log/slog"
	"net/http"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/transport/grpc/clock"
	"github.com/Kenji-Uema/guestManager/internal/transport/http/bindings"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GuestHandler interface {
	GetGuest(c *gin.Context)
	AddGuest(c *gin.Context)
	UpdateGuest(c *gin.Context)
}

type guestHandler struct {
	service     app.GuestService
	clockClient *clock.Emu
}

func NewGuestHandler(service app.GuestService, clockClient *clock.Emu) GuestHandler {
	return &guestHandler{service: service, clockClient: clockClient}
}

func (h guestHandler) GetGuest(c *gin.Context) {
	var guestIdUri bindings.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	guestId, err := bson.ObjectIDFromHex(guestIdUri.Id)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	guest, err := h.service.GetById(c.Request.Context(), guestId)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, guest.ToDto())
}

func (h guestHandler) AddGuest(c *gin.Context) {
	var guestRequest dto.GuestDto

	if err := c.ShouldBindJSON(&guestRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdTime, err := h.clockClient.Now(c.Request.Context())
	if err != nil {
		slog.ErrorContext(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get clock time"})
		return
	}

	guest, err := domain.NewGuest(
		bson.NewObjectID(),
		guestRequest.DocumentId,
		guestRequest.GivenNames,
		guestRequest.Surname,
		guestRequest.Email,
		createdTime,
		nil,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.service.Add(c.Request.Context(), guest)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, id)
}

func (h guestHandler) UpdateGuest(c *gin.Context) {
	var updatedRequest dto.GuestDto
	var guestIdUri bindings.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&updatedRequest); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	targetGuestId, err := bson.ObjectIDFromHex(guestIdUri.Id)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	createdTime, err := h.clockClient.Now(c.Request.Context())
	if err != nil {
		slog.ErrorContext(c.Request.Context(), err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get clock time"})
		return
	}

	updatedGuest, err := domain.NewGuest(
		bson.NewObjectID(),
		updatedRequest.DocumentId,
		updatedRequest.GivenNames,
		updatedRequest.Surname,
		updatedRequest.Email,
		nil,
		createdTime,
	)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	resultedGuest, err := h.service.Update(c.Request.Context(), targetGuestId, updatedGuest)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, resultedGuest)
}
