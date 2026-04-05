package redis

import (
	"context"
	"errors"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/port"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Redis struct {
	client *goredis.Client
}

var tracer = otel.Tracer("guest-manager.redis")

func NewRedisClient(client *goredis.Client) *Redis {
	return &Redis{client: client}
}

func (r *Redis) Ping(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "redis PING")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "PING"),
	)

	err := r.client.Ping(ctx).Err()
	recordSpanError(span, err)
	return err
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Client() *goredis.Client {
	return r.client
}

func (r *Redis) GetBytes(ctx context.Context, key string) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "redis GET")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "GET"),
		attribute.String("db.redis.key", key),
	)

	value, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		recordSpanError(span, port.ErrCacheMiss)
		return nil, port.ErrCacheMiss
	}
	recordSpanError(span, err)
	return value, err
}

func (r *Redis) SetBytes(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	ctx, span := tracer.Start(ctx, "redis SET")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "SET"),
		attribute.String("db.redis.key", key),
		attribute.Int("db.redis.value_size", len(value)),
		attribute.String("db.redis.expiration", expiration.String()),
	)

	err := r.client.Set(ctx, key, value, expiration).Err()
	recordSpanError(span, err)
	return err
}

func recordSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
