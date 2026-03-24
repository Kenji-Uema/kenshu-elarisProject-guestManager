package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
)

type CleaningRequestType string

const (
	PrepareForGuest = CleaningRequestType("prepareForGuest")
	DailyCleaning   = CleaningRequestType("dailyCleaning")
	FullCleaning    = CleaningRequestType("fullCleaning")
	PrepareForSleep = CleaningRequestType("prepareForSleep")
)

type CleaningRequest struct {
	RoomName    string
	RequestType CleaningRequestType
}

func NewCleaningRequest(roomName string, requestType CleaningRequestType) (CleaningRequest, error) {
	if err := validation.New().NotBlank("RoomName", roomName).Validate(); err != nil {
		return CleaningRequest{}, err
	}

	return CleaningRequest{roomName, requestType}, nil
}
