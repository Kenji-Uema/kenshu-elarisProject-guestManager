package port

import "context"

type MqClient interface {
	Close()
	EnsureTopology() error
	EnablePublisherConfirms() error
	Publish(ctx context.Context, body []byte) error
}
