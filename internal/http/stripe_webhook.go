package apihttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

func handleStripeWebhook(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	if deps.StripeWebhookSecret == "" {
		writeError(w, http.StatusNotImplemented, "stripe webhook is not configured")
		return
	}
	if deps.DB == nil || deps.Payments == nil || deps.Orders == nil {
		writeError(w, http.StatusInternalServerError, "payment backend is not configured")
		return
	}

	const maxBodyBytes = 1 << 20 // 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, sig, deps.StripeWebhookSecret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid signature")
		return
	}

	processed, err := processStripeEvent(r.Context(), deps, event, payload)
	if err != nil {
		deps.Log.Error("stripe webhook processing failed", err, logger.WithContext(r.Context(), map[string]any{
			"event_type": event.Type,
			"event_id":   event.ID,
		}))
		writeError(w, http.StatusInternalServerError, "webhook processing failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"received":  true,
		"processed": processed,
	})
}

func processStripeEvent(ctx context.Context, deps Dependencies, event stripe.Event, rawPayload []byte) (bool, error) {
	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.payment_failed", "payment_intent.canceled":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return false, fmt.Errorf("unmarshal payment intent: %w", err)
		}
		return applyPaymentIntentEvent(ctx, deps, event, &pi, rawPayload)
	default:
		// Ignore unknown events, but return 200 OK.
		return false, nil
	}
}

func applyPaymentIntentEvent(ctx context.Context, deps Dependencies, event stripe.Event, pi *stripe.PaymentIntent, rawPayload []byte) (bool, error) {
	if pi == nil {
		return false, fmt.Errorf("payment intent is required")
	}

	orderIDStr := ""
	if pi.Metadata != nil {
		orderIDStr = pi.Metadata["order_id"]
	}
	orderID, err := strconv.ParseInt(strings.TrimSpace(orderIDStr), 10, 64)
	if err != nil || orderID <= 0 {
		return false, fmt.Errorf("missing or invalid order_id metadata")
	}

	orderStatus := ""
	orderPaymentStatus := "pending"
	paymentStatus := "created"
	switch event.Type {
	case "payment_intent.succeeded":
		orderStatus = "paid"
		orderPaymentStatus = "paid"
		paymentStatus = "succeeded"
	case "payment_intent.payment_failed":
		orderPaymentStatus = "failed"
		paymentStatus = "failed"
	case "payment_intent.canceled":
		orderPaymentStatus = "cancelled"
		paymentStatus = "cancelled"
	}

	piID := pi.ID
	currency := strings.ToUpper(string(pi.Currency))
	amount := pi.Amount
	if amount < 0 {
		amount = 0
	}

	tx, err := deps.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	p := &repository.Payment{
		OrderID:           orderID,
		Provider:          "stripe",
		ProviderPaymentID: &piID,
		AmountCents:       amount,
		CurrencyCode:      currency,
		Status:            paymentStatus,
	}

	updatedPayment, err := deps.Payments.UpsertForOrder(ctx, tx, p)
	if err != nil {
		return false, err
	}

	inserted, err := deps.Payments.InsertLogIfNew(ctx, tx, updatedPayment.ID, event.Type, event.ID, json.RawMessage(rawPayload))
	if err != nil {
		return false, err
	}
	if !inserted {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit tx: %w", err)
		}
		return false, nil
	}

	if err := deps.Orders.UpdatePayment(ctx, tx, orderID, orderStatus, orderPaymentStatus, "stripe", piID); err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}

	return true, nil
}

