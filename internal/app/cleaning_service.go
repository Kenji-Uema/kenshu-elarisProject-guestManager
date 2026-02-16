package app

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/port"
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
	message, err := domain.NewRabbitMqMessage(
		"ex.cleanRoom", r.RoomName(), "application/json", nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create message", "error", err)
		return err
	}

	return c.publisher.Publish(ctx, message)
}

func (c cleaningService) MakeupRoom(ctx context.Context, r domain.CleaningRequest) error {
	message, err := domain.NewRabbitMqMessage(
		"ex.makeupRoom", r.RoomName(), "application/json", nil)
	if err != nil {
		return err
	}

	return c.publisher.Publish(ctx, message)
}
