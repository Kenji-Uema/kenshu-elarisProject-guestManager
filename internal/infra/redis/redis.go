package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/port"
	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	client *goredis.Client
}

func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username:     cfg.Username,
		Password:     string(cfg.Password),
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	r := &Redis{client: client}
	if err := r.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed for address %s:%d: %w", cfg.Host, cfg.Port, err)
	}

	return r, nil
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
