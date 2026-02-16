package ports

import (
	"context"
)

// NotificationService gère l'envoi de messages aux utilisateurs via différents canaux.
// L'implémentation (Adapter) utilisera SMTP, Twilio, Firebase, etc.
type NotificationService interface {
	// SendEmail envoie un email transactionnel (Facture, Bienvenue, Reset Password).
	SendEmail(ctx context.Context, to string, template string, data interface{}) error

	// SendSMS envoie un SMS (OTP, Alerte Quota).
	SendSMS(ctx context.Context, to string, message string) error

	// SendPush envoie une notification mobile (App Android/iOS).
	SendPush(ctx context.Context, userID string, title, body string) error
}

// EventPublisher permet de publier des événements de domaine pour le découplage (Architecture Event-Driven).
// L'implémentation utilisera NATS JetStream ou Kafka.
type EventPublisher interface {
	// Publish envoie un événement sur un topic (ex: "billing.invoice.paid").
	Publish(ctx context.Context, topic string, event interface{}) error
}
