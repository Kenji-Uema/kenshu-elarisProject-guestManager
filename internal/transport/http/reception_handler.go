package http

import (
	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/gin-gonic/gin"
)

type ReceptionHandler interface {
	CheckIn(c *gin.Context)
	CheckOut(c *gin.Context)
}

type receptionHandler struct {
	service app.ReceptionService
}

func NewReceptionHandler(service app.ReceptionService) ReceptionHandler {
	return &receptionHandler{service: service}
}

func (r receptionHandler) CheckIn(c *gin.Context) {
	//TODO implement me
	panic("implement me")
}

func (r receptionHandler) CheckOut(c *gin.Context) {
	//TODO implement me
	panic("implement me")
}
