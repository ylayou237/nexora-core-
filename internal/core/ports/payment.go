package ports

import (
	"context"
)

// PaymentGateway abstrait les fournisseurs de paiement (Stripe, Orange Money, MTN MoMo).
// Ce port permet de changer de fournisseur sans toucher au code métier (BillingService).
type PaymentGateway interface {
	// Charge effectue un débit immédiat sur une source (carte, mobile money).
	// Retourne l'ID de transaction externe (ex: "ch_1J2Y...").
	Charge(ctx context.Context, amount uint64, currency string, sourceID string) (string, error)

	// CreateCustomer enregistre le client chez le fournisseur (ex: Stripe Customer).
	// Retourne l'ID client externe (ex: "cus_N1...").
	CreateCustomer(ctx context.Context, email string, name string) (string, error)

	// Refund rembourse une transaction existante (totale ou partielle).
	Refund(ctx context.Context, transactionID string) error

	// GetPaymentStatus vérifie l'état d'une transaction asynchrone (Mobile Money).
	GetPaymentStatus(ctx context.Context, transactionID string) (string, error)
}
