package domain

import (
	"guestManager/internal/app/validation"
	"strings"
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
