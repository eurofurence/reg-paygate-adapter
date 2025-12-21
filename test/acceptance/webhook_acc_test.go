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
		ReferenceId: "221216-122218-000001",
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
		[]paymentservice.Transaction{},
		[]mailservice.MailSendDto{},
		[]entity.ProtocolEntry{
			{
				ReferenceId: "221216-122218-000001",
				ApiId:       "ef00000000000000000000000000cafe",
				Kind:        "success",
				Message:     "webhook payment.created",
				Details:     "amount=18500 currency=EUR",
			},
		},
	)
}

func TestWebhook_Success_ChargeCreated(t *testing.T) {
	docs.Description("payment.charge.created.v2 events are logged but otherwise ignored")
	tstWebhookSuccessCase(t, "payment.charge.created.v2",
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

func TestWebhook_Success_Status_Ignored(t *testing.T) {
	t.Skip("Skipping webhook tests for the moment")

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
	t.Skip("Skipping webhook tests for the moment")

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
	t.Skip("Skipping webhook tests for the moment")

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
	t.Skip("Skipping webhook tests for the moment")

	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who does not know the secret url")
	url := "/api/rest/v1/webhook/wrongsecret"

	docs.When("when they attempt to trigger our webhook endpoint")
	response := tstPerformPost(url, tstBuildValidWebhookRequest(t, "payment.created"), tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", nil)
}

func TestWebhook_DownstreamError(t *testing.T) {
	t.Skip("Skipping webhook tests for the moment")

	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.When("when they attempt to trigger our webhook endpoint while the downstream api is down")
	nexiMock.SimulateError(nexi.DownstreamError)
	response := tstPerformPost(url, tstBuildValidWebhookRequest(t, "payment.created"), tstNoToken())

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "webhook.downstream.error", nil)
}

func TestWebhook_Success_Status_WrongPrefix(t *testing.T) {
	t.Skip("Skipping webhook tests for the moment")

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

func tstWebhookSuccessCase(t *testing.T, event string, expectedPaymentServiceRecording []paymentservice.Transaction, expectedMailRecording []mailservice.MailSendDto, expectedProtocol []entity.ProtocolEntry) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an anonymous caller who knows the secret url")
	url := "/api/rest/v1/webhook/demosecret"

	docs.Given("and the payment service has a transaction in status tentative")

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

	if len(expectedProtocol) == 0 {
		docs.Then("and no protocol entries (other than the raw request log) have been written")
	} else {
		docs.Then("and the expected protocol entries have been written")
	}
	fullExpectedProtocol := make([]entity.ProtocolEntry, len(expectedProtocol)+1)
	fullExpectedProtocol[0] = entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     request,
	}
	for i := 0; i < len(expectedProtocol); i++ {
		fullExpectedProtocol[i+1] = expectedProtocol[i]
	}
	tstRequireProtocolEntries(t, fullExpectedProtocol...)
}
