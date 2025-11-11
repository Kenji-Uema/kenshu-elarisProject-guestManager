package common

type GuestIdURI struct {
	Id string `uri:"id" binding:"required"`
}

type ReservationIdURI struct {
	Id string `uri:"id" binding:"required"`
}

type RoomURI struct {
	Number int `uri:"number" binding:"required"`
}

type ItemURI struct {
	Name string `uri:"name" binding:"required"`
}

type CheckInOutURI struct {
	GuestIdURI
	ReservationIdURI
}

type ConsumeProductURI struct {
	RoomURI
	ItemURI
}
