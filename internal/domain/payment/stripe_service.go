package payment

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/paymentintent"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// StripeService wraps stripe-go usage for payments.
type StripeService struct {
	log            logger.Logger
	currency       string
	successURLBase string
	cancelURLBase  string
}

// NewStripeService configures the global Stripe key and returns a service.
func NewStripeService(secretKey, currency, successURLBase, cancelURLBase string, log logger.Logger) *StripeService {
	if secretKey != "" {
		stripe.Key = secretKey
	}
	if currency == "" {
		currency = "usd"
	}
	return &StripeService{
		log:            log,
		currency:       currency,
		successURLBase: successURLBase,
		cancelURLBase:  cancelURLBase,
	}
}

// CreatePaymentIntent creates a Stripe PaymentIntent for the given order.
func (s *StripeService) CreatePaymentIntent(ctx context.Context, order *repository.Order) (*stripe.PaymentIntent, error) {
	if stripe.Key == "" {
		return nil, fmt.Errorf("stripe key is not configured")
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(order.TotalAmountCents),
		Currency: stripe.String(s.currency),
		Metadata: map[string]string{
			"order_id": fmt.Sprint(order.ID),
		},
	}
	if s.successURLBase != "" {
		params.ReturnURL = stripe.String(fmt.Sprintf("%s?order_id=%d", s.successURLBase, order.ID))
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", err)
	}

	return pi, nil
}

