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

	// GetPaymentLink obtains the payment link information from the downstream api.
	GetPaymentLink(ctx context.Context, id string) (nexiapi.PaymentLinkDto, error)

	// DeletePaymentLink asks the downstream api to delete the given payment link.
	DeletePaymentLink(ctx context.Context, id string) error

	// HandleWebhook requests the payment link referenced in the webhook data and reacts to any new payments
	HandleWebhook(ctx context.Context, webhook nexiapi.WebhookEventDto) error

	// SendErrorNotifyMail notifies us about unexpected conditions in this service so we can look at the logs
	SendErrorNotifyMail(ctx context.Context, operation string, referenceId string, status string) error
}

var (
	WebhookValidationErr    = errors.New("webhook referenced invalid invoice id, must be positive integer")
	WebhookRefIdMismatchErr = errors.New("webhook reference_id differes from paylink reference_id")
)
