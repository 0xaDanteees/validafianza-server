package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

type ResendProvider struct {
	apiKey    string
	fromEmail string
	fromName  string
}

func NewResendProvider() *ResendProvider {
	return &ResendProvider{
		apiKey:    os.Getenv("RESEND_API_KEY"),
		fromEmail: os.Getenv("RESEND_FROM_EMAIL"),
		fromName:  os.Getenv("RESEND_FROM_NAME"),
	}
}

func (p *ResendProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	if p.apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not configured")
	}

	client := resend.NewClient(p.apiKey)

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", p.fromName, p.fromEmail),
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("resend error: %w", err)
	}

	return nil
}
