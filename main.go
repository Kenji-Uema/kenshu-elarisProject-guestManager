package main

import "github.com/gin-gonic/gin"

func main() {

	router := gin.Default()

	router.GET("/guest/:userId", nil)
	router.GET("/guest/:userId/bookings", nil)
	router.POST("/guest", nil)
	router.PATCH("/guest/:userId", nil)

	router.POST("/checkin/:userId/reservation/:reservationId", nil)
	router.POST("/checkout/:userId/reservation/:reservationId", nil)

	router.POST("/clean/:roomNumber", nil)

	router.POST("/consume/:roomNumber/item/:itemName", nil)

	err := router.Run("localhost:8080")
	if err != nil {
		return
	}
}
