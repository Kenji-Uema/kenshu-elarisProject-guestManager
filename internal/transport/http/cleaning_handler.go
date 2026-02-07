package http

import (
	"net/http"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/gin-gonic/gin"
)

type CleaningHandler interface {
	CleanRoom(c *gin.Context)
}

type cleaningHandler struct {
	service app.CleaningService
}

func NewCleaningHandler(service app.CleaningService) CleaningHandler {
	return &cleaningHandler{service: service}
}

func (h cleaningHandler) CleanRoom(c *gin.Context) {
	var needCleanQuery NeedCleanQuery
	var roomUri RoomURI

	if err := c.ShouldBindQuery(&needCleanQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindUri(&roomUri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cleaningRequest, err := domain.NewCleaningRequest(roomUri.Name, needCleanQuery.Clean)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.CleanRoom(c.Request.Context(), cleaningRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Clean room request sent"})
	return
}
