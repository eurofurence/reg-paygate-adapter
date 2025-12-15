package nexiapi

import (
	"encoding/json"
)

type WebhookDto struct {
	Id             string          `json:"id"`
	Event          string          `json:"event"`
	Timestamp      string          `json:"timestamp"`
	MerchantId     int64           `json:"merchantId"`
	MerchantNumber int64           `json:"merchantNumber"`
	Data           json.RawMessage `json:"data"`
}

const EventPaymentCheckoutCompleted = "payment.checkout.completed"

// DataPaymentCheckoutCompleted is Data when event is EventPaymentCheckoutCompleted.
//
// Triggered when the customer has completed the checkout.
type DataPaymentCheckoutCompleted struct {
	Order struct {
		Amount struct {
			Amount   string `json:"amount"`
			Currency string `json:"currency"`
		} `json:"amount"`
		Reference   string `json:"reference"`
		Description string `json:"description"`
		OrderItems  []struct {
			GrossTotalAmount string `json:"grossTotalAmount"`
			Name             string `json:"name"`
			NetTotalAmount   string `json:"netTotalAmount"`
			Quantity         string `json:"quantity"`
			Reference        string `json:"reference"`
			TaxRate          string `json:"taxRate"`
			TaxAmount        string `json:"taxAmount"`
			Unit             string `json:"unit"`
			UnitPrice        string `json:"unitPrice"`
		} `json:"orderItems"`
	} `json:"order"`
	Consumer struct {
		BillingAddress struct {
			AddressLine1 string `json:"addressLine1"`
			AddressLine2 string `json:"addressLine2"`
			City         string `json:"city"`
			Country      string `json:"country"`
			Postcode     string `json:"postcode"`
			ReceiverLine string `json:"receiverLine"`
		} `json:"billingAddress"`
		Country           string `json:"country"`
		Email             string `json:"email"`
		Ip                string `json:"ip"`
		MerchantReference string `json:"merchantReference"`
		PhoneNumber       struct {
			Prefix string `json:"prefix"`
			Number string `json:"number"`
		} `json:"phoneNumber"`
		ShippingAddress struct {
			AddressLine1 string `json:"addressLine1"`
			AddressLine2 string `json:"addressLine2"`
			City         string `json:"city"`
			Country      string `json:"country"`
			Postcode     string `json:"postcode"`
			ReceiverLine string `json:"receiverLine"`
		} `json:"shippingAddress"`
	} `json:"consumer"`
	MyReference string `json:"myReference"`
	PaymentId   string `json:"paymentId"`
}

const EventPaymentCancelCreated = "payment.cancel.created"

// DataPaymentCancelCreated is Data when event is EventPaymentCancelCreated.
//
// Sent when payment has been cancelled.
type DataPaymentCancelCreated struct {
	CancelId   string `json:"cancelId"`
	OrderItems []struct {
		GrossTotalAmount string `json:"grossTotalAmount"`
		Name             string `json:"name"`
		NetTotalAmount   string `json:"netTotalAmount"`
		Quantity         string `json:"quantity"`
		Reference        string `json:"reference"`
		TaxRate          string `json:"taxRate"`
		TaxAmount        string `json:"taxAmount"`
		Unit             string `json:"unit"`
		UnitPrice        string `json:"unitPrice"`
	} `json:"orderItems"`
	MyReference string `json:"myReference"`
	Amount      struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"amount"`
	SurchargeAmount string `json:"surchargeAmount"`
	PaymentId       string `json:"paymentId"`
}

const EventPaymentChargeCreated = "payment.charge.created"

// DataPaymentChargeCreated is Data when event is EventPaymentChargeCreated.
//
// Sent when charge operation is successful.
type DataPaymentChargeCreated struct {
	ChargeId       string `json:"chargeId"`
	InvoiceDetails struct {
		AccountNumber    string `json:"accountNumber"`
		DistributionType string `json:"distributionType"`
		InvoiceDueDate   string `json:"invoiceDueDate"`
		InvoiceNumber    string `json:"invoiceNumber"`
		OcrOrkid         string `json:"ocrOrkid"`
		OurReference     string `json:"ourReference"`
		YourReference    string `json:"yourReference"`
	} `json:"invoiceDetails"`
	OrderItems []struct {
		GrossTotalAmount string `json:"grossTotalAmount"`
		Name             string `json:"name"`
		NetTotalAmount   string `json:"netTotalAmount"`
		Quantity         string `json:"quantity"`
		Reference        string `json:"reference"`
		TaxRate          string `json:"taxRate"`
		TaxAmount        string `json:"taxAmount"`
		Unit             string `json:"unit"`
		UnitPrice        string `json:"unitPrice"`
	} `json:"orderItems"`
	ReservationId           string `json:"reservationId"`
	ReconciliationReference string `json:"reconciliationReference"`
	MyReference             string `json:"myReference"`
	Amount                  struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"amount"`
	SurchargeAmount string `json:"surchargeAmount"`
	PaymentId       string `json:"paymentId"`
}

const EventPaymentChargeCreatedV2 = "payment.charge.created.v2"

// DataPaymentChargeCreatedV2 is Data when event is EventPaymentChargeCreatedV2.
//
// Sent when the customer has successfully been charged, partially or fully.
type DataPaymentChargeCreatedV2 struct {
	ChargeId   string `json:"chargeId"`
	OrderItems []struct {
		GrossTotalAmount int32   `json:"grossTotalAmount"`
		Name             string  `json:"name"`
		NetTotalAmount   int32   `json:"netTotalAmount"`
		Quantity         float64 `json:"quantity"`
		Reference        string  `json:"reference"`
		TaxRate          int32   `json:"taxRate"`
		TaxAmount        int32   `json:"taxAmount"`
		Unit             string  `json:"unit"`
		UnitPrice        int32   `json:"unitPrice"`
	} `json:"orderItems"`
	PaymentMethod           string `json:"paymentMethod"`
	PaymentType             string `json:"paymentType"`
	SubscriptionId          string `json:"subscriptionId"`
	ReconciliationReference string `json:"reconciliationReference"`
	MyReference             string `json:"myReference"`
	Amount                  struct {
		Amount   int32  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"amount"`
	SurchargeAmount int32  `json:"surchargeAmount"`
	PaymentId       string `json:"paymentId"`
}

const EventPaymentChargeFailed = "payment.charge.failed"

// DataPaymentChargeFailed is Data when event is EventPaymentChargeFailed.
//
// Sent when a charge attempt has failed.
type DataPaymentChargeFailed struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Source  string `json:"source"`
	} `json:"error"`
	ChargeId       string `json:"chargeId"`
	InvoiceDetails struct {
		AccountNumber    string `json:"accountNumber"`
		DistributionType string `json:"distributionType"`
		InvoiceDueDate   string `json:"invoiceDueDate"`
		InvoiceNumber    string `json:"invoiceNumber"`
		OcrOrkid         string `json:"ocrOrkid"`
		OurReference     string `json:"ourReference"`
		YourReference    string `json:"yourReference"`
	} `json:"invoiceDetails"`
	OrderItems []struct {
		GrossTotalAmount string `json:"grossTotalAmount"`
		Name             string `json:"name"`
		NetTotalAmount   string `json:"netTotalAmount"`
		Quantity         string `json:"quantity"`
		Reference        string `json:"reference"`
		TaxRate          string `json:"taxRate"`
		TaxAmount        string `json:"taxAmount"`
		Unit             string `json:"unit"`
		UnitPrice        string `json:"unitPrice"`
	} `json:"orderItems"`
	ReservationId           string `json:"reservationId"`
	ReconciliationReference string `json:"reconciliationReference"`
	MyReference             string `json:"myReference"`
	Amount                  struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"amount"`
	SurchargeAmount string `json:"surchargeAmount"`
	PaymentId       string `json:"paymentId"`
}

const EventPaymentChargeFailedV2 = "payment.charge.failed.v2"

// DataPaymentChargeFailedV2 is Data when event is EventPaymentChargeFailedV2.
//
// Sent when a charge attempt has failed.
type DataPaymentChargeFailedV2 struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Source  string `json:"source"`
	} `json:"error"`
	ChargeId   string `json:"chargeId"`
	OrderItems []struct {
		GrossTotalAmount string `json:"grossTotalAmount"`
		Name             string `json:"name"`
		NetTotalAmount   string `json:"netTotalAmount"`
		Quantity         string `json:"quantity"`
		Reference        string `json:"reference"`
		TaxRate          string `json:"taxRate"`
		TaxAmount        string `json:"taxAmount"`
		Unit             string `json:"unit"`
		UnitPrice        string `json:"unitPrice"`
	} `json:"orderItems"`
	PaymentMethod           string `json:"paymentMethod"`
	PaymentType             string `json:"paymentType"`
	SubscriptionId          string `json:"subscriptionId"`
	ReconciliationReference string `json:"reconciliationReference"`
	MyReference             string `json:"myReference"`
	Amount                  struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"amount"`
	SurchargeAmount string `json:"surchargeAmount"`
	PaymentId       string `json:"paymentId"`
}

const EventPaymentCreated = "payment.created"

// DataPaymentCreated is Data when event is EventPaymentCreated.
//
// Sent when a new payment is created.
type DataPaymentCreated struct {
	Order struct {
		Amount struct {
			Amount   int32  `json:"amount"`
			Currency string `json:"currency"`
		} `json:"amount"`
		Reference   string `json:"reference"`
		Description string `json:"description"`
		OrderItems  []struct {
			GrossTotalAmount int32   `json:"grossTotalAmount"`
			Name             string  `json:"name"`
			NetTotalAmount   int32   `json:"netTotalAmount"`
			Quantity         float64 `json:"quantity"`
			Reference        string  `json:"reference"`
			TaxRate          int32   `json:"taxRate"`
			TaxAmount        int32   `json:"taxAmount"`
			Unit             string  `json:"unit"`
			UnitPrice        int32   `json:"unitPrice"`
		} `json:"orderItems"`
	} `json:"order"`
	MyReference    string `json:"myReference"`
	SubscriptionId string `json:"subscriptionId"`
	PaymentId      string `json:"paymentId"`
}
