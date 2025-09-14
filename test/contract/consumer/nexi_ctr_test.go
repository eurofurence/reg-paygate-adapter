package main

import (
	"context"
	auzerolog "github.com/StephanHCB/go-autumn-logging-zerolog"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	aurestverifier "github.com/StephanHCB/go-autumn-restclient/implementation/verifier"
	"github.com/eurofurence/reg-payment-nexi-adapter/docs"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/entity"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database/inmemorydb"
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
	createRequest := nexi.PaymentLinkCreateRequest{
		Title:       "Convention Registration",
		Description: "Please pay for your registration",
		PSP:         1,
		ReferenceId: "220118-150405-000004",
		OrderId:     "220118-150405-000004",
		Purpose:     "EF 2022 REG 000004",
		Amount:      10550,
		VatRate:     19.0,
		Currency:    "EUR",
		SKU:         "REG2022V01AT000004",
		Email:       "test@example.com",
	}
	createRequestSampleBody := `title=Convention%20Registration&description=Please%20pay%20for%20your%20registration&psp=1&` +
		`referenceId=220118-150405-000004&purpose=EF%202022%20REG%20000004&amount=10550&vatRate=19.0&currency=EUR&` +
		`sku=REG2022V01AT000004&preAuthorization=0&reservation=0&` +
		`fields%5Bemail%5D%5Bmandatory%5D=1&fields%5Bemail%5D%5BdefaultValue%5D=test@example.com&ApiSignature=omitted`
	createRequestResponse := `{
  "status": "success",
  "data": [
    {
      "id": 42,
      "status": "waiting",
      "hash": "77871ec636d239b136e4d79d32c376ad",
      "referenceId": "220118-150405-000004",
      "link": "http://localhost/some/pay/link",
      "invoices": [],
      "preAuthorization": false,
      "name": "",
      "title": "Convention Registration",
      "description": "Please pay for your registration",
      "buttonText": "",
      "api": true,
      "fields": {
        "header": {
          "active": true,
          "mandatory": false,
          "names": {
            "de": "Kontaktdaten"
          }
        }
      },
      "psp": [],
      "pm": [],
      "purpose": {
        "1": "EF 2022 REG 000004"
      },
      "amount": 10550,
      "currency": "EUR",
      "vatRate": "19.0",
      "sku": "REG2022V01AT000004",
      "subscriptionState": false,
      "subscriptionInterval": "",
      "subscriptionPeriod": "",
      "subscriptionPeriodMinAmount": "",
      "subscriptionCancellationInterval": "",
      "createdAt": 1665838673,
      "requestId": 7489378
    }
  ]
}`

	// STEP 2: query after it has been used
	queryRequestSampleBody := `ApiSignature=omitted`
	queryRequestResponse := `{
  "status": "success",
  "data": [
    {
      "id": 424242,
      "status": "confirmed",
      "hash": "77871ec636d239b136e4d79d32c376ad",
      "referenceId": "220118-150405-000004",
      "link": "http://localhost/some/pay/link",
      "invoices": [
        {
          "number": "EF 2022 REG 000004",
          "products": [
            {
              "name": "EF 2022 REG 000004",
              "price": 10550,
              "quantity": 1,
              "sku": "REG2022V01AT000004",
              "vatRate": null
            }
          ],
          "amount": 10550,
          "discount": {
            "code": null,
            "amount": 0,
            "percentage": null
          },
          "shippingAmount": null,
          "currency": "EUR",
          "test": 1,
          "referenceId": "220118-150405-000004",
          "paymentRequestId": 42,
          "paymentLink": {
            "hash": "77871ec636d239b136e4d79d32c376ad",
            "referenceId": "220118-150405-000004",
            "email": ""
          },
          "transactions": [
            {
              "id": 777777,
              "uuid": "b9bee580",
              "amount": 10550,
              "referenceId": "220118-150405-000004",
              "time": "2022-10-15 15:50:20",
              "status": "confirmed",
              "lang": "de",
              "psp": "ConCardis_PayEngine_3",
              "pspId": 29,
              "mode": "TEST",
              "metadata": [],
              "contact": {
                "id": 888888,
                "uuid": "50659dad",
                "title": "",
                "firstname": "",
                "lastname": "",
                "company": "",
                "street": "",
                "zip": "",
                "place": "",
                "country": "",
                "countryISO": "",
                "phone": "",
                "email": "",
                "date_of_birth": "",
                "delivery_title": "",
                "delivery_firstname": "",
                "delivery_lastname": "",
                "delivery_company": "",
                "delivery_street": "",
                "delivery_zip": "",
                "delivery_place": "",
                "delivery_country": "",
                "delivery_countryISO": ""
              },
              "subscription": null,
              "pageUuid": null,
              "payment": {
                "brand": "visa"
              }
            }
          ],
          "custom_fields": [
            {
              "type": "header",
              "name": "Kontaktdaten",
              "value": "Kontaktdaten"
            }
          ]
        }
      ],
      "preAuthorization": false,
      "name": "",
      "title": "Convention Registration",
      "description": "Please pay for your registration",
      "buttonText": "",
      "api": true,
      "fields": {
        "header": {
          "active": true,
          "mandatory": false,
          "names": {
            "de": "Kontaktdaten"
          }
        }
      },
      "psp": [],
      "pm": [],
      "purpose": {
        "1": "EF 2022 REG 000004"
      },
      "amount": 10550,
      "currency": "EUR",
      "vatRate": 19.0,
      "sku": "REG2022V01AT000004",
      "subscriptionState": false,
      "subscriptionInterval": "",
      "subscriptionPeriod": "",
      "subscriptionPeriodMinAmount": 0,
      "subscriptionCancellationInterval": "",
      "createdAt": 1665838673
    }
  ]
}`
	// STEP 3: delete
	deleteRequestSampleBody := `ApiSignature=omitted`

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
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
		Url:  "http://localhost:8000/v1.0/Invoice/?instance=myinstance",
		Body: createRequestSampleBody,
	}, aurestclientapi.ParsedResponse{
		Body:   createRequestResponse,
		Status: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Time: time.Time{},
	}, nil)
	verifierImpl.AddExpectation(aurestverifier.Request{
		Name:   "read-paylink-after-use",
		Method: http.MethodGet,
		Header: http.Header{ // not verified
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
		Url:  "http://localhost:8000/v1.0/Invoice/42/?instance=myinstance",
		Body: queryRequestSampleBody,
	}, aurestclientapi.ParsedResponse{
		Body:   queryRequestResponse,
		Status: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Time: time.Time{},
	}, nil)
	verifierImpl.AddExpectation(aurestverifier.Request{
		Name:   "delete-paylink",
		Method: http.MethodDelete,
		Header: http.Header{ // not verified
			"Content-Type": []string{"application/x-www-form-urlencoded"},
		},
		Url:  "http://localhost:8000/v1.0/Invoice/42/?instance=myinstance",
		Body: deleteRequestSampleBody,
	}, aurestclientapi.ParsedResponse{
		Status: http.StatusOK,
		Time:   time.Time{},
	}, nil)

	// set up downstream client
	client := nexi.NewTestingClient(verifierClient)
	// verifier does not support regex matchers for x-www-form-urlencoded
	nexi.FixedSignatureValue = "omitted"

	// STEP 1: create a new payment link
	created, err := client.CreatePaymentLink(ctx, createRequest)
	require.Nil(t, err)
	require.Equal(t, uint(42), created.ID)
	require.Equal(t, "220118-150405-000004", created.ReferenceID)
	require.Equal(t, "http://localhost/some/pay/link", created.Link)

	// STEP 2: read the payment link again after use
	read, err := client.QueryPaymentLink(ctx, created.ID)
	require.Nil(t, err)
	require.Equal(t, 1, len(read.Invoices))
	require.Equal(t, uint(42), read.Invoices[0].PaymentRequestId)
	require.Equal(t, "220118-150405-000004", read.ReferenceID)
	require.Equal(t, "confirmed", read.Status)

	// STEP 3: delete the payment link (wouldn't normally work after use)
	err = client.DeletePaymentLink(ctx, created.ID)
	require.Nil(t, err)

	docs.Then("and the expected interactions have occurred in the correct order")
	require.Nil(t, verifierImpl.FirstUnexpectedOrNil())

	docs.Then("and the expected protocol entries have been written to the database")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "220118-150405-000004",
		ApiId:       0,
		Kind:        "raw",
		Message:     "nexi request",
		Details:     "title=Convention%20Registration&description=Please%20pay%20for%20your%20registration&psp=1&referenceId=220118-150405-000004&purpose=EF%202022%20REG%20000004&amount=10550&vatRate=19.0&currency=EUR&sku=REG2022V01AT000004&preAuthorization=0&reservation=0&fields%5Bemail%5D%5Bmandatory%5D=1&fields%5Bemail%5D%5BdefaultValue%5D=test@example.com",
	}, entity.ProtocolEntry{
		ReferenceId: "220118-150405-000004",
		ApiId:       0,
		Kind:        "raw",
		Message:     "nexi response",
		Details:     `{"status":"success","data":[{"id":42,"status":"waiting","hash":"77871ec636d239b136e4d79d32c376ad","referenceId":"220118-150405-000004","link":"http://localhost/some/pay/link","invoices":[],"preAuthorization":false,"name":"","title":"ConventionRegistration","description":"Pleasepayforyourregistration","buttonText":"","api":true,"fields":{"header":{"active":true,"mandatory":false,"names":{"de":"Kontaktdaten"}}},"psp":[],"pm":[],"purpose":{"1":"EF2022REG000004"},"amount":10550,"currency":"EUR","vatRate":"19.0","sku":"REG2022V01AT000004","subscriptionState":false,"subscriptionInterval":"","subscriptionPeriod":"","subscriptionPeriodMinAmount":"","subscriptionCancellationInterval":"","createdAt":1665838673,"requestId":7489378}]}`,
	}, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       42,
		Kind:        "raw",
		Message:     "nexi response",
		Details:     `{"status":"success","data":[{"id":424242,"status":"confirmed","hash":"77871ec636d239b136e4d79d32c376ad","referenceId":"220118-150405-000004","link":"http://localhost/some/pay/link","invoices":[{"number":"EF2022REG000004","products":[{"name":"EF2022REG000004","price":10550,"quantity":1,"sku":"REG2022V01AT000004","vatRate":null}],"amount":10550,"discount":{"code":null,"amount":0,"percentage":null},"shippingAmount":null,"currency":"EUR","test":1,"referenceId":"220118-150405-000004","paymentRequestId":42,"paymentLink":{"hash":"77871ec636d239b136e4d79d32c376ad","referenceId":"220118-150405-000004","email":""},"transactions":[{"id":777777,"uuid":"b9bee580","amount":10550,"referenceId":"220118-150405-000004","time":"2022-10-1515:50:20","status":"confirmed","lang":"de","psp":"ConCardis_PayEngine_3","pspId":29,"mode":"TEST","metadata":[],"contact":{"id":888888,"uuid":"50659dad","title":"","firstname":"","lastname":"","company":"","street":"","zip":"","place":"","country":"","countryISO":"","phone":"","email":"","date_of_birth":"","delivery_title":"","delivery_firstname":"","delivery_lastname":"","delivery_company":"","delivery_street":"","delivery_zip":"","delivery_place":"","delivery_country":"","delivery_countryISO":""},"subscription":null,"pageUuid":null,"payment":{"brand":"visa"}}],"custom_fields":[{"type":"header","name":"Kontaktdaten","value":"Kontaktdaten"}]}],"preAuthorization":false,"name":"","title":"ConventionRegistration","description":"Pleasepayforyourregistration","buttonText":"","api":true,"fields":{"header":{"active":true,"mandatory":false,"names":{"de":"Kontaktdaten"}}},"psp":[],"pm":[],"purpose":{"1":"EF2022REG000004"},"amount":10550,"currency":"EUR","vatRate":19.0,"sku":"REG2022V01AT000004","subscriptionState":false,"subscriptionInterval":"","subscriptionPeriod":"","subscriptionPeriodMinAmount":0,"subscriptionCancellationInterval":"","createdAt":1665838673}]}`,
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
