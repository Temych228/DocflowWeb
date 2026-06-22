package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temych228/DocflowWeb/services/notification-service/internal/domain"
	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn                *nats.Conn
	logSubject          string
	mailJobSubject      string
	notificationSubject string
}

func NewNATSPublisher(conn *nats.Conn, logSubject, mailJobSubject, notificationSubject string) *NATSPublisher {
	return &NATSPublisher{
		conn:                conn,
		logSubject:          logSubject,
		mailJobSubject:      mailJobSubject,
		notificationSubject: notificationSubject,
	}
}

func (p *NATSPublisher) PublishLog(ctx context.Context, event domain.LogEvent) error {
	if p.conn == nil {
		return nil
	}
	event.At = time.Now().UTC()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.conn.Publish(p.logSubject, data)
}

func (p *NATSPublisher) PublishEmailJob(ctx context.Context, job domain.EmailJob) (string, error) {
	if p.conn == nil {
		return "", nil
	}
	if job.JobID == "" {
		job.JobID = newID()
	}
	job.CreatedAt = time.Now().UTC()

	data, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	if err := p.conn.Publish(p.mailJobSubject, data); err != nil {
		return "", err
	}
	return job.JobID, nil
}

func (p *NATSPublisher) PublishNotificationEvent(ctx context.Context, payload any) error {
	if p.conn == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.conn.Publish(p.notificationSubject, data)
}

func (p *NATSPublisher) Close() error {
	if p.conn != nil {
		_ = p.conn.Drain()
		p.conn.Close()
	}
	return nil
}

func newID() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}
