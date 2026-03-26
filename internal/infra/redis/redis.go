package redis

import (
	"context"
	"errors"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/port"
	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	client *goredis.Client
}

func NewRedisClient(client *goredis.Client) *Redis {
	return &Redis{client: client}
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Client() *goredis.Client {
	return r.client
}

func (r *Redis) GetBytes(ctx context.Context, key string) ([]byte, error) {
	value, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, port.ErrCacheMiss
	}
	return value, err
}

func (r *Redis) SetBytes(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}
