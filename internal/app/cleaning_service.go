package app

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var cleaningServiceTracer = otel.Tracer("guest-manager.app.cleaning-service")

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
	publishCtx := ctx
	_, span := cleaningServiceTracer.Start(ctx, "CleaningService.CleanRoom")
	defer span.End()
	span.SetAttributes(
		attribute.String("cleaning.room_name", cleaningRequest.RoomName),
		attribute.String("cleaning.request_type", string(cleaningRequest.RequestType)),
	)

	if err := c.publisher.Publish(publishCtx, cleaningRequest.ToDto(), ""); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish_failed")
		return err
	}

	return nil
}
