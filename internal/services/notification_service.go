package services

import (
	"context"
	"fmt"
	"log"
)

type NotificationService struct {
	emailProvider EmailProvider
	smsProvider   SMSProvider
}

type EmailProvider interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

type SMSProvider interface {
	SendSMS(ctx context.Context, to, message string) error
}

type ConsoleEmailProvider struct{}

func (p *ConsoleEmailProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	log.Printf("[EMAIL] To: %s | Subject: %s | Body: %s", to, subject, body)
	return nil
}

type ConsoleSMSProvider struct{}

func (p *ConsoleSMSProvider) SendSMS(ctx context.Context, to, message string) error {
	log.Printf("[SMS] To: %s | Message: %s", to, message)
	return nil
}

func NewNotificationService(emailProvider EmailProvider, smsProvider SMSProvider) *NotificationService {
	if emailProvider == nil {
		emailProvider = &ConsoleEmailProvider{}
	}
	if smsProvider == nil {
		smsProvider = &ConsoleSMSProvider{}
	}

	return &NotificationService{
		emailProvider: emailProvider,
		smsProvider:   smsProvider,
	}
}

func (s *NotificationService) SendPasscode(ctx context.Context, target, code, method string) error {
	switch method {
	case "email":
		subject := "Tu código de acceso - Validafianza"
		// change this to a html template later...
		body := fmt.Sprintf(`
Hola,

Tu código de acceso es: %s

Este código expira en 10 minutos.

Si no solicitaste este código, ignora este mensaje.

Saludos,
Equipo Validafianza
		`, code)
		return s.emailProvider.SendEmail(ctx, target, subject, body)

	case "sms":
		message := fmt.Sprintf("Tu código Validafianza: %s (válido 10 min)", code)
		return s.smsProvider.SendSMS(ctx, target, message)

	default:
		return fmt.Errorf("unsupported delivery method: %s", method)
	}
}
