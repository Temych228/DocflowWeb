package sender

import (
	"context"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
)

type Mailer interface {
	Send(ctx context.Context, msg domain.SMTPMessage) error
	Close() error
}
