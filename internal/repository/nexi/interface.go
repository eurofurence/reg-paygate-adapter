package nexi

import (
	"context"
	"errors"
	"time"
)

type NexiDownstream interface {
	CreatePaymentLink(ctx context.Context, request NexiCreateCheckoutSessionRequest) (NexiCreateCheckoutSessionResponse, error)
	QueryPaymentLink(ctx context.Context, transactionId string) (NexiPaymentQueryResponse, error)
	DeletePaymentLink(ctx context.Context, paymentId string, amount int64) error

	QueryTransactions(ctx context.Context, timeGreaterThan time.Time, timeLessThan time.Time) ([]TransactionData, error)
}

var (
	NoSuchID404Error = errors.New("payment link id not found")
	DownstreamError  = errors.New("downstream unavailable - see log for details")
	NotSuccessful    = errors.New("response body status field did not indicate success")
)

// -- New Nexi PayGate API Structures --

// --- NexiCreateCheckoutSessionRequest

type NexiCreateCheckoutSessionRequest struct {
	TransId               string                     `json:"transId"` // required
	ExternalIntegrationId string                     `json:"externalIntegrationId,omitempty"`
	RefNr                 string                     `json:"refNr,omitempty"`
	Amount                NexiAmount                 `json:"amount"` // required
	Language              string                     `json:"language,omitempty"`
	Template              *NexiTemplate              `json:"template,omitempty"`
	CaptureMethod         *NexiCaptureMethod         `json:"captureMethod,omitempty"`
	Order                 *NexiOrder                 `json:"order,omitempty"`
	SimulationMode        string                     `json:"simulationMode,omitempty"`
	Urls                  NexiPaymentUrlsRequest     `json:"urls,omitempty"`
	BillingAddress        *NexiBillingAddress        `json:"billingAddress,omitempty"`
	Shipping              *NexiShipping              `json:"shipping,omitempty"`
	StatementDescriptor   string                     `json:"statementDescriptor,omitempty"`
	CustomerInfo          *NexiCustomerInfoRequest   `json:"customerInfo,omitempty"`
	PartialPayment        bool                       `json:"partialPayment,omitempty"`
	ExpirationTime        string                     `json:"expirationTime,omitempty"` // RFC3339 - e.g. 2025-02-07T16:00:00Z
	FraudData             *NexiFraudData             `json:"fraudData,omitempty"`
	PaymentFacilitator    *NexiPaymentFacilitator    `json:"paymentFacilitator,omitempty"`
	BrowserInfo           *NexiBrowserInfo           `json:"browserInfo,omitempty"`
	Device                *NexiDevice                `json:"device,omitempty"`
	Channel               NexiChannel                `json:"channel,omitempty"` // ECOM | MOTO | APP | PAYBYLINK | POS
	RemittanceInfo        string                     `json:"remittanceInfo,omitempty"`
	Metadata              map[string]string          `json:"metadata,omitempty"`
	AllowedPaymentMethod  []PaymentMethodType        `json:"allowedPaymentMethod,omitempty"`
	ReferencePayId        string                     `json:"referencePayId,omitempty"` // only applicable for PFConnect Authorization and Subsequent card installment payments.
	PaymentMethods        *NexiPaymentMethodsRequest `json:"paymentMethods,omitempty"`
}

type NexiAmount struct {
	Value               int64  `json:"value"`    // required, smallest currency unit
	Currency            string `json:"currency"` // required
	TaxTotal            *int64 `json:"taxTotal,omitempty"`
	NetItemTotal        *int64 `json:"netItemTotal,omitempty"`
	NetShippingAmount   *int64 `json:"netShippingAmount,omitempty"`
	GrossShippingAmount *int64 `json:"grossShippingAmount,omitempty"`
	NetDiscount         *int64 `json:"netDiscount,omitempty"`
	GrossDiscount       *int64 `json:"grossDiscount,omitempty"`
}

type NexiCaptureMethod struct {
	Type    string                    `json:"type"`              // AUTOMATIC | MANUAL | DELAYED
	Delayed *NexiDelayedCaptureMethod `json:"delayed,omitempty"` // required if Type == DELAYED
}

type NexiDelayedCaptureMethod struct {
	DelayedHours int `json:"delayedHours"` // 1–696
}

type NexiTemplate struct {
	Name         string            `json:"name,omitempty"`
	CustomFields *NexiCustomFields `json:"customFields,omitempty"`
}

type NexiCustomFields struct {
	CustomField1  string `json:"customField1,omitempty"`
	CustomField2  string `json:"customField2,omitempty"`
	CustomField3  string `json:"customField3,omitempty"`
	CustomField4  string `json:"customField4,omitempty"`
	CustomField5  string `json:"customField5,omitempty"`
	CustomField6  string `json:"customField6,omitempty"`
	CustomField7  string `json:"customField7,omitempty"`
	CustomField8  string `json:"customField8,omitempty"`
	CustomField9  string `json:"customField9,omitempty"`
	CustomField10 string `json:"customField10,omitempty"`
	CustomField11 string `json:"customField11,omitempty"`
	CustomField12 string `json:"customField12,omitempty"`
	CustomField13 string `json:"customField13,omitempty"`
	CustomField14 string `json:"customField14,omitempty"`
}

type NexiOrder struct {
	MerchantReference string          `json:"merchantReference,omitempty"`
	NumberOfArticles  int64           `json:"numberOfArticles,omitempty"`
	CreationDate      string          `json:"creationDate,omitempty"` // RFC3339 date-time
	InvoiceId         string          `json:"invoiceId,omitempty"`
	Items             []NexiOrderItem `json:"items,omitempty"`
}

type NexiOrderItem struct {
	Id                      string           `json:"id,omitempty"`
	SKU                     string           `json:"sku,omitempty"`
	Name                    string           `json:"name,omitempty"`
	Type                    string           `json:"type,omitempty"` // e.g.  "physical", "digital", "discount", "sales_tax", "shipping_fee", "discount", "gift_card", "store_credit"
	Quantity                int64            `json:"quantity,omitempty"`
	QuantityUnit            string           `json:"quantityUnit,omitempty"` // e.g. "pcs", "kg", or "liters"
	TaxRate                 int64            `json:"taxRate,omitempty"`      // e.g. 25% → 2500
	NetPrice                int64            `json:"netPrice,omitempty"`
	GrossPrice              int64            `json:"grossPrice,omitempty"`
	DiscountAmount          int64            `json:"discountAmount,omitempty"`
	TaxAmount               int64            `json:"taxAmount,omitempty"`
	MerchantData            string           `json:"merchantData,omitempty"`
	Description             string           `json:"description,omitempty"`
	MarketplaceSellerId     string           `json:"marketplaceSellerId,omitempty"`
	LineNumber              int64            `json:"lineNumber,omitempty"`
	MerchantProductType     string           `json:"merchantProductType,omitempty"`
	GoogleProductCategory   string           `json:"googleProductCategory,omitempty"`
	GoogleProductCategoryId int64            `json:"googleProductCategoryId,omitempty"`
	GroupId                 string           `json:"groupId,omitempty"`
	TaxCategory             string           `json:"taxCategory,omitempty"`
	AdditionalDescription   string           `json:"additionalDescription,omitempty"`
	ProductInfo             *NexiProductInfo `json:"productInfo,omitempty"`
}

type NexiProductInfo struct {
	Brand                  string   `json:"brand,omitempty"`
	Categories             []string `json:"categories,omitempty"`
	GlobalTradeItemNumber  string   `json:"globalTradeItemNumber,omitempty"` // EAN / ISBN / UPC
	ManufacturerPartNumber string   `json:"manufacturerPartNumber,omitempty"`
	ImageURL               string   `json:"imageUrl,omitempty"`
	ProductURL             string   `json:"productUrl,omitempty"`
}

type NexiPaymentUrlsRequest struct {
	Return      string `json:"return,omitempty"`      // required if Hosted Payment Page
	Cancel      string `json:"cancel,omitempty"`      // required if Hosted Payment Page
	Webhook     string `json:"webhook,omitempty"`     // required if Hosted Payment Page
	AppRedirect string `json:"appRedirect,omitempty"` // The URL used in Instanea payments to provide the merchant’s base app URL for redirection.
}

type NexiPaymentUrlsResponse struct {
	Return  string `json:"return,omitempty"`  // required if Hosted Payment Page
	Cancel  string `json:"cancel,omitempty"`  // required if Hosted Payment Page
	Webhook string `json:"webhook,omitempty"` // required if Hosted Payment Page
}

type NexiBillingAddress struct {
	StreetName   string `json:"streetName,omitempty"`
	StreetNumber string `json:"streetNumber,omitempty"`
	AddressLine2 string `json:"addressLine2,omitempty"`
	AddressLine3 string `json:"addressLine3,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Country      string `json:"country,omitempty"` // ISO 3166-1 alpha-3
	PostalCode   string `json:"postalCode,omitempty"`
}

type NexiShipping struct {
	Type    string              `json:"type,omitempty"` // shipping method
	Address NexiShippingAddress `json:"address,omitempty"`
}

type NexiShippingAddress struct {
	FirstName    string     `json:"firstName,omitempty"`
	LastName     string     `json:"lastName,omitempty"`
	CompanyName  string     `json:"companyName,omitempty"`
	StreetName   string     `json:"streetName,omitempty"`
	StreetNumber string     `json:"streetNumber,omitempty"`
	AddressLine2 string     `json:"addressLine2,omitempty"`
	AddressLine3 string     `json:"addressLine3,omitempty"`
	City         string     `json:"city,omitempty"`
	State        string     `json:"state,omitempty"`
	Country      string     `json:"country,omitempty"` // ISO 3166-1 alpha-3
	PostalCode   string     `json:"postalCode,omitempty"`
	Phone        *NexiPhone `json:"phone,omitempty"`
}

type NexiPhone struct {
	CountryCode string `json:"countryCode"`
	Number      string `json:"number"`
}

type NexiCustomerInfoRequest struct {
	MerchantCustomerId        string     `json:"merchantCustomerId,omitempty"`
	CustomerType              string     `json:"customerType,omitempty"` // individual | business
	FirstName                 string     `json:"firstName,omitempty"`
	LastName                  string     `json:"lastName,omitempty"`
	Email                     string     `json:"email"` // required
	Phone                     *NexiPhone `json:"phone,omitempty"`
	Salutation                string     `json:"salutation,omitempty"`
	Title                     string     `json:"title,omitempty"`
	Gender                    string     `json:"gender,omitempty"`
	MaidenName                string     `json:"maidenName,omitempty"`
	MiddleName                string     `json:"middleName,omitempty"`
	BirthDate                 string     `json:"birthDate,omitempty"` // YYYY-MM-DD
	BirthPlace                string     `json:"birthPlace,omitempty"`
	SocialSecurityNumber      string     `json:"socialSecurityNumber,omitempty"`
	TaxId                     string     `json:"taxId,omitempty"`
	CompanyName               string     `json:"companyName,omitempty"`
	PositionOccupied          string     `json:"positionOccupied,omitempty"`
	CompanyRegistrationNumber string     `json:"companyRegistrationNumber,omitempty"`
	CompanyVatId              string     `json:"companyVatId,omitempty"`
	CompanyLegalForm          string     `json:"companyLegalForm,omitempty"`
}

type NexiCustomerInfoResponse struct {
	GatewayCustomerId         string     `json:"gatewayCustomerId,omitempty"`
	MerchantCustomerId        string     `json:"merchantCustomerId,omitempty"`
	CustomerType              string     `json:"customerType,omitempty"` // individual | business
	FirstName                 string     `json:"firstName,omitempty"`
	LastName                  string     `json:"lastName,omitempty"`
	Email                     string     `json:"email"` // required
	Phone                     *NexiPhone `json:"phone,omitempty"`
	Salutation                string     `json:"salutation,omitempty"`
	Title                     string     `json:"title,omitempty"`
	Gender                    string     `json:"gender,omitempty"`
	MaidenName                string     `json:"maidenName,omitempty"`
	MiddleName                string     `json:"middleName,omitempty"`
	BirthDate                 string     `json:"birthDate,omitempty"` // YYYY-MM-DD
	BirthPlace                string     `json:"birthPlace,omitempty"`
	SocialSecurityNumber      string     `json:"socialSecurityNumber,omitempty"`
	TaxId                     string     `json:"taxId,omitempty"`
	CompanyName               string     `json:"companyName,omitempty"`
	PositionOccupied          string     `json:"positionOccupied,omitempty"`
	CompanyRegistrationNumber string     `json:"companyRegistrationNumber,omitempty"`
	CompanyVatId              string     `json:"companyVatId,omitempty"`
	CompanyLegalForm          string     `json:"companyLegalForm,omitempty"`
}

type NexiFraudData struct {
	AccountInfo               *NexiFraudAccountInfo `json:"accountInfo,omitempty"`
	DeliveryEmail             string                `json:"deliveryEmail,omitempty"`
	DeliveryTimeframe         string                `json:"deliveryTimeframe,omitempty"`
	GiftCardAmount            *int64                `json:"giftCardAmount,omitempty"` // smallest currency unit
	GiftCardCount             *int64                `json:"giftCardCount,omitempty"`
	GiftCardCurrency          *int                  `json:"giftCardCurrency,omitempty"` // ISO 4217 numeric
	PreOrderDate              string                `json:"preOrderDate,omitempty"`     // YYYY-MM-DD
	PreOrderPurchaseIndicator *bool                 `json:"preOrderPurchaseIndicator,omitempty"`
	ReorderItemsIndicator     *bool                 `json:"reorderItemsIndicator,omitempty"`
	ShippingAddressIndicator  string                `json:"shippingAddressIndicator,omitempty"`
	IPZone                    string                `json:"ipZone,omitempty"` // comma-separated numeric ISO 3166-1
	Zone                      string                `json:"zone,omitempty"`   // comma-separated numeric/alphanumeric ISO 3166-1
}

type NexiFraudAccountInfo struct {
	Id                          string                       `json:"id,omitempty"`
	AuthenticationInfo          *NexiFraudAuthenticationInfo `json:"authenticationInfo,omitempty"`
	AgeIndicator                string                       `json:"ageIndicator,omitempty"`                // GUEST_CHECKOUT | THIS_TRANSACTION | LESS_THAN_30_DAYS | FROM_30_TO_60_DAYS | MORE_THAN_60_DAYS
	ChangeDate                  string                       `json:"changeDate,omitempty"`                  // YYYY-MM-DD
	ChangeIndicator             string                       `json:"changeIndicator,omitempty"`             // THIS_TRANSACTION | LESS_THAN_30_DAYS | FROM_30_TO_60_DAYS | MORE_THAN_60_DAYS
	CreationDate                string                       `json:"creationDate,omitempty"`                // YYYY-MM-DD
	PasswordChangeDate          string                       `json:"passwordChangeDate,omitempty"`          // YYYY-MM-DD
	PasswordChangeDateIndicator string                       `json:"passwordChangeDateIndicator,omitempty"` // NO_CHANGE | THIS_TRANSACTION | LESS_THAN_30_DAYS | FROM_30_TO_60_DAYS | MORE_THAN_60_DAYS
	NumberOfPurchases           *int64                       `json:"numberOfPurchases,omitempty"`
	AddCardAttemptsDay          *int64                       `json:"addCardAttemptsDay,omitempty"`
	NumberTransactionsDay       *int64                       `json:"numberTransactionsDay,omitempty"`
	NumberTransactionsYear      *int64                       `json:"numberTransactionsYear,omitempty"`
	PaymentAccountAge           string                       `json:"paymentAccountAge,omitempty"`          // YYYY-MM-DD
	PaymentAccountAgeIndicator  string                       `json:"paymentAccountAgeIndicator,omitempty"` // GUEST_CHECKOUT | THIS_TRANSACTION | LESS_THAN_30_DAYS | FROM_30_TO_60_DAYS | MORE_THAN_60_DAYS
	ShipAddressUsageDate        string                       `json:"shipAddressUsageDate,omitempty"`       // YYYY-MM-DD
	ShipAddressUsageIndicator   string                       `json:"shipAddressUsageIndicator,omitempty"`  // THIS_TRANSACTION | LESS_THAN_30_DAYS | FROM_30_TO_60_DAYS | MORE_THAN_60_DAYS
	SuspiciousAccountActivity   *bool                        `json:"suspiciousAccountActivity,omitempty"`
}

type NexiFraudAuthenticationInfo struct {
	Method    string `json:"method,omitempty"`    // GUEST | MERCHANT_CREDENTIALS | FEDERATED_Id | ISSUER_CREDENTIALS | THIRD_PARTY_AUTHENTICATION | FIdO | SIGNED_FIdO | SRC_ASSURANCE_DATA
	Timestamp string `json:"timestamp,omitempty"` // RFC3339 date-time YYYY-MM-DDTHH:MM:SS+00:00
	Data      string `json:"data,omitempty"`      // optional FIdO / attestation data
}

type NexiPaymentFacilitator struct {
	SubMerchantId      string `json:"subMerchantId,omitempty"`
	SubMerchantName    string `json:"subMerchantName,omitempty"`
	SubMerchantCity    string `json:"subMerchantCity,omitempty"`
	SubMerchantCountry string `json:"subMerchantCountry,omitempty"` // ISO 3166-1 alpha-2
	SubMerchantStreet  string `json:"subMerchantStreet,omitempty"`
	SubMerchantZip     string `json:"subMerchantZip,omitempty"`
	SubMerchantState   string `json:"subMerchantState,omitempty"`
}

type NexiBrowserInfo struct {
	AcceptHeaders     string  `json:"acceptHeaders,omitempty"`
	IPAddress         string  `json:"ipAddress,omitempty"`
	JavaEnabled       *bool   `json:"javaEnabled,omitempty"`
	JavaScriptEnabled bool    `json:"javaScriptEnabled"`
	Language          string  `json:"language,omitempty"` // IETF BCP47 e.g. "en", "de-DE"
	ColorDepth        *int    `json:"colorDepth,omitempty"`
	ScreenHeight      *int    `json:"screenHeight,omitempty"`
	ScreenWidth       *int    `json:"screenWidth,omitempty"`
	TimeZoneOffset    *string `json:"timeZoneOffset,omitempty"` // Minutes offset from UTC
	UserAgent         string  `json:"userAgent,omitempty"`
}

type NexiDevice struct {
	DeviceId             string `json:"deviceId,omitempty"`   // Unique device identifier
	DeviceType           string `json:"deviceType,omitempty"` // DESKTOP | TABLET | SMARTPHONE
	DeviceOS             string `json:"deviceOs,omitempty"`   // ANDROId | IOS | OTHERS
	Confidence           *int   `json:"confidence,omitempty"` // 0-100
	NewDevice            *bool  `json:"newDevice,omitempty"`
	IsAnonymousProxyUsed *bool  `json:"isAnonymousProxyUsed,omitempty"`
	IsProxyUsed          *bool  `json:"isProxyUsed,omitempty"`

	Country   string `json:"country,omitempty"`
	City      string `json:"city,omitempty"`
	Latitude  string `json:"latitude,omitempty"`
	Longitude string `json:"longitude,omitempty"`

	FraudScore       *int     `json:"fraudScore,omitempty"`
	FraudScoreRules  []string `json:"fraudScoreRules,omitempty"`
	BrowserLanguages []string `json:"browserLanguages,omitempty"`
	IsMobileDevice   *bool    `json:"isMobileDevice,omitempty"`
	Fraud            string   `json:"fraud,omitempty"` // e.g. fraud | suspicion | nofraud
}

type NexiChannel string

const (
	ChannelECOM      NexiChannel = "ECOM"
	ChannelMOTO      NexiChannel = "MOTO"
	ChannelAPP       NexiChannel = "APP"
	ChannelPayByLink NexiChannel = "PAYBYLINK"
	ChannelPOS       NexiChannel = "POS"
)

type IntegrationType string

const (
	IntegrationHosted IntegrationType = "HOSTED"
	IntegrationDirect IntegrationType = "DIRECT"
)

type PaymentMethodType string

const (
	PaymentApplePay    PaymentMethodType = "APPLEPAY"
	PaymentBancontact  PaymentMethodType = "BANCONTACT"
	PaymentBoleto      PaymentMethodType = "BOLETO"
	PaymentCard        PaymentMethodType = "CARD"
	PaymentDirectDebit PaymentMethodType = "DIRECTDEBIT"
	PaymentEasyCollect PaymentMethodType = "EASYCOLLECT"
	PaymentEPS         PaymentMethodType = "EPS"
	PaymentGooglePay   PaymentMethodType = "GOOGLEPAY"
	PaymentiDEAL       PaymentMethodType = "IdEAL"
	PaymentInstanea    PaymentMethodType = "INSTANEA"
	PaymentKlarna      PaymentMethodType = "KLARNA"
	PaymentMultiBanco  PaymentMethodType = "MULTIBANCO"
	PaymentMyBank      PaymentMethodType = "MYBANK"
	PaymentPayPal      PaymentMethodType = "PAYPAL"
	PaymentPrzelewy24  PaymentMethodType = "PRZELEWY24"
	PaymentTrustly     PaymentMethodType = "TRUSTLY"
	PaymentTwint       PaymentMethodType = "TWINT"
	PaymentVipps       PaymentMethodType = "VIPPS"
	PaymentWero        PaymentMethodType = "WERO"
)

type NexiPaymentMethodsRequest struct {
	IntegrationType IntegrationType   `json:"integrationType"`
	Type            PaymentMethodType `json:"type"`
	Card            *NexiCard         `json:"card,omitempty"`
}

type CardEventToken string

const (
	CardEventDelayedShipment     CardEventToken = "DELAYED_SHIPMENT"
	CardEventPreAuth             CardEventToken = "PRE_AUTH"
	CardEventPLBS                CardEventToken = "PLBS"
	CardEventOrder               CardEventToken = "ORDER"
	CardEventAccountVerification CardEventToken = "ACCOUNT_VERIFICATION"
)

type ThreeDsPolicySkip string

const (
	ThreeDsSkipThisTransaction ThreeDsPolicySkip = "THIS_TRANSACTION"
	ThreeDsSkipOutOfScope      ThreeDsPolicySkip = "OUT_OF_SCOPE"
	ThreeDsSkipDataOnly        ThreeDsPolicySkip = "DATA_ONLY"
)

type ThreeDsExemptionReason string

const (
	ExemptionTransactionRiskAnalysis ThreeDsExemptionReason = "TRANSACTION_RISK_ANALYSIS"
	ExemptionDelegateAuthority       ThreeDsExemptionReason = "DELEGATE_AUTHORITY"
	ExemptionLowValue                ThreeDsExemptionReason = "LOW_VALUE"
	ExemptionTrustedBeneficiary      ThreeDsExemptionReason = "TRUSTED_BENEFICIARY"
	ExemptionSecureCorporatePayment  ThreeDsExemptionReason = "SECURE_CORPORATE_PAYMENT"
)

type ThreeDsChallengePreference string

const (
	ChallengeNoPreference     ThreeDsChallengePreference = "NO_PREFERENCE"
	ChallengeNoChallenge      ThreeDsChallengePreference = "NO_CHALLENGE"
	ChallengeRequestChallenge ThreeDsChallengePreference = "REQUEST_CHALLENGE"
	ChallengeMandateChallenge ThreeDsChallengePreference = "MANDATE_CHALLENGE"
)

type NexiCardThreeDsPolicyExemption struct {
	Reason            ThreeDsExemptionReason `json:"reason,omitempty"`
	MerchantFraudRate int                    `json:"merchantFraudRate,omitempty"`
}

type NexiCardThreeDsPolicy struct {
	Skip                ThreeDsPolicySkip               `json:"skip,omitempty"`
	Exemption           *NexiCardThreeDsPolicyExemption `json:"exemption,omitempty"`
	ChallengePreference ThreeDsChallengePreference      `json:"challengePreference,omitempty"`
}

type NexiCardTemplateCustomFields struct {
	CustomField1  string `json:"customField1,omitempty"`
	CustomField2  string `json:"customField2,omitempty"`
	CustomField3  string `json:"customField3,omitempty"`
	CustomField4  string `json:"customField4,omitempty"`
	CustomField5  string `json:"customField5,omitempty"`
	CustomField6  string `json:"customField6,omitempty"`
	CustomField7  string `json:"customField7,omitempty"`
	CustomField8  string `json:"customField8,omitempty"`
	CustomField9  string `json:"customField9,omitempty"`
	CustomField10 string `json:"customField10,omitempty"`
	CustomField11 string `json:"customField11,omitempty"`
	CustomField12 string `json:"customField12,omitempty"`
	CustomField13 string `json:"customField13,omitempty"`
	CustomField14 string `json:"customField14,omitempty"`
}

type NexiCardTemplate struct {
	Name            string                        `json:"name,omitempty"`
	BackgroundColor string                        `json:"backgroundColor,omitempty"`
	BackgroundImage string                        `json:"backgroundImage,omitempty"`
	TextColor       string                        `json:"textColor,omitempty"`
	FontName        string                        `json:"fontName,omitempty"`
	FontSize        int                           `json:"fontSize,omitempty"`
	TableWidth      int                           `json:"tableWidth,omitempty"`
	TableHeight     int                           `json:"tableHeight,omitempty"`
	CustomFields    *NexiCardTemplateCustomFields `json:"customFields,omitempty"`
}

type NexiCardPrefillInfo struct {
	Number         string `json:"number,omitempty"`
	CardholderName string `json:"cardholderName,omitempty"`
	ExpiryDate     string `json:"expiryDate,omitempty"` // YYYYMM
	Brand          string `json:"brand,omitempty"`
	SecurityCode   string `json:"securityCode,omitempty"`
}

type NexiCard struct {
	EventToken    CardEventToken         `json:"eventToken,omitempty"`
	SubType       []string               `json:"subType,omitempty"` // list of allowed card networks - overrides brand selection in merchant account
	ThreeDsPolicy *NexiCardThreeDsPolicy `json:"threeDsPolicy,omitempty"`
	Template      *NexiCardTemplate      `json:"template,omitempty"`
	PrefillInfo   *NexiCardPrefillInfo   `json:"prefillInfo,omitempty"`
}

// --- NexiCreateCheckoutSessionResponse

type NexiCreateCheckoutSessionResponse struct {
	Links NexiCheckoutSessionResponseLinks `json:"_links,omitempty"`
}

type NexiCheckoutSessionResponseLinks struct {
	Redirect *RedirectLink `json:"redirect,omitempty"`
}

type RedirectLink struct {
	Href string `json:"href,omitempty"`
	Type string `json:"type,omitempty"`
}

// --- NexiPaymentQueryResponse

type ApplePayResponse struct {
	SchemeReferenceId string `json:"schemeReferenceId,omitempty"`
}

type CardResponse struct {
	CardHolderName          string                      `json:"cardHolderName,omitempty"`
	PseudoCardNumber        string                      `json:"pseudoCardNumber,omitempty"`
	First6Digits            string                      `json:"first6Digits,omitempty"`
	Last4Digits             string                      `json:"last4Digits,omitempty"`
	ExpiryDate              string                      `json:"expiryDate,omitempty"`
	Brand                   string                      `json:"brand,omitempty"`
	Product                 string                      `json:"product,omitempty"`
	Source                  string                      `json:"source,omitempty"`
	Type                    string                      `json:"type,omitempty"`
	Issuer                  string                      `json:"issuer,omitempty"`
	Country                 string                      `json:"country,omitempty"`
	Bin                     *CardBINResponse            `json:"bin,omitempty"`
	VersioningData          *CardVersioningDataResponse `json:"versioningData,omitempty"`
	FraudData               *CardFraudDataResponse      `json:"fraudData,omitempty"`
	AuthenticationData      *CardAuthenticationResponse `json:"authenticationData,omitempty"`
	SchemeReferenceId       string                      `json:"schemeReferenceId,omitempty"`
	ProviderApprovalCode    string                      `json:"providerApprovalCode,omitempty"`
	ProviderResponseCode    string                      `json:"providerResponseCode,omitempty"`
	ProviderResponseMessage string                      `json:"providerResponseMessage,omitempty"`
	ProviderTransactionId   string                      `json:"providerTransactionId,omitempty"`
	ProviderToken           string                      `json:"providerToken,omitempty"`
	ProviderMerchantId      string                      `json:"providerMerchantId,omitempty"`
	ProviderTerminalId      string                      `json:"providerTerminalId,omitempty"`
	ProviderOrderId         string                      `json:"providerOrderId,omitempty"`
	IssuerResponseCode      string                      `json:"issuerResponseCode,omitempty"`
	IssuerResponseMessage   string                      `json:"issuerResponseMessage,omitempty"`
}

type CardBINResponse struct {
	AccountBin       string `json:"accountBin,omitempty"`
	AccountRangeLow  string `json:"accountRangeLow,omitempty"`
	AccountRangeHigh string `json:"accountRangeHigh,omitempty"`
}

type CardVersioningDataResponse struct {
	ThreeDSServerTransId    string                       `json:"threeDSServerTransId,omitempty"`
	AcsStartProtocolVersion string                       `json:"acsStartProtocolVersion,omitempty"`
	AcsEndProtocolVersion   string                       `json:"acsEndProtocolVersion,omitempty"`
	DsStartProtocolVersion  string                       `json:"dsStartProtocolVersion,omitempty"`
	DsEndProtocolVersion    string                       `json:"dsEndProtocolVersion,omitempty"`
	ThreeDSMethodDataForm   string                       `json:"threeDSMethodDataForm,omitempty"`
	ThreeDSMethodURL        string                       `json:"threeDSMethodUrl,omitempty"`
	ThreeDSMethodData       *ThreeDSMethodDataResponse   `json:"threeDSMethodData,omitempty"`
	ErrorDetails            *ThreeDSErrorDetailsResponse `json:"errorDetails,omitempty"`
}

type ThreeDSMethodDataResponse struct {
	ThreeDSMethodNotificationURL string `json:"threeDSMethodNotificationUrl,omitempty"`
	ThreeDSServerTransId         string `json:"threeDSServerTransId,omitempty"`
}

type ThreeDSErrorDetailsResponse struct {
	ThreeDSServerTransId string `json:"threeDSServerTransId,omitempty"`
	ErrorCode            string `json:"errorCode,omitempty"`
	ErrorComponent       string `json:"errorComponent,omitempty"`
	ErrorDescription     string `json:"errorDescription,omitempty"`
}

type CardFraudDataResponse struct {
	Zone        string `json:"zone,omitempty"`
	IPZone      string `json:"ipZone,omitempty"`
	IPZoneA2    string `json:"ipZoneA2,omitempty"`
	IPState     string `json:"ipState,omitempty"`
	IPCity      string `json:"ipCity,omitempty"`
	IPLongitude string `json:"ipLongitude,omitempty"`
	IPLatitude  string `json:"ipLatitude,omitempty"`
	FSStatus    string `json:"fsStatus,omitempty"`
	FSCode      string `json:"fsCode,omitempty"`
}

type CardAuthenticationResponse struct {
	ThreeDSServerTransId  string                    `json:"threeDSServerTransId,omitempty"`
	AcsChallengeMandated  bool                      `json:"acsChallengeMandated,omitempty"`
	AcsDecConInd          bool                      `json:"acsDecConInd,omitempty"`
	AcsOperatorId         string                    `json:"acsOperatorId,omitempty"`
	AcsReferenceNumber    string                    `json:"acsReferenceNumber,omitempty"`
	AcsSignedContent      string                    `json:"acsSignedContent,omitempty"`
	AcsTransId            string                    `json:"acsTransId,omitempty"`
	AcsURL                string                    `json:"acsURL,omitempty"`
	AuthenticationType    string                    `json:"authenticationType,omitempty"`
	AuthenticationValue   string                    `json:"authenticationValue,omitempty"`
	BroadInfo             string                    `json:"broadInfo,omitempty"`
	CardholderInfo        string                    `json:"cardholderInfo,omitempty"`
	DsReferenceNumber     string                    `json:"dsReferenceNumber,omitempty"`
	DsTransId             string                    `json:"dsTransId,omitempty"`
	ECI                   string                    `json:"eci,omitempty"`
	MessageExtension      string                    `json:"messageExtension,omitempty"`
	MessageType           string                    `json:"messageType,omitempty"`
	MessageVersion        string                    `json:"messageVersion,omitempty"`
	SDKTransId            string                    `json:"sdkTransId,omitempty"`
	TransStatus           string                    `json:"transStatus,omitempty"`
	TransStatusReason     string                    `json:"transStatusReason,omitempty"`
	WhiteListStatus       string                    `json:"whiteListStatus,omitempty"`
	WhiteListStatusSource string                    `json:"whiteListStatusSource,omitempty"`
	ChallengeRequest      *ChallengeRequestResponse `json:"challengeRequest,omitempty"`
}

type ChallengeRequestResponse struct {
	ThreeDSServerTransId          string `json:"threeDSServerTransId,omitempty"`
	AcsTransId                    string `json:"acsTransId,omitempty"`
	ChallengeWindowSize           string `json:"challengeWindowSize,omitempty"`
	MessageVersion                string `json:"messageVersion,omitempty"`
	MessageType                   string `json:"messageType,omitempty"`
	Base64EncodedChallengeRequest string `json:"base64EncodedChallengeRequest,omitempty"`
	ThreeDSCompInd                string `json:"threeDSCompInd,omitempty"`
}

type GooglePayResponse struct {
	SchemeReferenceId string `json:"schemeReferenceId,omitempty"`
}

type NexiPaymentMethodsResponse struct {
	Type string `json:"type,omitempty"`

	ApplePay  *ApplePayResponse  `json:"applePay,omitempty"`
	Card      *CardResponse      `json:"card,omitempty"`
	GooglePay *GooglePayResponse `json:"googlePay,omitempty"`
}

type NexiAmountResponse struct {
	Value               int64  `json:"value"`    // required, smallest currency unit
	Currency            string `json:"currency"` // required
	TaxTotal            *int64 `json:"taxTotal,omitempty"`
	NetItemTotal        *int64 `json:"netItemTotal,omitempty"`
	NetShippingAmount   *int64 `json:"netShippingAmount,omitempty"`
	GrossShippingAmount *int64 `json:"grossShippingAmount,omitempty"`
	NetDiscount         *int64 `json:"netDiscount,omitempty"`
	GrossDiscount       *int64 `json:"grossDiscount,omitempty"`
	CapturedValue       *int64 `json:"capturedValue,omitempty"`
	RefundedValue       *int64 `json:"refundedValue,omitempty"`
}

type NexiPaymentQueryResponse struct {
	PayId                 string                      `json:"payId,omitempty"`
	TransId               string                      `json:"transId,omitempty"`
	ExternalIntegrationId string                      `json:"externalIntegrationId,omitempty"`
	RefNr                 string                      `json:"refNr,omitempty"`
	XId                   string                      `json:"xId,omitempty"`
	Status                string                      `json:"status,omitempty"`
	ResponseCode          string                      `json:"responseCode,omitempty"`
	ResponseDescription   string                      `json:"responseDescription,omitempty"`
	Metadata              map[string]any              `json:"metadata,omitempty"`
	Amount                *NexiAmountResponse         `json:"amount,omitempty"`
	Language              string                      `json:"language,omitempty"`
	CaptureMethod         *NexiCaptureMethod          `json:"captureMethod,omitempty"`
	Order                 *NexiOrder                  `json:"order,omitempty"`
	SimulationMode        string                      `json:"simulationMode,omitempty"`
	URLs                  *NexiPaymentUrlsResponse    `json:"urls,omitempty"`
	BillingAddress        *NexiBillingAddress         `json:"billingAddress,omitempty"`
	Shipping              *NexiShipping               `json:"shipping,omitempty"`
	StatementDescriptor   string                      `json:"statementDescriptor,omitempty"`
	CustomerInfo          *NexiCustomerInfoResponse   `json:"customerInfo,omitempty"`
	ExpirationTime        string                      `json:"expirationTime,omitempty"`
	FraudData             *NexiFraudData              `json:"fraudData,omitempty"`
	PaymentFacilitator    *NexiPaymentFacilitator     `json:"paymentFacilitator,omitempty"`
	BrowserInfo           *NexiBrowserInfo            `json:"browserInfo,omitempty"`
	Device                *NexiDevice                 `json:"device,omitempty"`
	Channel               string                      `json:"channel,omitempty"`
	PaymentMethods        *NexiPaymentMethodsResponse `json:"paymentMethods,omitempty"`
	MerchantId            string                      `json:"merchantId,omitempty"`
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
	Id          int64   `json:"id"`
	UUId        string  `json:"uuid"` // sent as merchantOrderId
	Amount      int64   `json:"amount"`
	Status      string  `json:"status"` // react to declined, confirmed, authorized, what else?
	Time        string  `json:"time"`   // take effective date from first 10 chars (ISO Date)
	Lang        string  `json:"lang"`   // ISO 639-1 of shopper language (de, en)
	PageUUId    string  `json:"pageUuid"`
	Payment     Payment `json:"payment"`
	Psp         string  `json:"psp"`   // Name of the payment service provider used, for example "ConCardis_PayEngine_3"
	PspId       int64   `json:"pspId"` // Id of the Psp
	Mode        string  `json:"mode"`  // "LIVE", "TEST"
	ReferenceId string  `json:"referenceId"`
	Invoice     Invoice `json:"invoice"`
}

type Payment struct {
	Brand string `json:"brand"`
}

type Invoice struct {
	ReferenceId      string `json:"referenceId"`
	PaymentRequestId uint   `json:"paymentRequestId"` // the payment link id
	Currency         string `json:"currency"`         // "EUR"
	OriginalAmount   int64  `json:"originalAmount"`
	RefundedAmount   int64  `json:"refundedAmount"`
}
