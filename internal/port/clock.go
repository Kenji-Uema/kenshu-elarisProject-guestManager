package port

import (
	"context"
	"time"
)

type ClockClient interface {
	Now(ctx context.Context) (*time.Time, error)
}
