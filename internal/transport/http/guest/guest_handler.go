package guest

import (
	"guestManager/internal/app"
	"guestManager/internal/domain"
	"guestManager/internal/transport/http/common"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler interface {
	GetGuest(c *gin.Context)
	AddGuest(c *gin.Context)
	UpdateGuest(c *gin.Context)
}

type guestHandler struct {
	service app.GuestService
}

func NewGuestHandler(service app.GuestService) Handler {
	return &guestHandler{service: service}
}

func (g guestHandler) GetGuest(c *gin.Context) {
	var guestIdUri common.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	guestId, err := primitive.ObjectIDFromHex(guestIdUri.Id)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	guest, err := g.service.Get(c.Request.Context(), guestId)

	c.JSON(200, guest)
}

func (g guestHandler) AddGuest(c *gin.Context) {
	var guest domain.Guest

	if err := c.ShouldBindJSON(&guest); err != nil {
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
	var updatedGuest domain.Guest
	var guestIdUri common.GuestIdURI

	if err := c.ShouldBindUri(&guestIdUri); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&updatedGuest); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	targetGuestId, err := primitive.ObjectIDFromHex(guestIdUri.Id)

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
