package port

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain"
)

type MqPublisher interface {
	Publish(ctx context.Context, message domain.RabbitMqMessage) error
	Close() error
}
