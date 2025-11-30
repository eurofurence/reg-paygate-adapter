package acceptance

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/eurofurence/reg-paygate-adapter/docs"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/mailservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
	"github.com/stretchr/testify/require"
)

func TestWebhook_Success_TolerantReader(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with valid information with lots of extra fields which we ignore")
	response := tstPerformPost(url, tstBuildValidWebhookRequest(), tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "221216-122218-000001",
		ApiId:       "42",
		Kind:        "success",
		Message:     "webhook query-pay-link",
		Details:     "status=confirmed amount=390",
	})
}

func TestWebhook_Success_Status_Confirmed(t *testing.T) {
	tstWebhookSuccessCase(t, "confirmed", []paymentservice.Transaction{
		{
			ID: "mock-transaction-id",
			Amount: paymentservice.Amount{
				Currency:  "EUR",
				GrossCent: 390,
			},
			Status:        "valid",
			EffectiveDate: "2023-01-08",
			Comment:       "CC orderId 42",
		},
	}, []mailservice.MailSendDto{}, []entity.ProtocolEntry{
		{
			ReferenceId: "221216-122218-000001",
			ApiId:       "42",
			Kind:        "success",
			Message:     "webhook query-pay-link",
			Details:     "status=confirmed amount=390",
		},
	})
}

func TestWebhook_Success_Status_Ignored(t *testing.T) {
	for _, status := range []string{"cancelled", "declined"} {
		testname := fmt.Sprintf("Status_%s", status)
		t.Run(testname, func(t *testing.T) {
			tstWebhookSuccessCase(t, status, []paymentservice.Transaction{}, []mailservice.MailSendDto{}, []entity.ProtocolEntry{
				{
					ReferenceId: "221216-122218-000001",
					ApiId:       "42",
					Kind:        "success",
					Message:     "webhook query-pay-link",
					Details:     fmt.Sprintf("status=%s amount=390", status),
				},
			})
		})
	}
}

func TestWebhook_Success_Status_NotifyMail(t *testing.T) {
	for _, status := range []string{"waiting", "authorized", "refunded", "partially-refunded", "refund_pending", "chargeback", "error", "uncaptured", "reserved"} {
		testname := fmt.Sprintf("Status_%s", status)
		t.Run(testname, func(t *testing.T) {
			tstWebhookSuccessCase(t, status, []paymentservice.Transaction{}, []mailservice.MailSendDto{
				tstExpectedMailNotification("webhook", status),
			}, []entity.ProtocolEntry{
				{
					ReferenceId: "221216-122218-000001",
					ApiId:       "42",
					Kind:        "success",
					Message:     "webhook query-pay-link",
					Details:     fmt.Sprintf("status=%s amount=390", status),
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
	response := tstPerformPost(url, tstBuildValidWebhookRequest(), tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", nil)
}

func TestWebhook_DownstreamError(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they attempt to trigger our webhook endpoint while the downstream api is down")
	nexiMock.SimulateError(nexi.DownstreamError)
	response := tstPerformPost(url, tstBuildValidWebhookRequest(), tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "webhook.downstream.error", nil)
}

func TestWebhook_Success_Status_WrongPrefix(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given the payment provider has a transaction in status confirmed")

	docs.Given("and an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with the wrong prefix")
	request := `
{
   "transaction": {
       "id": 1892362736,
       "invoice": {
           "paymentRequestId": 4242,
           "referenceId": "230001-122218-000001"
       }
   }
}
`
	response := tstPerformPost(url, request, tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and the expected downstream requests have been made to the nexi api")
	tstRequireNexiRecording(t,
		"QueryPaymentLink 4242",
	)

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
				"status":      "ref-id-prefix",
				"operation":   "webhook",
				"referenceId": "230001-122218-000001",
			},
		},
	})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "230001-122218-000001",
		ApiId:       "4242",
		Kind:        "error",
		Message:     "webhook ref-id-prefix",
		Details:     "ref-id=230001-122218-000001",
	})
}

// --- helpers ---

func tstWebhookSuccessCase(t *testing.T, status string, expectedPaymentServiceRecording []paymentservice.Transaction, expectedMailRecording []mailservice.MailSendDto, expectedProtocol []entity.ProtocolEntry) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given(fmt.Sprintf("given the payment provider has a transaction in status %s", status))
	if status != "confirmed" {
		nexiMock.ManipulateStatus("42", status)
	}

	docs.Given("and an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they trigger our webhook endpoint with valid information")
	response := tstPerformPost(url, tstBuildValidWebhookRequest(), tstNoToken())

	docs.Then("then the request is successful")
	require.Equal(t, http.StatusOK, response.status)

	docs.Then("and the expected downstream requests have been made to the nexi api")
	tstRequireNexiRecording(t,
		"QueryPaymentLink 42",
	)

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

	if len(expectedProtocol) == 0 {
		docs.Then("and no protocol entries have been written")
	} else {
		docs.Then("and the expected protocol entries have been written")
	}
	tstRequireProtocolEntries(t, expectedProtocol...)
}
