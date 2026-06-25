package worker

import (
	"context"
	"encoding/json"
	"time"

	pkgmsg "github.com/Temych228/DocflowWeb/pkg/messaging"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/messaging"
	"github.com/Temych228/DocflowWeb/services/mail-service/internal/service"
	"github.com/nats-io/nats.go"
)

type Consumer struct {
	conn       *nats.Conn
	subject    string
	queueGroup string
	service    *service.MailService
	publisher  messaging.Publisher
}

func New(conn *nats.Conn, subject, queueGroup string, svc *service.MailService, publisher messaging.Publisher) *Consumer {
	return &Consumer{
		conn:       conn,
		subject:    subject,
		queueGroup: queueGroup,
		service:    svc,
		publisher:  publisher,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	if c.conn == nil {
		return nil
	}

	_, err := c.conn.QueueSubscribe(c.subject, c.queueGroup, func(msg *nats.Msg) {
		startedAt := time.Now()
		var incoming pkgmsg.EmailJob
		if err := json.Unmarshal(msg.Data, &incoming); err != nil {
			_ = c.publisher.PublishLog(ctx, domain.LogEvent{
				Service: "mail-service",
				Action:  "QueueSubscribe",
				Level:   "error",
				Message: "invalid mail job payload",
				Meta:    map[string]any{"error": err.Error()},
				At:      time.Now().UTC(),
			})
			return
		}

		var job domain.MailJob
		job.JobID = incoming.JobID
		job.NotificationID = incoming.NotificationID
		job.UserID = incoming.UserID
		job.Recipient = incoming.Recipient
		job.TemplateID = incoming.TemplateID
		job.Subject = incoming.Subject
		job.Body = incoming.Body
		job.Variables = incoming.Variables
		job.Category = domain.Category(incoming.Category)
		job.CreatedAt = incoming.CreatedAt

		if _, err := c.service.HandleQueuedJob(ctx, &job); err != nil {
			_ = c.publisher.PublishLog(ctx, domain.LogEvent{
				Service: "mail-service",
				Action:  "HandleQueuedJob",
				Level:   "error",
				Message: "failed to process job",
				JobID:   job.JobID,
				Meta:    map[string]any{"error": err.Error()},
				At:      time.Now().UTC(),
			})
			return
		}

		_ = c.publisher.PublishLog(ctx, domain.LogEvent{
			Service: "mail-service",
			Action:  "HandleQueuedJob",
			Level:   "info",
			Message: "job processed",
			JobID:   job.JobID,
			Meta:    map[string]any{"duration_ms": time.Since(startedAt).Milliseconds()},
			At:      time.Now().UTC(),
		})
	})
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (c *Consumer) Close() error {
	if c.conn != nil {
		_ = c.conn.Drain()
		c.conn.Close()
	}
	return nil
}
