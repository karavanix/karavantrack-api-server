package smtp

import (
	"context"
	"fmt"
	"html/template"

	gomail "github.com/wneessen/go-mail"
)

type SmtpClient struct {
	client *gomail.Client
	name   string
	from   string
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Name     string
	From     string
}

func New(cfg Config) (*SmtpClient, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("smtp: host is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("smtp: username is required")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("smtp: password is required")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("smtp: from is required")
	}

	opts := []gomail.Option{
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(cfg.Username),
		gomail.WithPassword(cfg.Password),
	}
	if cfg.Port != 0 {
		opts = append(opts, gomail.WithPort(cfg.Port))
	}

	client, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("smtp: failed to create client: %w", err)
	}

	return &SmtpClient{
		client: client,
		name:   cfg.Name,
		from:   cfg.From,
	}, nil
}

func (c *SmtpClient) SendHTML(ctx context.Context, to, subject, body string) error {
	msg := gomail.NewMsg()
	if err := msg.FromFormat(c.name, c.from); err != nil {
		return fmt.Errorf("smtp: failed to set From: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("smtp: failed to set To: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(gomail.TypeTextHTML, body)

	if err := c.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("smtp: failed to send email: %w", err)
	}
	return nil
}

func (c *SmtpClient) SendTemplate(ctx context.Context, to, subject string, tmpl *template.Template, data any) error {
	msg := gomail.NewMsg()
	if err := msg.FromFormat(c.name, c.from); err != nil {
		return fmt.Errorf("smtp: failed to set From: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("smtp: failed to set To: %w", err)
	}
	msg.Subject(subject)
	if err := msg.SetBodyHTMLTemplate(tmpl, data); err != nil {
		return fmt.Errorf("smtp: failed to render template: %w", err)
	}

	if err := c.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("smtp: failed to send email: %w", err)
	}
	return nil
}

func (c *SmtpClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
