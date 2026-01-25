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
	"github.com/eurofurence/reg-paygate-adapter/docs"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database/inmemorydb"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
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
	createRequest := nexi.NexiCreateCheckoutSessionRequest{
		TransId: "220118-150405-000004",
		Amount: nexi.NexiAmount{
			Value:    10550,
			Currency: "EUR",
		},
		Language: "de",
		Urls: nexi.NexiPaymentUrlsRequest{
			Return:  "https://example.com/success",
			Cancel:  "https://example.com/failure",
			Webhook: "http://localhost:8080/api/rest/v1/webhook/1234",
		},
		StatementDescriptor: "Convention Registration",
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
		Url:  "http://localhost:8000/payments/sessions",
		Body: `{"transId":"220118-150405-000004","amount":{"value":10550,"currency":"EUR"},"language":"de","urls":{"return":"https://example.com/success","cancel":"https://example.com/failure","webhook":"http://localhost:8080/api/rest/v1/webhook/1234"},"statementDescriptor":"Convention Registration"}`,
	}, aurestclientapi.ParsedResponse{
		Body: &nexi.NexiCreateCheckoutSessionResponse{
			Links: nexi.NexiCheckoutSessionResponseLinks{
				Redirect: &nexi.RedirectLink{
					Href: "http://localhost/some/pay/link",
					Type: "hosted", // TODO what is the real value?
				},
			},
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
		Url:    "http://localhost:8000/payments/getByTransId/220118-150405-000004",
		Body:   "",
	}, aurestclientapi.ParsedResponse{
		Body: &nexi.NexiPaymentQueryResponse{
			PayId:               "42",
			TransId:             "220118-150405-000004",
			Status:              "OK",
			ResponseCode:        "00000000",
			ResponseDescription: "success",
			Amount: &nexi.NexiAmountResponse{
				Value:    10550,
				Currency: "EUR",
			},
			// TODO better example response
		},
		Status: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Time: time.Time{},
	}, nil)

	// set up downstream client
	client := nexi.NewTestingClient(verifierClient)

	// STEP 1: create a new payment link
	created, err := client.CreatePaymentLink(ctx, createRequest)
	require.Nil(t, err)
	require.Equal(t, "http://localhost/some/pay/link", created.Links.Redirect.Href)

	// STEP 2: read the payment link again after use
	read, err := client.QueryPaymentLink(ctx, "220118-150405-000004")
	require.Nil(t, err)
	require.Equal(t, "42", read.PayId)
	require.Equal(t, "220118-150405-000004", read.TransId)
	require.Equal(t, "OK", read.Status)
	require.Equal(t, int64(10550), read.Amount.Value)
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
