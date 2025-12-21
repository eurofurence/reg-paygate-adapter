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

func TestWebhook_Success_TolerantReader(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with valid information with some extra fields")
	request := tstBuildValidWebhookRequest(t, "payment.created")
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
		Message:     "webhook payment.created",
		Details:     "amount=18500 currency=EUR",
	})

	docs.Then("and no notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{})
}

func TestWebhook_Success_PaymentCreated(t *testing.T) {
	docs.Description("payment.created events are logged but otherwise ignored")
	tstWebhookSuccessCase(t, "payment.created",
		paymentservice.Transaction{},
		[]paymentservice.Transaction{},
		[]mailservice.MailSendDto{},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "webhook payment.created",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_ChargeCreatedV2(t *testing.T) {
	docs.Description("payment.charge.created.v2 events are logged but otherwise ignored")
	tstWebhookSuccessCase(t, "payment.charge.created.v2",
		paymentservice.Transaction{},
		[]paymentservice.Transaction{},
		[]mailservice.MailSendDto{},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "webhook payment.charge.created.v2",
				Details:     "method=Visa type=CARD amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_CheckoutCompleted_TentativeNoDiff(t *testing.T) {
	docs.Description("payment.checkout.completed events set existing tentative tx to valid")
	tstWebhookSuccessCase(t, "payment.checkout.completed",
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
				Comment:       "CC paymentId ef00000000000000000000000000cafe",
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
				Message:     "webhook payment.checkout.completed updated tx",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_CheckoutCompleted_TentativeDiff(t *testing.T) {
	docs.Description("payment.checkout.completed events set existing tentative tx to pending and warn about differing amounts")
	tstWebhookSuccessCase(t, "payment.checkout.completed",
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
				Comment:       "CC paymentId ef00000000000000000000000000cafe",
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
				Message:     "webhook payment.checkout.completed payment amount differs",
				Details:     "old_amount=22500 amount=18500 old_currency=EUR currency=EUR",
			},
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "webhook payment.checkout.completed updated tx",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Error_CheckoutCompleted_Pending(t *testing.T) {
	docs.Description("payment.checkout.completed events warn about trying to update pending tx and do not touch tx")
	tstWebhookSuccessCase(t, "payment.checkout.completed",
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
			tstExpectedMailNotification("webhook", "abort-update-for-pending"),
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "webhook payment.checkout.completed payment already in status pending",
				Details:     "existing_amount=22500 ignored_amount=18500 existing_currency=EUR ignored_currency=EUR",
			},
		},
	)
}

func TestWebhook_Error_CheckoutCompleted_Valid(t *testing.T) {
	docs.Description("payment.checkout.completed events warn about trying to update valid tx and do not touch tx")
	tstWebhookSuccessCase(t, "payment.checkout.completed",
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
			tstExpectedMailNotification("webhook", "abort-update-for-valid"),
		},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "EF1995-000001-221216-122218-4132",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "warning",
				Message:     "webhook payment.checkout.completed payment already in status valid",
				Details:     "existing_amount=10500 ignored_amount=18500 existing_currency=USD ignored_currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_UnexpectedEvents(t *testing.T) {
	// TODO charge failed should be a supported event
	for _, event := range []string{"payment.cancel.created", "payment.charge.created", "payment.charge.failed", "payment.charge.failed.v2"} {
		testname := fmt.Sprintf("Event_%s", event)
		t.Run(testname, func(t *testing.T) {
			tstWebhookSuccessCase(t, event, paymentservice.Transaction{}, []paymentservice.Transaction{},
				[]mailservice.MailSendDto{
					{
						CommonID: "payment-nexi-adapter-error",
						Lang:     "en-US",
						To: []string{
							"errors@example.com",
						},
						Variables: map[string]string{
							"status":      "unexpected-event",
							"operation":   "webhook",
							"referenceId": fmt.Sprintf("unknown event: %s", event),
						},
					},
				},
				[]entity.ProtocolEntry{
					{
						ReferenceId: "",
						ApiId:       "",
						Kind:        "error",
						Message:     fmt.Sprintf("webhook %s unknown event", event),
						Details:     "",
					},
				})
		})
	}
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
	response := tstPerformPost(url, tstBuildValidWebhookRequest(t, "payment.created"), tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", nil)
}

func TestWebhook_PaySrvDownstreamError(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they attempt to trigger our webhook endpoint while the downstream api is down")
	request := tstBuildValidWebhookRequest(t, "payment.checkout.completed")
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
		Message:     "webhook payment.checkout.completed failed to create transaction in payment service",
		Details:     "amount=18500 currency=EUR error=downstream unavailable - see log for details",
	})
}

func TestWebhook_Success_Status_WrongPrefix(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with the wrong prefix")
	request := `
  {
    "id": "01234567890abcdef0123456789abcde",
    "merchantId": 123456789,
    "timestamp": "2025-12-15T13:24:23.0175+00:00",
    "event": "payment.checkout.completed",
    "data": {
      "order": {
        "amount": {
          "amount": 18500,
          "currency": "EUR"
        },
        "reference": "EF2001-000001-221216-122218-4132"
      },
      "paymentId": "ef00000000000000000000000000cafe"
    }
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
		Message:     "webhook payment.checkout.completed ref-id-prefix wrong",
		Details:     "expecting prefix EF1995",
	})
}

// --- helpers ---

func tstWebhookSuccessCase(
	t *testing.T,
	event string,
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
	request := tstBuildValidWebhookRequest(t, event)
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
