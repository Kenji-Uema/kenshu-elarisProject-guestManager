package app

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/port"
)

type CleaningService interface {
	CleanRoom(ctx context.Context, r domain.CleaningOrder) error
}

type cleaningService struct {
	publisher port.MqPublisher
}

func NewCleaningService(publisher port.MqPublisher) (CleaningService, error) {
	if err := validation.New().
		NotZeroValue("publisher", publisher).
		Validate(); err != nil {
		return nil, fmt.Errorf("NewCleaningService: %w", err)
	}

	return &cleaningService{publisher: publisher}, nil
}

func (c cleaningService) CleanRoom(ctx context.Context, cleaningRequest domain.CleaningOrder) error {
	return c.publisher.Publish(ctx, cleaningRequest.ToDto(), "")
}
