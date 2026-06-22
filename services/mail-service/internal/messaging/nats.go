package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn       *nats.Conn
	logSubject string
}

func NewNATSPublisher(conn *nats.Conn, logSubject string) *NATSPublisher {
	return &NATSPublisher{
		conn:       conn,
		logSubject: logSubject,
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

func (p *NATSPublisher) Close() error {
	if p.conn != nil {
		_ = p.conn.Drain()
		p.conn.Close()
	}
	return nil
}
