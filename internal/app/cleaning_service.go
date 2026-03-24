package app

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/port"
)

type CleaningService interface {
	CleanRoom(ctx context.Context, r domain.CleaningRequest) error
}

type cleaningService struct {
	publisher port.MqPublisher
}

func NewCleaningService(publisher port.MqPublisher) CleaningService {
	return &cleaningService{publisher: publisher}
}

func (c cleaningService) CleanRoom(ctx context.Context, r domain.CleaningRequest) error {
	cleaningRequest := &dto.CleaningRequest{
		RoomName: r.RoomName,
		Request:  c.mapCleaningRequestType(r.RequestType),
	}

	return c.publisher.Publish(ctx, cleaningRequest, "")
}

func (c cleaningService) mapCleaningRequestType(cleaningType domain.CleaningRequestType) dto.RequestType {
	switch cleaningType {
	case domain.FullCleaning:
		return dto.RequestType_FULL_CLEANING
	case domain.DailyCleaning:
		return dto.RequestType_DAILY_CLEANING
	case domain.PrepareForSleep:
		return dto.RequestType_PREPARE_FOR_SLEEP
	case domain.PrepareForGuest:
		return dto.RequestType_PREPARE_FOR_GUEST
	default:
		return dto.RequestType_UNSPECIFIED
	}
}
