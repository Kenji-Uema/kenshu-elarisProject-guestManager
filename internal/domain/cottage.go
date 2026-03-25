package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Cottage struct {
	Id             bson.ObjectID
	Name           string
	View           string
	Details        CottageDetails
	Photos         []string
	PricePerNight  float32
	Bookings       []bson.ObjectID
	CurrentGuest   bson.ObjectID
	CleaningStatus enum.CleaningStatus
	Key            Key
}

type CottageDetails struct {
	Description          string
	View                 string
	FurnitureDescription string
	BathroomDescription  string
	AmenitiesDescription string
}

type Key struct {
	Number string
	Holder enum.KeyHolder
}
