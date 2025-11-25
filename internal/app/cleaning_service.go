package app

import (
	"context"
	"guestManager/internal/domain"
	"guestManager/internal/port"
	"strconv"
)

type CleaningService interface {
	CleanRoom(ctx context.Context, r domain.CleaningRequest) error
	MakeupRoom(ctx context.Context, r domain.CleaningRequest) error
}

type cleaningService struct {
	publisher port.MqPublisher
}

func NewCleaningService(publisher port.MqPublisher) CleaningService {
	return &cleaningService{publisher: publisher}
}

func (c cleaningService) CleanRoom(ctx context.Context, r domain.CleaningRequest) error {
	if r.Request == "DO_NOT_DISTURB" {
		return nil
	}

	return c.publisher.Publish(ctx, domain.RabbitMqMessage{
		Exchange:    "ex.cleanRoom",
		Key:         strconv.Itoa(r.RoomNumber),
		ContentType: "application/json",
		Body:        nil,
	})
}

func (c cleaningService) MakeupRoom(ctx context.Context, r domain.CleaningRequest) error {
	return c.publisher.Publish(ctx, domain.RabbitMqMessage{
		Exchange:    "ex.makeupRoom",
		Key:         strconv.Itoa(r.RoomNumber),
		ContentType: "application/json",
		Body:        nil,
	})
}
