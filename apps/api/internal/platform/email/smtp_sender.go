package email

import (
	"context"
	"fmt"
	"net/smtp"
)

// SMTPSender is the production implementation of the Sender interface.
type SMTPSender struct {
	host     string
	port     string
	username string
	password string
}

// NewSMTPSender creates a new SMTP sender. Host and port are required.
func NewSMTPSender(host, port, username, password string) (*SMTPSender, error) {
	if host == "" || port == "" {
		return nil, fmt.Errorf("SMTP host and port are required")
	}
	return &SMTPSender{host: host, port: port, username: username, password: password}, nil
}

func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	// Build a simple RFC822 message with HTML body
	headers := ""
	headers += fmt.Sprintf("To: %s\r\n", msg.To[0])
	headers += fmt.Sprintf("Subject: %s\r\n", msg.Subject)
	headers += "MIME-version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
	return smtp.SendMail(addr, auth, msg.From, msg.To, []byte(headers+msg.Body))
}
