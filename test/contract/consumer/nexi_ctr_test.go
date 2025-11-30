package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	auzerolog "github.com/StephanHCB/go-autumn-logging-zerolog"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestverifier "github.com/StephanHCB/go-autumn-restclient/implementation/verifier"
	"github.com/eurofurence/reg-payment-nexi-adapter/docs"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/entity"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database/inmemorydb"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/nexi"
	"github.com/stretchr/testify/require"
)

func TestNexiApiClient(t *testing.T) {
	auzerolog.SetupPlaintextLogging()

	db := inmemorydb.Create()
	database.SetRepository(db)

	docs.Given("given the nexi adapter is correctly configured (not in local mock mode)")
	// set up basic configuration
	config.LoadTestingConfigurationFromPathOrAbort("../../resources/testconfig.yaml")

	// prepare dtos

	// STEP 1: create
	createRequest := nexi.NexiCreatePaymentRequest{
		Order: nexi.NexiOrder{
			Items: []nexi.NexiOrderItem{
				{
					Reference:        "EF 2022 REG 000004",
					Name:             "Convention Registration",
					Quantity:         1.0,
					Unit:             "qty",
					UnitPrice:        10550,
					TaxRate:          1900,
					TaxAmount:        0,
					GrossTotalAmount: 10550,
					NetTotalAmount:   10550,
					ImageUrl:         nil,
				},
			},
			Amount:    10550,
			Currency:  "EUR",
			Reference: "220118-150405-000004",
		},
		Checkout: nexi.NexiCheckout{
			Url:             nil,
			IntegrationType: "HostedPaymentPage",
			ReturnUrl:       "https://example.com/success",
			CancelUrl:       "https://example.com/failure",
			//Consumer: &nexi.NexiConsumer{
			//	Reference: "test@example.com",
			//	Email:     "test@example.com",
			//},
			TermsUrl: "https://help.eurofurence.org/legal/terms",
			//ShippingCountries: []nexi.NexiCountry{
			//	{CountryCode: "DEU"},
			//},
			//Shipping: &nexi.NexiShipping{
			//	Countries: []nexi.NexiCountry{
			//		{CountryCode: "DEU"},
			//	},
			//	MerchantHandlesShippingCost: false,
			//	EnableBillingAddress:        true,
			//},
			//ConsumerType: &nexi.NexiConsumerType{
			//	Default:        "b2c",
			//	SupportedTypes: []string{"b2c", "b2b"},
			//},
			Charge:                      false,
			PublicDevice:                false,
			MerchantHandlesConsumerData: true,
			CountryCode:                 p("DEU"),
			Appearance: &nexi.NexiAppearance{
				DisplayOptions: nexi.NexiDisplayOptions{
					ShowMerchantName: true,
					ShowOrderSummary: true,
				},
				TextOptions: nexi.NexiTextOptions{
					CompletePaymentButtonText: "pay",
				},
			},
		},
		Notifications: &nexi.NexiNotifications{
			Webhooks: []nexi.NexiWebhook{
				{
					EventName:     "payment.created",
					Url:           "http://localhost:8080/api/rest/v1/webhook/1234",
					Authorization: "",
				},
			},
		},
	}

	ctx := auzerolog.AddLoggerToCtx(context.Background())

	// set a server url so local simulator mode is off
	config.Configuration().Service.NexiDownstream = "http://localhost:8000"

	docs.When("when requests to create, then read, then delete a paylink are made")
	docs.Then("then all three requests are successful")

	// Set up our expected interactions.
	verifierClient, verifierImpl := aurestverifier.New()
	verifierImpl.AddExpectation(aurestverifier.Request{
		Name:   "create-paylink",
		Method: http.MethodPost,
		Header: http.Header{ // not verified
			"Content-Type": []string{"application/json"},
		},
		Url:  "http://localhost:8000/v1/payments",
		Body: `{"order":{"items":[{"reference":"EF 2022 REG 000004","name":"Convention Registration","quantity":1,"unit":"qty","unitPrice":10550,"taxRate":1900,"taxAmount":0,"grossTotalAmount":10550,"netTotalAmount":10550}],"amount":10550,"currency":"EUR","reference":"220118-150405-000004"},"checkout":{"integrationType":"HostedPaymentPage","returnUrl":"https://example.com/success","cancelUrl":"https://example.com/failure","termsUrl":"https://help.eurofurence.org/legal/terms","charge":false,"publicDevice":false,"merchantHandlesConsumerData":true,"appearance":{"displayOptions":{"showMerchantName":true,"showOrderSummary":true},"textOptions":{"completePaymentButtonText":"pay"}},"countryCode":"DEU"},"notifications":{"webhooks":[{"eventName":"payment.created","url":"http://localhost:8080/api/rest/v1/webhook/1234","authorization":""}]}}`,
	}, aurestclientapi.ParsedResponse{
		Body: &nexi.NexiCreateLowlevelResponseBody{
			PaymentId:            "42",
			HostedPaymentPageUrl: "http://localhost/some/pay/link",
		},
		Status: http.StatusCreated,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Time: time.Time{},
	}, nil)
	verifierImpl.AddExpectation(aurestverifier.Request{
		Name:   "read-paylink-after-use",
		Method: http.MethodGet,
		Header: http.Header{}, // not verified
		Url:    "http://localhost:8000/v1/payments/42",
		Body:   "",
	}, aurestclientapi.ParsedResponse{
		Body: &nexi.NexiQueryLowlevelResponseBody{
			Payment: nexi.NexiPayment{
				PaymentId: "42",
				Summary: nexi.NexiSummary{
					ReservedAmount:           0,
					ReservedSurchargeAmount:  0,
					ChargedAmount:            10550,
					ChargedSurchargeAmount:   0,
					RefundedAmount:           0,
					RefundedSurchargeAmount:  0,
					CancelledAmount:          0,
					CancelledSurchargeAmount: 0,
				},
				Consumer: nexi.NexiConsumerFull{
					ShippingAddress: nexi.NexiAddressFull{
						AddressLine1: "",
						AddressLine2: "",
						ReceiverLine: "",
						PostalCode:   "",
						City:         "",
						Country:      "",
						PhoneNumber: nexi.NexiPhone{
							Prefix: "",
							Number: "",
						},
					},
					Company: nexi.NexiCompanyFull{
						MerchantReference:  "",
						Name:               "",
						RegistrationNumber: "",
						ContactDetails: nexi.NexiContactFull{
							FirstName: "",
							LastName:  "",
							Email:     "",
							PhoneNumber: nexi.NexiPhone{
								Prefix: "",
								Number: "",
							},
						},
					},
					PrivatePerson: nexi.NexiPrivatePersonFull{
						MerchantReference: "",
						DateOfBirth:       "",
						FirstName:         "",
						LastName:          "",
						Email:             "",
						PhoneNumber: nexi.NexiPhone{
							Prefix: "",
							Number: "",
						},
					},
					BillingAddress: nexi.NexiAddressFull{
						AddressLine1: "",
						AddressLine2: "",
						ReceiverLine: "",
						PostalCode:   "",
						City:         "",
						Country:      "",
						PhoneNumber: nexi.NexiPhone{
							Prefix: "",
							Number: "",
						},
					},
				},
				PaymentDetails: nexi.NexiPaymentDetails{
					PaymentType:   "card",
					PaymentMethod: "visa",
					InvoiceDetails: nexi.NexiInvoiceDetails{
						InvoiceNumber: "",
					},
					CardDetails: nexi.NexiCardDetails{
						MaskedPan:  "411111******1111",
						ExpiryDate: "12/25",
					},
				},
				OrderDetails: nexi.NexiOrderDetails{
					Amount:    10550,
					Currency:  "EUR",
					Reference: "220118-150405-000004",
				},
				Checkout: nexi.NexiCheckoutDetails{
					Url:       "http://localhost/some/pay/link",
					CancelUrl: "https://example.com/failure",
				},
				Created:    "2024-10-15T15:50:20Z",
				Refunds:    []nexi.NexiRefund{},
				Charges:    []nexi.NexiCharge{},
				Terminated: "",
			},
		},
		Status: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Time: time.Time{},
	}, nil)
	verifierImpl.AddExpectation(aurestverifier.Request{
		Name:   "delete-paylink",
		Method: http.MethodPost,
		Header: http.Header{}, // not verified
		Url:    "http://localhost:8000/v1/payments/42/cancels",
		Body:   `{"amount":10550}`,
	}, aurestclientapi.ParsedResponse{
		Status: http.StatusOK,
		Time:   time.Time{},
	}, nil)

	// set up downstream client
	client := nexi.NewTestingClient(verifierClient)

	// STEP 1: create a new payment link
	created, err := client.CreatePaymentLink(ctx, createRequest)
	require.Nil(t, err)
	require.Equal(t, "42", created.ID)
	require.Equal(t, "http://localhost/some/pay/link", created.Link)

	// STEP 2: read the payment link again after use
	read, err := client.QueryPaymentLink(ctx, created.ID)
	require.Nil(t, err)
	require.Equal(t, "42", read.ID)
	require.Equal(t, "220118-150405-000004", read.ReferenceID)
	require.Equal(t, "confirmed", read.Status)
	require.Equal(t, int32(10550), read.Amount)

	// STEP 3: delete the payment link (wouldn't normally work after use)
	err = client.DeletePaymentLink(ctx, created.ID, 10550)
	require.Nil(t, err)

	docs.Then("and the expected interactions have occurred in the correct order")
	require.Nil(t, verifierImpl.FirstUnexpectedOrNil())

	docs.Then("and the expected protocol entries have been written to the database")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "220118-150405-000004",
		Kind:        "raw",
		Message:     "nexi create request",
		Details:     `{"order":{"items":[{"reference":"EF 2022 REG 000004","name":"Convention Registration","quantity":1,"unit":"qty","unitPrice":10550,"taxRate":1900,"taxAmount":0,"grossTotalAmount":10550,"netTotalAmount":10550}],"amount":10550,"currency":"EUR","reference":"220118-150405-000004"},"checkout":{"integrationType":"HostedPaymentPage","returnUrl":"https://example.com/success","cancelUrl":"https://example.com/failure","termsUrl":"https://help.eurofurence.org/legal/terms","charge":false,"publicDevice":false,"merchantHandlesConsumerData":true,"appearance":{"displayOptions":{"showMerchantName":true,"showOrderSummary":true},"textOptions":{"completePaymentButtonText":"pay"}},"countryCode":"DEU"},"notifications":{"webhooks":[{"eventName":"payment.created","url":"http://localhost:8080/api/rest/v1/webhook/1234","authorization":""}]}}`,
	}, entity.ProtocolEntry{
		ReferenceId: "220118-150405-000004",
		ApiId:       "42",
		Kind:        "raw",
		Message:     "nexi create response",
		Details:     `{"paymentId":"42","hostedPaymentPageUrl":"http://localhost/some/pay/link"}`,
	})
}

func tstRequireProtocolEntries(t *testing.T, expectedProtocol ...entity.ProtocolEntry) {
	db := database.GetRepository().(*inmemorydb.InMemoryRepository)
	actualProtocol := db.ProtocolEntries()
	require.Equal(t, len(expectedProtocol), len(actualProtocol))
	for i, expected := range expectedProtocol {
		actual := *(actualProtocol[i])
		require.Equal(t, expected.ReferenceId, actual.ReferenceId)
		require.Equal(t, expected.ApiId, actual.ApiId)
		require.Equal(t, expected.Kind, actual.Kind)
		require.Equal(t, expected.Message, actual.Message)
		require.Equal(t, expected.Details, actual.Details)
	}
}

func p[T comparable](t T) *T {
	var nullValue T
	if t == nullValue {
		return nil
	}
	return &t
}

func unindentJSON(v string) string {
	parsed := map[string]interface{}{}
	if err := json.Unmarshal([]byte(v), &parsed); err != nil {
		return "error parsing json: " + err.Error()
	}
	unindented, err := json.Marshal(parsed)
	if err != nil {
		return "error rendering json: " + err.Error()
	}
	return string(unindented)
}
