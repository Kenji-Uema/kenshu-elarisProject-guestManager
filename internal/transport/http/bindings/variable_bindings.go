package bindings

type GuestIdURI struct {
	Id string `uri:"userId" binding:"required"`
}

type ReservationIdURI struct {
	Id string `uri:"reservationId" binding:"required"`
}

type RoomURI struct {
	Name string `uri:"roomNumber" binding:"required"`
}

type ItemURI struct {
	Name string `uri:"itemName" binding:"required"`
}

type CheckInOutURI struct {
	GuestIdURI
	ReservationIdURI
}

type ConsumeProductURI struct {
	RoomURI
	ItemURI
}
