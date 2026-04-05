package mq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/mqErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

var mqProducerTracer = otel.Tracer("guest-manager.mq.producer")

type rabbitmqProducer struct {
	*RabbitMqChannel
	exchangeName  string
	exchangeKind  string
	publishConfig config.PublishConfig
}

func NewRabbitmqProducer(rabbitmqConnection *RabbitMqConnection, publishConfig config.PublishConfig) (port.MqPublisher, error) {
	paymentProducer := rabbitmqProducer{
		RabbitMqChannel: NewRabbitMqChannel(rabbitmqConnection),
		publishConfig:   publishConfig,
	}

	if err := paymentProducer.openChannel(); err != nil {
		return nil, err
	}

	return &paymentProducer, nil
}

func (p *rabbitmqProducer) DeclareExchange(config config.ExchangeConfig) error {
	p.exchangeName = config.Name
	p.exchangeKind = config.Kind
	if config.Kind == "" {
		slog.WarnContext(context.Background(), "exchange kind not specified, defaulting to 'direct'")
		p.exchangeKind = "direct"
	}

	if p.channel == nil || p.channel.IsClosed() {
		if err := p.reopenChannel(context.Background()); err != nil {
			return err
		}
	}

	if err := p.channel.ExchangeDeclare(p.exchangeName, p.exchangeKind,
		config.Durable, config.AutoDelete, config.Internal,
		config.NoWait, nil); err != nil {

		return fmt.Errorf("declare exchange %q: %w", config.Name, err)
	}

	return nil
}

func (p *rabbitmqProducer) Publish(ctx context.Context, message proto.Message, routingKey string) error {
	ctx, span := mqProducerTracer.Start(ctx, "RabbitMQ.Publish")
	defer span.End()
	span.SetAttributes(
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination.name", p.exchangeName),
		attribute.String("messaging.rabbitmq.routing_key", routingKey),
		attribute.String("messaging.message.type", string(message.ProtoReflect().Descriptor().FullName())),
	)

	if p.channel == nil || p.channel.IsClosed() {
		if err := p.reopenChannel(ctx); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "channel_reopen_failed")
			return err
		}
	}

	payload, err := proto.Marshal(message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal_failed")
		slog.ErrorContext(ctx, "failed to marshal message", "error", err)
		return &mqErrors.UnexpectedErr{Msg: "failed to marshal message", Err: err}
	}
	span.SetAttributes(attribute.Int("messaging.message_payload_size_bytes", len(payload)))

	headers := amqp.Table{}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		for k, v := range carrier {
			headers[k] = v
		}
	}
	headers["message_type"] = string(message.ProtoReflect().Descriptor().FullName())

	if err := p.channel.PublishWithContext(
		ctx,
		p.exchangeName,
		routingKey,
		p.publishConfig.Mandatory,
		p.publishConfig.Immediate,
		amqp.Publishing{
			ContentType:  "application/protobuf",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Timestamp:    time.Now(),
		},
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish_failed")
		slog.ErrorContext(ctx, "failed to publish message", "error", err)
		return &mqErrors.UnexpectedErr{Msg: "failed to publish message", Err: err}
	}

	return nil
}
