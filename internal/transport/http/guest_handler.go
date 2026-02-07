package http

import (
	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/transport/http/bindings"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/gin-gonic/gin"
)

type GuestHandler interface {
	GetGuest(c *gin.Context)
	AddGuest(c *gin.Context)
	UpdateGuest(c *gin.Context)
}

type guestHandler struct {
	service app.GuestService
}

func NewGuestHandler(service app.GuestService) GuestHandler {
	return &guestHandler{service: service}
}

func (g guestHandler) GetGuest(c *gin.Context) {
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

	guest, err := g.service.GetById(c.Request.Context(), guestId)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, guest.ToDto())
}

func (g guestHandler) AddGuest(c *gin.Context) {
	var guestRequest dto.GuestDto

	if err := c.ShouldBindJSON(&guestRequest); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	guest, err := domain.NewGuest(
		bson.NewObjectID(),
		guestRequest.DocumentId,
		guestRequest.GivenNames,
		guestRequest.Surname,
		guestRequest.Email,
	)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id, err := g.service.Add(c.Request.Context(), guest)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, id)
}

func (g guestHandler) UpdateGuest(c *gin.Context) {
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

	updatedGuest, err := domain.NewGuest(
		bson.NewObjectID(),
		updatedRequest.DocumentId,
		updatedRequest.GivenNames,
		updatedRequest.Surname,
		updatedRequest.Email,
	)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	resultedGuest, err := g.service.Update(c.Request.Context(), targetGuestId, updatedGuest)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, resultedGuest)
}
