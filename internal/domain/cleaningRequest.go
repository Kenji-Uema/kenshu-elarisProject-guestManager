package domain

import (
	"guestManager/internal/domain/errors/appErrors"
	"strings"
)

type CleaningRequest struct {
	roomName string
	request  string
}

func NewCleaningRequest(roomName, request string) (CleaningRequest, error) {
	if strings.TrimSpace(roomName) == "" {
		return CleaningRequest{}, &appErrors.ErrValidationConstrain{
			Field:   "roomName",
			Message: "must not be empty",
		}
	}

	if request != "DO_NOT_DISTURB" && request != "CLEAN" {
		return CleaningRequest{}, &appErrors.ErrValidationConstrain{
			Field:   "request",
			Message: "must be either DO_NOT_DISTURB or CLEAN",
		}
	}

	return CleaningRequest{roomName, request}, nil
}

func (r *CleaningRequest) RoomName() string {
	return strings.TrimSpace(r.roomName)
}
