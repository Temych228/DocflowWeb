package scheduler

import (
	"context"
	"time"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/messaging"
	"github.com/Temych228/DocflowWeb/services/notification-service/internal/service"
)

type Runner struct {
	svc       *service.NotificationService
	publisher messaging.Publisher
	enabled   bool
	stopCh    chan struct{}
}

func NewRunner(svc *service.NotificationService, publisher messaging.Publisher, enabled bool) *Runner {
	return &Runner{
		svc:       svc,
		publisher: publisher,
		enabled:   enabled,
		stopCh:    make(chan struct{}),
	}
}

func (r *Runner) Run(ctx context.Context) {
	if !r.enabled {
		return
	}

	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}

	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-timer.C:
			_ = r.publisher.PublishLog(ctx, domain.LogEvent{
				Service: "notification-service",
				Action:  "DailyCron",
				Level:   "info",
				Message: "daily overdue scan tick",
				At:      time.Now().UTC(),
			})
			timer.Reset(24 * time.Hour)
		}
	}
}

func (r *Runner) Stop() {
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
}
