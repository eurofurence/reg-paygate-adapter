package acceptance

import (
	"net/http"
	"testing"

	"github.com/eurofurence/reg-paygate-adapter/docs"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/mailservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/stretchr/testify/require"
)

// --- status check ---

func TestStatusCheck_InvalidId(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to trigger a status check, but supply an invalid reference id")
	response := tstPerformPost("/api/rest/v1/paylinks/%2f%4c/status-check", "", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "payment.refid.invalid", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)

	docs.Then("and no error mails have been sent")
	tstRequireMailServiceRecording(t, nil)
}

func TestStatusCheck_WrongPrefix(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to trigger a status check, but supply reference id with wrong prefix")
	response := tstPerformPost("/api/rest/v1/paylinks/EF2022/status-check", "", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "payment.refid.invalid", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)

	docs.Then("and no error mails have been sent")
	tstRequireMailServiceRecording(t, nil)
}

func TestStatusCheck_NexiNotFound(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to get a payment link but supply an id that does not exist in paygate")
	response := tstPerformPost("/api/rest/v1/paylinks/EF1995-000017-000000-000000-0000/status-check", "", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "payment.refid.notfound", nil)

	docs.Then("and the expected request for a payment has been made")
	tstRequireNexiRecording(t,
		"QueryPaymentLink EF1995-000017-000000-000000-0000",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "EF1995-000017-000000-000000-0000",
		ApiId:       "",
		Kind:        "error",
		Message:     "get-payment failed",
		Details:     "payment link id not found",
	})
}

func TestStatusCheck_PaySrvNotFound(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to get a payment link but supply an id that does not exist in the payment service")
	response := tstPerformPost("/api/rest/v1/paylinks/EF1995-000001-230001-122218-5555/status-check", "", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "payment.refid.notfound", "")

	docs.Then("and the expected request for a payment has been made")
	tstRequireNexiRecording(t,
		"QueryPaymentLink EF1995-000001-230001-122218-5555",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-230001-122218-5555",
		ApiId:       "4242",
		Kind:        "success",
		Message:     "get-payment",
		Details:     "",
	})
}

func TestStatusCheck_Anonymous(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated caller")
	token := tstNoToken()

	docs.When("when they attempt to trigger a status check for an existing payment")
	response := tstPerformPost("/api/rest/v1/paylinks/EF1995-000001-221216-122218-4132/status-check", "", token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestStatusCheck_WrongToken(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a wrong api token")
	token := tstInvalidApiToken()

	docs.When("when they attempt to trigger a status check for an existing payment")
	response := tstPerformPost("/api/rest/v1/paylinks/EF1995-000001-221216-122218-4132/status-check", "", token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "invalid api token")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestStatusCheck_DownstreamError(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to trigger a status check while the paygate api is down")
	nexiMock.SimulateError(nexi.DownstreamError)
	response := tstPerformPost("/api/rest/v1/paylinks/EF1995-000001-221216-122218-4132/status-check", "", token)

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "paylink.downstream.error", nil)

	docs.Then("and the expected email notifications have been sent")
	expNotif := tstExpectedMailNotification("get-payment", "downstream unavailable - see log for details")
	expNotif.Variables["referenceId"] = "reference id EF1995-000001-221216-122218-4132"
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{expNotif})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		ApiId:       "",
		Kind:        "error",
		Message:     "get-payment failed",
		Details:     "downstream unavailable - see log for details",
	})
}
