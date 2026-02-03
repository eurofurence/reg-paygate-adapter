package paymentlinksrv

import (
	"context"
	"errors"
	"net/url"

	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
)

type PaymentLinkService interface {
	// ValidatePaymentLinkRequest checks the nexiapi.PaymentLinkRequestDto for validity.
	//
	// The returned url.Values contains detailed error messages that can be used to construct a meaningful response.
	// It is nil if no validation errors were encountered. Any errors encountered are also logged.
	ValidatePaymentLinkRequest(ctx context.Context, data nexiapi.PaymentLinkRequestDto) url.Values

	// CreatePaymentLink expects an already validated nexiapi.PaymentLinkRequestDto, and makes a downstream
	// request to create a payment link, returning the nexiapi.PaymentLinkDto with all its information and the
	// id under which to manage the payment link.
	CreatePaymentLink(ctx context.Context, request nexiapi.PaymentLinkRequestDto) (nexiapi.PaymentLinkDto, string, error)

	// GetPayment obtains the payment information from the downstream api.
	GetPayment(ctx context.Context, id string) (nexiapi.PaymentDto, error)

	// CheckPaymentStatus can be used to process an existing pending payment as if a webhook was received
	//
	// id is a reference id. First it gets the payment from payment service to ensure it exists, then it
	CheckPaymentStatus(ctx context.Context, id string) (nexiapi.PaymentDto, error)

	// LogRawWebhook logs the payload of an incoming webhook both in the DB and the service log
	LogRawWebhook(ctx context.Context, payload string) error

	// HandleWebhook requests the payment referenced in the webhook data and reacts to any payment status updates
	HandleWebhook(ctx context.Context, webhook nexiapi.WebhookDto) error

	// SendErrorNotifyMail notifies us about unexpected conditions in this service so we can look at the logs
	SendErrorNotifyMail(ctx context.Context, operation string, referenceId string, status string) error
}

var (
	ReceivedEmptyPaylink         = errors.New("received empty paylink")
	WebhookValidationErr         = errors.New("webhook referenced invalid invoice id, must be positive integer")
	WebhookRefIdMismatchErr      = errors.New("webhook reference_id differes from paylink reference_id")
	TransactionStatusError       = errors.New("transaction status blocks update")
	TransactionDataMismatchError = errors.New("transaction data mismatch")
)
