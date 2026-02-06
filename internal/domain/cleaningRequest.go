package domain

import (
	"strings"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
)

type CleaningRequest struct {
	roomName string
	request  string
}

func NewCleaningRequest(roomName, request string) (CleaningRequest, error) {
	if err := validation.New().NotBlank("roomName", roomName).CleaningRequest(request).Validate(); err != nil {
		return CleaningRequest{}, err
	}

	return CleaningRequest{roomName, request}, nil
}

func (r *CleaningRequest) RoomName() string {
	return strings.TrimSpace(r.roomName)
}
