package fakes

import (
	"context"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/port"
)

var _ port.Cache = (*FakeCache)(nil)

type FakeCache struct {
	GetBytesFn func(ctx context.Context, key string) ([]byte, error)
	SetBytesFn func(ctx context.Context, key string, value []byte, expiration time.Duration) error

	GetBytesCallCount int
	SetBytesCallCount int

	LastGetBytesCtx        context.Context
	LastGetBytesKey        string
	LastSetBytesCtx        context.Context
	LastSetBytesKey        string
	LastSetBytesValue      []byte
	LastSetBytesExpiration time.Duration
}

func (f *FakeCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	f.GetBytesCallCount++
	f.LastGetBytesCtx = ctx
	f.LastGetBytesKey = key

	if f.GetBytesFn != nil {
		return f.GetBytesFn(ctx, key)
	}

	return nil, nil
}

func (f *FakeCache) SetBytes(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	f.SetBytesCallCount++
	f.LastSetBytesCtx = ctx
	f.LastSetBytesKey = key
	f.LastSetBytesValue = value
	f.LastSetBytesExpiration = expiration

	if f.SetBytesFn != nil {
		return f.SetBytesFn(ctx, key, value, expiration)
	}

	return nil
}
