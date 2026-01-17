package nexiapi

type WebhookDto struct {
	PayId               string                 `json:"payId"`
	TransId             string                 `json:"transId"`
	Status              string                 `json:"status"`
	ResponseCode        string                 `json:"responseCode"`
	ResponseDescription string                 `json:"responseDescription"`
	Amount              WebhookAmount          `json:"amount"`
	PaymentMethods      []WebhookPaymentMethod `json:"paymentMethods,omitempty"`
	CreationDate        string                 `json:"creationDate"`
}

type WebhookAmount struct {
	Value    int64  `json:"value"`
	Currency string `json:"currency"`
}

type WebhookPaymentMethod struct {
	Type string `json:"type"`
}
