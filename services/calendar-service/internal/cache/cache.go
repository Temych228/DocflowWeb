package cache

import (
	"context"
	"time"
)

type EventCache interface {
	GetMonth(ctx context.Context, userID string, year, month int) ([]byte, bool)
	SetMonth(ctx context.Context, userID string, year, month int, value []byte, ttl time.Duration)
	InvalidateMonth(ctx context.Context, userID string, year, month int)
}
