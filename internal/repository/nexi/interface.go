package nexi

import (
	"context"
	"errors"
	"time"
)

type NexiDownstream interface {
	CreatePaymentLink(ctx context.Context, request PaymentLinkCreateRequest) (PaymentLinkCreated, error)
	QueryPaymentLink(ctx context.Context, id uint) (PaymentLinkQueryResponse, error)
	DeletePaymentLink(ctx context.Context, id uint) error

	QueryTransactions(ctx context.Context, timeGreaterThan time.Time, timeLessThan time.Time) ([]TransactionData, error)
}

var (
	NoSuchID404Error = errors.New("payment link id not found")
	DownstreamError  = errors.New("downstream unavailable - see log for details")
	NotSuccessful    = errors.New("response body status field did not indicate success")
)

// -- CreatePaymentLink --

// PaymentLinkCreateRequest contains the data to construct the paylink create request body.
//
// Note that the json field names are not actually used because the Nexi API doesn't
// accept JSON, and instead insists on a slightly unusual XWwwFormUrlencoded request.
type PaymentLinkCreateRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PSP         uint64  `json:"psp"`
	ReferenceId string  `json:"referenceId"`
	OrderId     string  `json:"nexiOrderId"`
	Purpose     string  `json:"purpose"`
	Amount      int64   `json:"amount"`  // in cents
	VatRate     float64 `json:"vatRate"` // in %
	Currency    string  `json:"currency"`
	SKU         string  `json:"sku"`
	Email       string  `json:"email"`

	SuccessRedirectUrl string `json:"successRedirectUrl"` // optional - leave empty
	FailedRedirectUrl  string `json:"failedRedirectUrl"`  // optional - leave empty
}

type PaymentLinkCreated struct {
	ID          uint   `json:"id"`
	ReferenceID string `json:"referenceId"`
	Link        string `json:"link"`
}

// -- QueryPaymentLink --

type PaymentLinkQueryResponse struct {
	ID          uint                 `json:"id"` // not the payment link id!
	Status      string               `json:"status"`
	ReferenceID string               `json:"referenceId"`
	Link        string               `json:"link"`
	Invoices    []PaymentLinkInvoice `json:"invoices"`
	Name        string               `json:"name"`
	Purpose     map[string]string    `json:"purpose"`
	Amount      int64                `json:"amount"`
	Currency    string               `json:"currency"`
	CreatedAt   int64                `json:"createdAt"`
	VatRate     float64              `json:"vatRate"`
}

type PaymentLinkInvoice struct {
	ReferenceID      string            `json:"referenceId"`
	PaymentRequestId uint              `json:"paymentRequestId"` // the payment link id
	Currency         string            `json:"currency"`
	Amount           int64             `json:"amount"`
	Transactions     []TransactionData `json:"transactions"`
}

// -- QueryTransactions --

// Status
//
// Successful payment processed (status: confirmed) => book
//
// Payment aborted by customer (status: cancelled) => log info and ignore
// Payment declined (status: declined) => log info and ignore
//
// Order placed (status: waiting) => log warn and notify
//
// Pre-authorization successful (status: authorized) => log error and notify
// Payment (partial-) refunded by merchant (status: refunded / partially-refunded) => log error and notify
// Refund pending (status: refund_pending) (for transactions for which the refund has been initialized but not yet confirmed by the bank) => log error and notify
// Chargeback by card holder (status: chargeback) => log error and notify
// Technical error (status: error) => log error and notify
// Uncaptured (status: uncaptured) (only with PSP Clearhaus Acquiring) => log error and notify
// Reserved (status: reserved) (??? not explained in docs) => log error and notify

type TransactionData struct {
	ID          int64   `json:"id"`
	UUID        string  `json:"uuid"` // sent as merchantOrderId
	Amount      int64   `json:"amount"`
	Status      string  `json:"status"` // react to declined, confirmed, authorized, what else?
	Time        string  `json:"time"`   // take effective date from first 10 chars (ISO Date)
	Lang        string  `json:"lang"`   // ISO 639-1 of shopper language (de, en)
	PageUUID    string  `json:"pageUuid"`
	Payment     Payment `json:"payment"`
	Psp         string  `json:"psp"`   // Name of the payment service provider used, for example "ConCardis_PayEngine_3"
	PspID       int64   `json:"pspId"` // ID of the Psp
	Mode        string  `json:"mode"`  // "LIVE", "TEST"
	ReferenceID string  `json:"referenceId"`
	Invoice     Invoice `json:"invoice"`
}

type Payment struct {
	Brand string `json:"brand"`
}

type Invoice struct {
	ReferenceID      string `json:"referenceId"`
	PaymentRequestId uint   `json:"paymentRequestId"` // the payment link id
	Currency         string `json:"currency"`         // "EUR"
	OriginalAmount   int64  `json:"originalAmount"`
	RefundedAmount   int64  `json:"refundedAmount"`
}
