package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
)

var cleaningOrderTypeDtoMap = map[enum.CleaningRequestType]dto.RequestType{
	enum.PrepareForGuest: dto.RequestType_PREPARE_FOR_GUEST,
	enum.DailyCleaning:   dto.RequestType_DAILY_CLEANING,
	enum.FullCleaning:    dto.RequestType_FULL_CLEANING,
	enum.PrepareForSleep: dto.RequestType_PREPARE_FOR_SLEEP,
}

type CleaningOrder struct {
	RoomName    string
	RequestType enum.CleaningRequestType
}

func NewCleaningOrder(roomName string, requestType enum.CleaningRequestType) (CleaningOrder, error) {
	if err := validation.New().NotBlank("RoomName", roomName).Validate(); err != nil {
		return CleaningOrder{}, err
	}

	return CleaningOrder{roomName, requestType}, nil
}

func (r CleaningOrder) ToDto() *dto.CleaningRequest {
	requestType, ok := cleaningOrderTypeDtoMap[r.RequestType]
	if !ok {
		requestType = dto.RequestType_UNSPECIFIED
	}

	return &dto.CleaningRequest{
		RoomName: r.RoomName,
		Request:  requestType,
	}
}
