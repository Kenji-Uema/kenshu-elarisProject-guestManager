package port

import (
	"context"
	"guestManager/internal/domain"
)

type MqPublisher interface {
	Publish(ctx context.Context, message domain.RabbitMqMessage) error
	Close() error
}
