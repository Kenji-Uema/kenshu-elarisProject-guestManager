package cleaning

import (
	"guestManager/internal/app"
	"guestManager/internal/transport/http/common"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	CleanRoom(c *gin.Context)
}

type cleaningHandler struct {
	service app.CleaningService
}

func NewCleaningHandler(service app.CleaningService) Handler {
	return &cleaningHandler{service: service}
}

func (h cleaningHandler) CleanRoom(c *gin.Context) {
	var needCleanQuery common.NeedCleanQuery
	var roomUri common.RoomURI

	if err := c.ShouldBindQuery(&needCleanQuery); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindUri(&roomUri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if needCleanQuery.Clean == common.PleaseClean {
		h.service.CleanRoom(c.Request.Context(), roomUri.Number)

		c.JSON(http.StatusOK, gin.H{"message": "Clean room request sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "(do Not Disturb) Guest did not want to clean the room"})
}
