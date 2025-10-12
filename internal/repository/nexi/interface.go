package nexi

import (
	"context"
	"errors"
	"time"
)

type NexiDownstream interface {
	CreatePaymentLink(ctx context.Context, request NexiCreatePaymentRequest) (NexiPaymentLinkCreated, error)
	QueryPaymentLink(ctx context.Context, paymentId string) (NexiPaymentQueryResponse, error)
	DeletePaymentLink(ctx context.Context, paymentId string, amount int64) error

	QueryTransactions(ctx context.Context, timeGreaterThan time.Time, timeLessThan time.Time) ([]TransactionData, error)
}

var (
	NoSuchID404Error = errors.New("payment link id not found")
	DownstreamError  = errors.New("downstream unavailable - see log for details")
	NotSuccessful    = errors.New("response body status field did not indicate success")
)

// -- New Nexi API v1 Structures --

type NexiCreatePaymentRequest struct {
	Order          NexiOrder        `json:"order"`
	Checkout       NexiCheckout     `json:"checkout"`
	Appearance     NexiAppearance   `json:"appearance"`
	Notifications  NexiNotifications `json:"notifications"`
}

type NexiPaymentLinkCreated struct {
	ID          string `json:"id"`
	ReferenceID string `json:"referenceId"`
	Link        string `json:"link"`
}

type NexiOrder struct {
	Items     []NexiOrderItem `json:"items"`
	Amount    int64           `json:"amount"`
	Currency  string          `json:"currency"`
	Reference string          `json:"reference"`
}

type NexiOrderItem struct {
	Reference        string  `json:"reference"`
	Name             string  `json:"name"`
	Quantity         float64 `json:"quantity"`
	Unit             string  `json:"unit"`
	UnitPrice        int64   `json:"unitPrice"`
	TaxRate          float64 `json:"taxRate"`
	TaxAmount        int64   `json:"taxAmount"`
	GrossTotalAmount int64   `json:"grossTotalAmount"`
	NetTotalAmount   int64   `json:"netTotalAmount"`
	ImageUrl         string  `json:"imageUrl"`
}

type NexiCheckout struct {
	Url                         string           `json:"url"`
	IntegrationType             string           `json:"integrationType"`
	ReturnUrl                   string           `json:"returnUrl"`
	CancelUrl                   string           `json:"cancelUrl"`
	Consumer                    NexiConsumer     `json:"consumer"`
	TermsUrl                    string           `json:"termsUrl"`
	MerchantTermsUrl            string           `json:"merchantTermsUrl"`
	ShippingCountries           []NexiCountry    `json:"shippingCountries"`
	Shipping                    NexiShipping     `json:"shipping"`
	ConsumerType                NexiConsumerType `json:"consumerType"`
	Charge                      bool             `json:"charge"`
	PublicDevice                bool             `json:"publicDevice"`
	MerchantHandlesConsumerData bool             `json:"merchantHandlesConsumerData"`
	Appearance                  NexiAppearance   `json:"appearance"`
	CountryCode                 string           `json:"countryCode"`
}

type NexiConsumer struct {
	Reference       string            `json:"reference"`
	Email           string            `json:"email"`
	ShippingAddress NexiAddress       `json:"shippingAddress"`
	BillingAddress  NexiAddress       `json:"billingAddress"`
	PhoneNumber     NexiPhone         `json:"phoneNumber"`
	PrivatePerson   NexiPrivatePerson `json:"privatePerson"`
	Company         NexiCompany       `json:"company"`
}

type NexiAddress struct {
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	PostalCode   string `json:"postalCode"`
	City         string `json:"city"`
	Country      string `json:"country"`
}

type NexiPhone struct {
	Prefix string `json:"prefix"`
	Number string `json:"number"`
}

type NexiPrivatePerson struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type NexiCompany struct {
	Name    string      `json:"name"`
	Contact NexiContact `json:"contact"`
}

type NexiContact struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type NexiCountry struct {
	CountryCode string `json:"countryCode"`
}

type NexiShipping struct {
	Countries                   []NexiCountry `json:"countries"`
	MerchantHandlesShippingCost bool          `json:"merchantHandlesShippingCost"`
	EnableBillingAddress        bool          `json:"enableBillingAddress"`
}

type NexiConsumerType struct {
	Default        string   `json:"default"`
	SupportedTypes []string `json:"supportedTypes"`
}

type NexiAppearance struct {
	DisplayOptions NexiDisplayOptions `json:"displayOptions"`
	TextOptions    NexiTextOptions    `json:"textOptions"`
}

type NexiDisplayOptions struct {
	ShowMerchantName bool `json:"showMerchantName"`
	ShowOrderSummary bool `json:"showOrderSummary"`
}

type NexiTextOptions struct {
	CompletePaymentButtonText string `json:"completePaymentButtonText"`
}

type NexiNotifications struct {
	Webhooks []NexiWebhook `json:"webhooks"`
}

type NexiWebhook struct {
	EventName     string `json:"eventName"`
	Url           string `json:"url"`
	Authorization string `json:"authorization"`
}

type NexiPayment struct {
	PaymentId      string              `json:"paymentId"`
	Summary        NexiSummary         `json:"summary"`
	Consumer       NexiConsumerFull    `json:"consumer"`
	PaymentDetails NexiPaymentDetails  `json:"paymentDetails"`
	OrderDetails   NexiOrderDetails    `json:"orderDetails"`
	Checkout       NexiCheckoutDetails `json:"checkout"`
	Created        string              `json:"created"`
	Refunds        []NexiRefund        `json:"refunds"`
	Charges        []NexiCharge        `json:"charges"`
	Terminated     string              `json:"terminated"`
}

type NexiQueryLowlevelResponseBody struct {
	Payment NexiPayment `json:"payment"`
}

// -- Query response structures --

type NexiPaymentQueryResponse struct {
	ID          string                  `json:"id"`
	Status      string                  `json:"status"`
	ReferenceID string                  `json:"referenceId"`
	Link        string                  `json:"link"`
	Amount      int64                   `json:"amount"`
	Currency    string                  `json:"currency"`
	CreatedAt   int64                   `json:"createdAt"`
	VatRate     float64                 `json:"vatRate"`
	Order       NexiOrderDetails        `json:"order,omitempty"`
	Summary     NexiSummary             `json:"summary,omitempty"`
	Consumer    NexiConsumerFull        `json:"consumer,omitempty"`
	Payments    []NexiPaymentDetails    `json:"payments,omitempty"`
	Refunds     []NexiRefund            `json:"refunds,omitempty"`
	Charges     []NexiCharge            `json:"charges,omitempty"`
}

type NexiOrderDetails struct {
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Reference string `json:"reference"`
	Items     []NexiOrderItem `json:"items,omitempty"`
}

type NexiSummary struct {
	ReservedAmount           int64 `json:"reservedAmount"`
	ReservedSurchargeAmount  int64 `json:"reservedSurchargeAmount"`
	ChargedAmount            int64 `json:"chargedAmount"`
	ChargedSurchargeAmount   int64 `json:"chargedSurchargeAmount"`
	RefundedAmount           int64 `json:"refundedAmount"`
	RefundedSurchargeAmount  int64 `json:"refundedSurchargeAmount"`
	CancelledAmount          int64 `json:"cancelledAmount"`
	CancelledSurchargeAmount int64 `json:"cancelledSurchargeAmount"`
}

type NexiConsumerFull struct {
	ShippingAddress NexiAddressFull       `json:"shippingAddress"`
	Company         NexiCompanyFull       `json:"company"`
	PrivatePerson   NexiPrivatePersonFull `json:"privatePerson"`
	BillingAddress  NexiAddressFull       `json:"billingAddress"`
}

type NexiAddressFull struct {
	AddressLine1 string    `json:"addressLine1"`
	AddressLine2 string    `json:"addressLine2"`
	ReceiverLine string    `json:"receiverLine"`
	PostalCode   string    `json:"postalCode"`
	City         string    `json:"city"`
	Country      string `json:"country"`
	PhoneNumber  NexiPhone `json:"phoneNumber"`
}

type NexiCompanyFull struct {
	MerchantReference  string          `json:"merchantReference"`
	Name               string          `json:"name"`
	RegistrationNumber string          `json:"registrationNumber"`
	ContactDetails     NexiContactFull `json:"contactDetails"`
}

type NexiContactFull struct {
	FirstName   string    `json:"firstName"`
	LastName    string    `json:"lastName"`
	Email       string    `json:"email"`
	PhoneNumber NexiPhone `json:"phoneNumber"`
}

type NexiPrivatePersonFull struct {
	MerchantReference string    `json:"merchantReference"`
	DateOfBirth       string    `json:"dateOfBirth"`
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	Email             string    `json:"email"`
	PhoneNumber       NexiPhone `json:"phoneNumber"`
}

type NexiPaymentDetails struct {
	PaymentType    string             `json:"paymentType"`
	PaymentMethod  string             `json:"paymentMethod"`
	InvoiceDetails NexiInvoiceDetails `json:"invoiceDetails"`
	CardDetails    NexiCardDetails    `json:"cardDetails"`
}

type NexiInvoiceDetails struct {
	InvoiceNumber string `json:"invoiceNumber"`
}

type NexiCardDetails struct {
	MaskedPan  string `json:"maskedPan"`
	ExpiryDate string `json:"expiryDate"`
}

type NexiRefund struct {
	RefundId        string          `json:"refundId"`
	Amount          int64           `json:"amount"`
	SurchargeAmount int64           `json:"surchargeAmount"`
	State           string          `json:"state"`
	LastUpdated     string          `json:"lastUpdated"`
	OrderItems      []NexiOrderItem `json:"orderItems"`
}

type NexiCharge struct {
	ChargeId        string          `json:"chargeId"`
	Amount          int64           `json:"amount"`
	SurchargeAmount int64           `json:"surchargeAmount"`
	Created         string          `json:"created"`
	OrderItems      []NexiOrderItem `json:"orderItems"`
}

type NexiCheckoutDetails struct {
	Url       string `json:"url"`
	CancelUrl string `json:"cancelUrl"`
}

// -- Legacy Transaction Data Structures --

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
