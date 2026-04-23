package providers

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

type FreeSMSProvider struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string
}

func NewFreeSMSProvider() *FreeSMSProvider {
	return &FreeSMSProvider{
		smtpHost:     os.Getenv("SMTP_HOST"),
		smtpPort:     os.Getenv("SMTP_PORT"),
		smtpUser:     os.Getenv("SMTP_USER"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
		fromEmail:    os.Getenv("RESEND_API_KEY"),
	}
}

func (p *FreeSMSProvider) SendSMS(ctx context.Context, to, message string) error {
	if p.smtpHost == "" {
		return fmt.Errorf("SMTP not configured for SMS gateway")
	}

	cleanPhone := strings.ReplaceAll(to, "+52", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, " ", "")
	cleanPhone = strings.ReplaceAll(cleanPhone, "-", "")

	var gatewayEmail string
	if len(cleanPhone) == 10 {
		gatewayEmail = fmt.Sprintf("%s@sms.telcel.com", cleanPhone)
	} else {
		return fmt.Errorf("invalid phone number format")
	}

	auth := smtp.PlainAuth("", p.smtpUser, p.smtpPassword, p.smtpHost)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: Codigo Validafianza\r\n"+
		"\r\n"+
		"%s\r\n", gatewayEmail, message))

	addr := fmt.Sprintf("%s:%s", p.smtpHost, p.smtpPort)
	err := smtp.SendMail(addr, auth, p.fromEmail, []string{gatewayEmail}, msg)
	if err != nil {
		return fmt.Errorf("SMS gateway error: %w", err)
	}

	return nil
}
