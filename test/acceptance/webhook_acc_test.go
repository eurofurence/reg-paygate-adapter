package acceptance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/eurofurence/reg-paygate-adapter/docs"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/mailservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
	"github.com/stretchr/testify/require"
)

// --- webhook ---

func TestWebhook_Success_TolerantReader(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with valid information with some extra fields")
	_ = paymentMock.InjectTransaction(context.TODO(), paymentservice.Transaction{
		DebitorID: 1,
		ID:        "EF1995-000001-221216-122218-4132",
		Type:      "payment",
		Method:    "credit",
		Amount: paymentservice.Amount{
			Currency:  "EUR",
			GrossCent: 18500,
			VatRate:   19.0,
		},
		Comment:       "CC previously created", // will update to include paymentId
		Status:        "tentative",             // will update to valid
		EffectiveDate: "2022-12-10",            // will update to 2022-12-16 (mocked Now() date)
		DueDate:       "2022-12-10",
	})
	request := tstBuildValidWebhookRequest(t, "EF1995-000001-221216-122218-4132", "OK", 18500)
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the extra fields are ignored and the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	}, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		ApiId:       "ef00000000000000000000000000cafe",
		Kind:        "success",
		Message:     "transaction updated successfully",
		Details:     "amount=18500 currency=EUR",
	})

	docs.Then("and no notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{})
}

func TestWebhook_Success_TentativeNoDiff(t *testing.T) {
	docs.Description("webhook with status OK sets existing tentative tx to valid")
	tstWebhookSuccessCase(t,
		"EF1995-000001-221216-122218-4132",
		"OK",
		18500,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-221216-122218-4132",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 18500,
				VatRate:   19.0,
			},
			Comment:       "CC previously created", // will update to include paymentId
			Status:        "tentative",             // will update to valid
			EffectiveDate: "2022-12-10",            // will update to 2022-12-16 (mocked Now() date)
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{
			{
				DebitorID: 1,
				ID:        "EF1995-000001-221216-122218-4132",
				Type:      "payment",
				Method:    "credit",
				Amount: paymentservice.Amount{
					Currency:  "EUR",
					GrossCent: 18500,
					VatRate:   19.0,
				},
				Comment:       "CC paymentId ef00000000000000000000000000cafe - status OK",
				Status:        "valid",
				EffectiveDate: "2022-12-16",
				DueDate:       "2022-12-10",
			},
		},
		[]mailservice.MailSendDto{},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "transaction updated successfully",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_TentativeAuthorizedOKNoDiff(t *testing.T) {
	docs.Description("webhook with status AUTHORIZED but upstream OK sets existing tentative tx to valid")
	tstWebhookSuccessCase(t,
		"EF1995-000001-221216-122218-4132",
		"AUTHORIZED",
		18500,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-221216-122218-4132",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 18500,
				VatRate:   19.0,
			},
			Comment:       "CC previously created", // will update to include paymentId
			Status:        "tentative",             // will update to valid
			EffectiveDate: "2022-12-10",            // will update to 2022-12-16 (mocked Now() date)
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{
			{
				DebitorID: 1,
				ID:        "EF1995-000001-221216-122218-4132",
				Type:      "payment",
				Method:    "credit",
				Amount: paymentservice.Amount{
					Currency:  "EUR",
					GrossCent: 18500,
					VatRate:   19.0,
				},
				Comment:       "CC paymentId ef00000000000000000000000000cafe - status OK",
				Status:        "valid",
				EffectiveDate: "2022-12-16",
				DueDate:       "2022-12-10",
			},
		},
		[]mailservice.MailSendDto{},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "transaction updated successfully",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_TentativeDiff(t *testing.T) {
	docs.Description("webhook with status OK sets existing tentative tx to pending and warns about differing amounts")
	tstWebhookSuccessCase(t,
		"EF1995-000001-221216-122218-4132",
		"OK",
		18500,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-221216-122218-4132",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 22500, // will update to actual amount, but send warning email
				VatRate:   19.0,
			},
			Comment:       "CC previously created", // will update to include paymentId
			Status:        "tentative",             // will update to pending due to differenct amount
			EffectiveDate: "2022-12-10",            // will update to 2022-12-16 (mocked Now() date)
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{
			{
				DebitorID: 1,
				ID:        "EF1995-000001-221216-122218-4132",
				Type:      "payment",
				Method:    "credit",
				Amount: paymentservice.Amount{
					Currency:  "EUR",
					GrossCent: 18500,
					VatRate:   19.0,
				},
				Comment:       "CC paymentId ef00000000000000000000000000cafe - status OK",
				Status:        "pending", // !!!
				EffectiveDate: "2022-12-16",
				DueDate:       "2022-12-10",
			},
		},
		[]mailservice.MailSendDto{
			tstExpectedMailNotification("webhook", "amount-difference-kept-pending-please-check"),
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "webhook payment amount differs",
				Details:     "old_amount=22500 amount=18500 old_currency=EUR currency=EUR",
			},
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "transaction updated successfully",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_AuthorizedWarns(t *testing.T) {
	docs.Description("webhook with status AUTHORIZED upstream AUTHORIZED sets existing tentative tx to pending and warns about status")
	tstWebhookSuccessCase(t,
		"EF1995-000001-230001-122218-5555",
		"AUTHORIZED",
		39000,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-230001-122218-5555",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 39000,
				VatRate:   19.0,
			},
			Comment:       "CC previously created", // will update to include paymentId
			Status:        "tentative",             // will update to pending due to differenct amount
			EffectiveDate: "2022-12-10",            // will update to 2022-12-16 (mocked Now() date)
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{
			{
				DebitorID: 1,
				ID:        "EF1995-000001-230001-122218-5555",
				Type:      "payment",
				Method:    "credit",
				Amount: paymentservice.Amount{
					Currency:  "EUR",
					GrossCent: 39000,
					VatRate:   19.0,
				},
				Comment:       "CC paymentId ef00000000000000000000000000cafe - status AUTHORIZED",
				Status:        "pending", // !!!
				EffectiveDate: "2022-12-16",
				DueDate:       "2022-12-10",
			},
		},
		[]mailservice.MailSendDto{
			{
				CommonID: "payment-nexi-adapter-error",
				Lang:     "en-US",
				To: []string{
					"errors@example.com",
				},
				Variables: map[string]string{
					"status":      "upstream-status-not-OK-kept-pending-please-check",
					"operation":   "webhook",
					"referenceId": "EF1995-000001-230001-122218-5555",
				},
				Async: true,
			},
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-230001-122218-5555",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "verified status not OK",
				Details:     "webhook=AUTHORIZED verified=AUTHORIZED",
			},
			{
				ReferenceId: "EF1995-000001-230001-122218-5555",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "transaction updated successfully",
				Details:     "amount=39000 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_OkAuthorizedWarns(t *testing.T) {
	docs.Description("webhook with status OK upstream AUTHORIZED sets existing tentative tx to pending and warns about status")
	tstWebhookSuccessCase(t,
		"EF1995-000001-230001-122218-5555",
		"OK",
		39000,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-230001-122218-5555",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 39000,
				VatRate:   19.0,
			},
			Comment:       "CC previously created", // will update to include paymentId
			Status:        "tentative",             // will update to pending due to differenct amount
			EffectiveDate: "2022-12-10",            // will update to 2022-12-16 (mocked Now() date)
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{
			{
				DebitorID: 1,
				ID:        "EF1995-000001-230001-122218-5555",
				Type:      "payment",
				Method:    "credit",
				Amount: paymentservice.Amount{
					Currency:  "EUR",
					GrossCent: 39000,
					VatRate:   19.0,
				},
				Comment:       "CC paymentId ef00000000000000000000000000cafe - status AUTHORIZED",
				Status:        "pending", // !!!
				EffectiveDate: "2022-12-16",
				DueDate:       "2022-12-10",
			},
		},
		[]mailservice.MailSendDto{
			{
				CommonID: "payment-nexi-adapter-error",
				Lang:     "en-US",
				To: []string{
					"errors@example.com",
				},
				Variables: map[string]string{
					"status":      "upstream-status-not-OK-kept-pending-please-check",
					"operation":   "webhook",
					"referenceId": "EF1995-000001-230001-122218-5555",
				},
				Async: true,
			},
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-230001-122218-5555",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "verified status not OK",
				Details:     "webhook=OK verified=AUTHORIZED",
			},
			{
				ReferenceId: "EF1995-000001-230001-122218-5555",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "transaction updated successfully",
				Details:     "amount=39000 currency=EUR",
			},
		},
	)
}

func TestWebhook_Error_Pending(t *testing.T) {
	docs.Description("webhook with status OK warns about trying to update pending tx and does not touch tx")
	tstWebhookSuccessCase(t,
		"EF1995-000001-221216-122218-4132",
		"OK",
		18500,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-221216-122218-4132",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 22500, // will NOT update to webhook amount, but send warning email
				VatRate:   19.0,
			},
			Comment:       "CC previously created",
			Status:        "pending", // will lead to error notification per mail and log
			EffectiveDate: "2022-12-10",
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{}, // NO update occurs!
		[]mailservice.MailSendDto{
			tstExpectedMailNotification("webhook", "abort-update-for-pending-OK"),
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "webhook payment already in status pending",
				Details:     "existing_amount=22500 ignored_amount=18500 existing_currency=EUR ignored_currency=EUR webhook_status=OK upstream_status=OK",
			},
		},
	)
}

func TestWebhook_Error_Valid(t *testing.T) {
	docs.Description("webhook with status OK warns about trying to update valid tx and does not touch tx")
	tstWebhookSuccessCase(t,
		"EF1995-000001-221216-122218-4132",
		"OK",
		18500,
		paymentservice.Transaction{
			DebitorID: 1,
			ID:        "EF1995-000001-221216-122218-4132",
			Type:      "payment",
			Method:    "credit",
			Amount: paymentservice.Amount{
				Currency:  "USD",
				GrossCent: 10500, // will NOT update to webhook amount, but send warning email
				VatRate:   7.0,
			},
			Comment:       "CC previously created",
			Status:        "valid", // will lead to error notification per mail and log
			EffectiveDate: "2022-12-10",
			DueDate:       "2022-12-10",
		},
		[]paymentservice.Transaction{}, // NO update occurs!
		[]mailservice.MailSendDto{
			tstExpectedMailNotification("webhook", "abort-update-for-valid-OK"),
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "webhook payment already in status valid",
				Details:     "existing_amount=10500 ignored_amount=18500 existing_currency=USD ignored_currency=EUR webhook_status=OK upstream_status=OK",
			},
		},
	)
}

func TestWebhook_FailureStatus(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with valid information indicating a non-OK status")
	request := tstBuildFailedWebhookRequest(t)
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and no requests to the payment service have been made")
	tstRequirePaymentServiceRecording(t, []paymentservice.Transaction{})

	docs.Then("and the expected error notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{
		{
			CommonID: "payment-nexi-adapter-error",
			Lang:     "en-US",
			To: []string{
				"errors@example.com",
			},
			Variables: map[string]string{
				"status":      "unexpected-status",
				"operation":   "webhook",
				"referenceId": "unknown status: FAILED",
			},
			Async: true,
		},
	})

	docs.Then("and the expected protocol entries have been written")
	fullExpectedProtocol := make([]entity.ProtocolEntry, 2)
	fullExpectedProtocol[0] = entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	}
	fullExpectedProtocol[1] = entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		ApiId:       "ef00000000000000000000000000cafe",
		Kind:        "error",
		Message:     "webhook FAILED unknown status",
		Details:     "code=00000020 desc=failed",
	}
	tstRequireProtocolEntries(t, fullExpectedProtocol...)
}

func TestWebhook_InvalidJson(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they attempt to trigger our webhook endpoint with an invalid json body")
	response := tstPerformPost(url, `{{{{}}`, tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "webhook.parse.error", nil)
}

func TestWebhook_WrongSecret(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who does not know the secret url")
	url := "/api/rest/v1/webhook/wrongsecret"

	docs.When("when they attempt to trigger our webhook endpoint")
	request := tstBuildValidWebhookRequest(t, "EF1995-000001-221216-122218-4132", "OK", 18500)
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", nil)
}

func TestWebhook_PaySrvDownstreamError(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they attempt to trigger our webhook endpoint while the downstream api is down")
	request := tstBuildValidWebhookRequest(t, "EF1995-000001-221216-122218-4132", "OK", 18500)
	paymentMock.SimulateAddError(paymentservice.DownstreamError)
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "webhook.downstream.error", nil)

	docs.Then("and the expected error notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{
		tstExpectedMailNotification("webhook", "create-missing-err"),
	})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	}, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		ApiId:       "ef00000000000000000000000000cafe",
		Kind:        "error",
		Message:     "webhook failed to create transaction in payment service",
		Details:     "amount=18500 currency=EUR error=downstream unavailable - see log for details",
	})
}

func TestWebhook_Success_Status_WrongPrefix(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint for a transaction id with the wrong prefix (usually a previous year)")
	request := `
  {
    "payId": "ef00000000000000000000000000cafe",
    "transId": "EF2001-000001-221216-122218-4132",
    "status": "OK",
    "responseCode": "00000000",
    "responseDescription": "success",
    "amount": {
      "value": 18500,
      "currency": "EUR"
    },
    "paymentMethods": {
      "type": "CARD"
    },
    "creationDate": "2025-12-15T13:24:23Z"
  }
`
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and no requests to the payment service have been made")
	tstRequirePaymentServiceRecording(t, nil)

	docs.Then("and the expected error notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{
		{
			CommonID: "payment-nexi-adapter-error",
			Lang:     "en-US",
			To: []string{
				"errors@example.com",
			},
			Variables: map[string]string{
				"status":      "ref-id-prefix-mismatch",
				"operation":   "webhook",
				"referenceId": "EF2001-000001-221216-122218-4132",
			},
			Async: true,
		},
	})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	}, entity.ProtocolEntry{
		ReferenceId: "EF2001-000001-221216-122218-4132",
		ApiId:       "ef00000000000000000000000000cafe",
		Kind:        "error",
		Message:     "webhook OK ref-id-prefix wrong",
		Details:     "expecting prefix EF1995",
	})
}

// --- helpers ---

func tstWebhookSuccessCase(
	t *testing.T,
	txId string,
	webhookStatus string,
	amount int64,
	injectedTx paymentservice.Transaction,
	expectedPaymentServiceRecording []paymentservice.Transaction,
	expectedMailRecording []mailservice.MailSendDto,
	expectedDBProtocol []entity.ProtocolEntry,
) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	if injectedTx.Status == "deleted" {
		docs.Given("and the payment service has no matching transaction")
	} else if injectedTx.Status != "" {
		// allows using a blank transaction if the transactions in payment service do not affect the test case
		docs.Given(fmt.Sprintf("and the payment service has a matching transaction in status %s amount %d", string(injectedTx.Status), injectedTx.Amount.GrossCent))
		_ = paymentMock.InjectTransaction(context.TODO(), injectedTx)
	}

	docs.When("when they trigger our webhook endpoint with valid information")
	request := tstBuildValidWebhookRequest(t, txId, webhookStatus, amount)
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	if len(expectedPaymentServiceRecording) == 0 {
		docs.Then("and no requests to the payment service have been made")
	} else {
		docs.Then("and the expected requests to the payment service have been made")
	}
	tstRequirePaymentServiceRecording(t, expectedPaymentServiceRecording)

	if len(expectedMailRecording) == 0 {
		docs.Then("and no error notification emails have been sent")
	} else {
		docs.Then("and the expected error notification emails have been sent")
	}
	tstRequireMailServiceRecording(t, expectedMailRecording)

	if len(expectedDBProtocol) == 0 {
		docs.Then("and no protocol entries (other than the raw request log) have been written")
	} else {
		docs.Then("and the expected protocol entries have been written")
	}
	fullExpectedProtocol := make([]entity.ProtocolEntry, len(expectedDBProtocol)+1)
	fullExpectedProtocol[0] = entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	}
	for i := 0; i < len(expectedDBProtocol); i++ {
		fullExpectedProtocol[i+1] = expectedDBProtocol[i]
	}
	tstRequireProtocolEntries(t, fullExpectedProtocol...)
}

// --- weblogger ---

func TestWeblogger_Success(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/weblogger/demosecret"

	docs.When("when they trigger our weblogger endpoint")
	request := tstBuildValidWebhookRequest(t, "EF1995-000001-221216-122218-4132", "OK", 18500)
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	})

	docs.Then("and no notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{})
}
