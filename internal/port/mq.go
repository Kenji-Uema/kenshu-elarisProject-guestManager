package port

import (
	"context"
	"guestManager/internal/config"
	"guestManager/internal/domain"
)

type MqClient interface {
	CloseChannel()
	EnsurePublisherTopology(config config.CleaningExchangeConfig) error
}

type MqPublisher interface {
	Publish(ctx context.Context, message domain.RabbitMqMessage) error
}
