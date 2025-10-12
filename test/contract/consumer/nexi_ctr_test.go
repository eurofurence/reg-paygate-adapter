package main

import (
	"context"
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
	"net/http"
	"testing"
	"time"
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
					TaxRate:          19.0,
					TaxAmount:        0,
					GrossTotalAmount: 10550,
					NetTotalAmount:   10550,
					ImageUrl:         "",
				},
			},
			Amount:    10550,
			Currency:  "EUR",
			Reference: "220118-150405-000004",
		},
		Checkout: nexi.NexiCheckout{
			Url:             "",
			IntegrationType: "hostedPaymentPage",
			ReturnUrl:       "https://example.com/success",
			CancelUrl:       "https://example.com/failure",
			Consumer: nexi.NexiConsumer{
				Reference: "test@example.com",
				Email:     "test@example.com",
			},
			TermsUrl:         "",
			MerchantTermsUrl: "",
			ShippingCountries: []nexi.NexiCountry{
				{CountryCode: "DE"},
			},
			Shipping: nexi.NexiShipping{
				Countries: []nexi.NexiCountry{
					{CountryCode: "DE"},
				},
				MerchantHandlesShippingCost: false,
				EnableBillingAddress:        true,
			},
			ConsumerType: nexi.NexiConsumerType{
				Default:        "b2c",
				SupportedTypes: []string{"b2c", "b2b"},
			},
			Charge:                      true,
			PublicDevice:                false,
			MerchantHandlesConsumerData: false,
			CountryCode: "DE",
		},
		Appearance: nexi.NexiAppearance{
			DisplayOptions: nexi.NexiDisplayOptions{
				ShowMerchantName: true,
				ShowOrderSummary: true,
			},
			TextOptions: nexi.NexiTextOptions{
				CompletePaymentButtonText: "Complete Payment",
			},
		},
		Notifications: nexi.NexiNotifications{
			Webhooks: []nexi.NexiWebhook{},
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
		Body: `{"order":{"items":[{"reference":"EF 2022 REG 000004","name":"Convention Registration","quantity":1,"unit":"qty","unitPrice":10550,"taxRate":19,"taxAmount":0,"grossTotalAmount":10550,"netTotalAmount":10550,"imageUrl":""}],"amount":10550,"currency":"EUR","reference":"220118-150405-000004"},"checkout":{"url":"","integrationType":"hostedPaymentPage","returnUrl":"https://example.com/success","cancelUrl":"https://example.com/failure","consumer":{"reference":"test@example.com","email":"test@example.com","shippingAddress":{"addressLine1":"","addressLine2":"","postalCode":"","city":"","country":""},"billingAddress":{"addressLine1":"","addressLine2":"","postalCode":"","city":"","country":""},"phoneNumber":{"prefix":"","number":""},"privatePerson":{"firstName":"","lastName":""},"company":{"name":"","contact":{"firstName":"","lastName":""}}},"termsUrl":"","merchantTermsUrl":"","shippingCountries":[{"countryCode":"DE"}],"shipping":{"countries":[{"countryCode":"DE"}],"merchantHandlesShippingCost":false,"enableBillingAddress":true},"consumerType":{"default":"b2c","supportedTypes":["b2c","b2b"]},"charge":true,"publicDevice":false,"merchantHandlesConsumerData":false,"countryCode":"DE"},"merchantNumber":"1234","appearance":{"displayOptions":{"showMerchantName":true,"showOrderSummary":true},"textOptions":{"completePaymentButtonText":"Complete Payment"}},"notifications":{"webhooks":[]}}`,
	}, aurestclientapi.ParsedResponse{
		Body:   `{"paymentId":"42","hostedPaymentPageUrl":"http://localhost/some/pay/link"}`,
		Status: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Time: time.Time{},
	}, nil)
	verifierImpl.AddExpectation(aurestverifier.Request{
		Name:   "read-paylink-after-use",
		Method: http.MethodGet,
		Header: http.Header{}, // not verified
		Url:  "http://localhost:8000/v1/payments/42",
		Body: "",
	}, aurestclientapi.ParsedResponse{
		Body:   `{"payment":{"paymentId":"42","summary":{"reservedAmount":0,"reservedSurchargeAmount":0,"chargedAmount":10550,"chargedSurchargeAmount":0,"refundedAmount":0,"refundedSurchargeAmount":0,"cancelledAmount":0,"cancelledSurchargeAmount":0},"consumer":{"shippingAddress":{"addressLine1":"","addressLine2":"","receiverLine":"","postalCode":"","city":"","country":"","phoneNumber":{"prefix":"","number":""}},"company":{"merchantReference":"","name":"","registrationNumber":"","contactDetails":{"firstName":"","lastName":"","email":"","phoneNumber":{"prefix":"","number":""}}},"privatePerson":{"merchantReference":"","dateOfBirth":"","firstName":"","lastName":"","email":"","phoneNumber":{"prefix":"","number":""}},"billingAddress":{"addressLine1":"","addressLine2":"","receiverLine":"","postalCode":"","city":"","country":"","phoneNumber":{"prefix":"","number":""}}},"paymentDetails":{"paymentType":"card","paymentMethod":"visa","invoiceDetails":{"invoiceNumber":""},"cardDetails":{"maskedPan":"411111******1111","expiryDate":"12/25"}},"orderDetails":{"amount":10550,"currency":"EUR","reference":"220118-150405-000004"},"checkout":{"url":"http://localhost/some/pay/link","cancelUrl":"https://example.com/failure"},"created":"2024-10-15T15:50:20Z","refunds":[],"charges":[],"terminated":""}}}`,
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
		Url:  "http://localhost:8000/v1/payments/42/cancels",
		Body: `{"amount":10550}`,
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
	require.Equal(t, "220118-150405-000004", created.ReferenceID)
	require.Equal(t, "http://localhost/some/pay/link", created.Link)

	// STEP 2: read the payment link again after use
	read, err := client.QueryPaymentLink(ctx, created.ID)
	require.Nil(t, err)
	require.Equal(t, "42", read.ID)
	require.Equal(t, "220118-150405-000004", read.ReferenceID)
	require.Equal(t, "confirmed", read.Status)
	require.Equal(t, int64(10550), read.Amount)

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
		Details:     `{"order":{"items":[{"reference":"EF 2022 REG 000004","name":"Convention Registration","quantity":1,"unit":"qty","unitPrice":10550,"taxRate":19,"taxAmount":0,"grossTotalAmount":10550,"netTotalAmount":10550,"imageUrl":""}],"amount":10550,"currency":"EUR","reference":"220118-150405-000004"},"checkout":{"url":"","integrationType":"hostedPaymentPage","returnUrl":"https://example.com/success","cancelUrl":"https://example.com/failure","consumer":{"reference":"test@example.com","email":"test@example.com","shippingAddress":{"addressLine1":"","addressLine2":"","postalCode":"","city":"","country":""},"billingAddress":{"addressLine1":"","addressLine2":"","postalCode":"","city":"","country":""},"phoneNumber":{"prefix":"","number":""},"privatePerson":{"firstName":"","lastName":""},"company":{"name":"","contact":{"firstName":"","lastName":""}}},"termsUrl":"","merchantTermsUrl":"","shippingCountries":[{"countryCode":"DE"}],"shipping":{"countries":[{"countryCode":"DE"}],"merchantHandlesShippingCost":false,"enableBillingAddress":true},"consumerType":{"default":"b2c","supportedTypes":["b2c","b2b"]},"charge":true,"publicDevice":false,"merchantHandlesConsumerData":false,"countryCode":"DE"},"merchantNumber":"1234","appearance":{"displayOptions":{"showMerchantName":true,"showOrderSummary":true},"textOptions":{"completePaymentButtonText":"Complete Payment"}},"notifications":{"webhooks":[]}}`,
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
