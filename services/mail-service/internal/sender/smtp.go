package sender

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Temych228/DocflowWeb/services/mail-service/internal/domain"
)

type SMTPMailer struct {
	host        string
	port        string
	username    string
	password    string
	from        string
	useTLS      bool
	useStartTLS bool
	skipVerify  bool
	timeout     time.Duration
}

func NewSMTPMailer(host, port, username, password, from string, useTLS, useStartTLS, skipVerify bool, timeout time.Duration) *SMTPMailer {
	return &SMTPMailer{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		from:        from,
		useTLS:      useTLS,
		useStartTLS: useStartTLS,
		skipVerify:  skipVerify,
		timeout:     timeout,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, msg domain.SMTPMessage) error {
	if len(msg.To) == 0 {
		return domain.ErrInvalidInput
	}
	if msg.From == "" {
		msg.From = m.from
	}
	if msg.From == "" {
		return domain.ErrInvalidInput
	}

	addr := net.JoinHostPort(m.host, m.port)
	var conn net.Conn
	var err error

	dialer := &net.Dialer{Timeout: m.timeout}

	if m.useTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName:         m.host,
			InsecureSkipVerify: m.skipVerify,
		})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Close()

	if m.useStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName:         m.host,
				InsecureSkipVerify: m.skipVerify,
			}); err != nil {
				return err
			}
		}
	}

	if m.username != "" {
		auth := smtp.PlainAuth("", m.username, m.password, m.host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(msg.From); err != nil {
		return err
	}
	for _, to := range msg.To {
		if err := client.Rcpt(strings.TrimSpace(to)); err != nil {
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	content := buildMessage(msg.From, msg.To, msg.Subject, msg.Text, msg.HTML)
	if _, err := wc.Write([]byte(content)); err != nil {
		return err
	}

	return nil
}

func (m *SMTPMailer) Close() error {
	return nil
}

func buildMessage(from string, to []string, subject, textBody, htmlBody string) string {
	var buf bytes.Buffer
	boundary := fmt.Sprintf("boundary-%d", time.Now().UnixNano())

	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if htmlBody != "" && textBody != "" {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
		buf.WriteString(textBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
		buf.WriteString(htmlBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		return buf.String()
	}

	if htmlBody != "" {
		buf.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
		buf.WriteString(htmlBody)
		buf.WriteString("\r\n")
		return buf.String()
	}

	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	buf.WriteString(textBody)
	buf.WriteString("\r\n")
	return buf.String()
}
