package acceptance

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/eurofurence/reg-paygate-adapter/docs"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/attendeeservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/mailservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/stretchr/testify/require"
)

// --- create ---

func TestCreatePaylink_Success(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()
	tstClearDatabase() // ensure clean database for protocol entry checks

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to create a payment link with valid information")
	requestBody := tstBuildValidPaymentLinkRequest()
	response := tstPerformPost("/api/rest/v1/paylinks", tstRenderJson(requestBody), token)

	docs.Then("then the request is successful and the response is as expected")
	tstRequirePaymentLinkResponse(t, response, http.StatusCreated, tstBuildValidPaymentLink())

	docs.Then("and the expected request for a payment link has been made")
	tstRequireNexiRecording(t,
		"CreatePaymentLink",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		Kind:        "success",
		Message:     "create-pay-link",
		Details:     "http://localhost:1111/some/paylink/EF1995-000001-221216-122218-4132",
	})
}

func TestCreatePaylink_InvalidJson(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to create a payment link but supply an invalid json body")
	response := tstPerformPost("/api/rest/v1/paylinks", "{{{::}{}{}", token)

	docs.Then("then the request is denied with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "paylink.parse.error", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestCreatePaylink_ValidJsonWrongFields(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to create a payment link but supply a json body with the wrong fields")
	response := tstPerformPost("/api/rest/v1/paylinks", `{"hello":"kitty"}`, token)

	docs.Then("then the request is denied with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "paylink.parse.error", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestCreatePaylink_InvalidData(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to create a payment link but supply invalid field values")
	requestBody := nexiapi.PaymentLinkRequestDto{
		DebitorId: 0,
		AmountDue: -53,
		Currency:  "CHF",
		VatRate:   -33.3,
	}
	response := tstPerformPost("/api/rest/v1/paylinks", tstRenderJson(requestBody), token)

	docs.Then("then the request is denied with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "paylink.data.invalid", url.Values{
		"amount_due": []string{"must be a positive integer (the amount to bill)"},
		"currency":   []string{"right now, only EUR is supported"},
		"debitor_id": []string{"field must be a positive integer (the badge number to bill for)"},
		"vat_rate":   []string{"vat rate should be provided in percent and must be between 0.0 and 50.0"},
	})

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestCreatePaylink_Anonymous(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated caller")
	token := tstNoToken()

	docs.When("when they attempt to create a payment link")
	requestBody := tstBuildValidPaymentLinkRequest()
	response := tstPerformPost("/api/rest/v1/paylinks", tstRenderJson(requestBody), token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestCreatePaylink_WrongToken(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a wrong api token")
	token := tstInvalidApiToken()

	docs.When("when they attempt to create a payment link")
	requestBody := tstBuildValidPaymentLinkRequest()
	response := tstPerformPost("/api/rest/v1/paylinks", tstRenderJson(requestBody), token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "invalid api token")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestCreatePaylink_DownstreamErrorAttSrv(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to create a payment link with valid information while the attendee service is down")
	attendeeMock.SimulateGetError(attendeeservice.DownstreamError)
	requestBody := tstBuildValidPaymentLinkRequest()
	response := tstPerformPost("/api/rest/v1/paylinks", tstRenderJson(requestBody), token)

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "attsrv.downstream.error", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestCreatePaylink_DownstreamErrorCncrd(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to create a payment link with valid information while the paylink api is down")
	nexiMock.SimulateError(nexi.DownstreamError)
	requestBody := tstBuildValidPaymentLinkRequest()
	response := tstPerformPost("/api/rest/v1/paylinks", tstRenderJson(requestBody), token)

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "paylink.downstream.error", nil)

	docs.Then("and the expected email notifications have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{
		tstExpectedMailNotification("create-pay-link", "downstream unavailable - see log for details"),
	})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		ApiId:       "",
		Kind:        "error",
		Message:     "create-pay-link failed",
		Details:     "downstream unavailable - see log for details",
	})
}

// --- get ---

func TestGetPaylink_Success(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to get an existing payment link")
	response := tstPerformGet("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request is successful and the response is as expected")
	tstRequirePaymentLinkResponse(t, response, http.StatusOK, tstBuildValidPaymentLinkGetResponse())

	docs.Then("and the expected request for a payment link has been made")
	tstRequireNexiRecording(t,
		"QueryPaymentLink 42",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		ApiId:       "42",
		Kind:        "success",
		Message:     "get-pay-link",
		Details:     "http://localhost:1111/some/paylink/42",
	})
}

func TestGetPaylink_InvalidId(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to get a payment link but supply an invalid id")
	response := tstPerformGet("/api/rest/v1/paylinks/%2f%4c", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "paylink.id.invalid", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestGetPaylink_NotFound(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to get a payment link but supply an id that does not exist")
	response := tstPerformGet("/api/rest/v1/paylinks/13", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "paylink.id.notfound", nil)

	docs.Then("and the expected request for a payment link has been made")
	tstRequireNexiRecording(t,
		"QueryPaymentLink 13",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "13",
		Kind:        "error",
		Message:     "get-pay-link failed",
		Details:     "payment link id not found",
	})
}

func TestGetPaylink_Anonymous(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated caller")
	token := tstNoToken()

	docs.When("when they attempt to get a payment link")
	response := tstPerformGet("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestGetPaylink_WrongToken(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a wrong api token")
	token := tstInvalidApiToken()

	docs.When("when they attempt to get a payment link")
	response := tstPerformGet("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "invalid api token")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestGetPaylink_DownstreamError(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to get a payment link while the paylink api is down")
	nexiMock.SimulateError(nexi.DownstreamError)
	response := tstPerformGet("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "paylink.downstream.error", nil)

	docs.Then("and the expected email notifications have been sent")
	expNotif := tstExpectedMailNotification("get-pay-link", "downstream unavailable - see log for details")
	expNotif.Variables["referenceId"] = "paylink id 42"
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{expNotif})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "42",
		Kind:        "error",
		Message:     "get-pay-link failed",
		Details:     "downstream unavailable - see log for details",
	})
}

// --- delete ---

func TestDeletePaylink_Success(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to delete an existing payment link")
	response := tstPerformDelete("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request is successful and the response is as expected")
	require.Equal(t, http.StatusNoContent, response.status)
	require.Equal(t, "", response.body)

	docs.Then("and the expected request for payment link deletion has been made")
	tstRequireNexiRecording(t,
		"QueryPaymentLink 42",
		"DeletePaymentLink 42",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "42",
		Kind:        "success",
		Message:     "delete-pay-link",
		Details:     "",
	})
}

func TestDeletePaylink_InvalidId(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to delete a payment link but supply an invalid id")
	response := tstPerformDelete("/api/rest/v1/paylinks/%2f%4c", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusBadRequest, "paylink.id.invalid", nil)

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestDeletePaylink_NotFound(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to delete a payment link but supply an id that does not exist")
	response := tstPerformDelete("/api/rest/v1/paylinks/13", token)

	docs.Then("then the request fails with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusNotFound, "paylink.id.notfound", nil)

	docs.Then("and the expected request for payment link deletion has been made")
	tstRequireNexiRecording(t,
		"QueryPaymentLink 13",
	)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "13",
		Kind:        "error",
		Message:     "delete-pay-link failed",
		Details:     "payment link id not found",
	})
}

func TestDeletePaylink_Anonymous(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given an unauthenticated caller")
	token := tstNoToken()

	docs.When("when they attempt to delete a payment link")
	response := tstPerformDelete("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "you must be logged in for this operation")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestDeletePaylink_WrongToken(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a wrong api token")
	token := tstInvalidApiToken()

	docs.When("when they attempt to delete a payment link")
	response := tstPerformDelete("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request is denied as unauthenticated (401) with the appropriate error message")
	tstRequireErrorResponse(t, response, http.StatusUnauthorized, "auth.unauthorized", "invalid api token")

	docs.Then("and no requests to the payment provider have been made")
	require.Empty(t, nexiMock.Recording())

	docs.Then("and no protocol entries have been written")
	tstRequireProtocolEntries(t)
}

func TestDeletePaylink_DownstreamError(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a caller who supplies a correct api token")
	token := tstValidApiToken()

	docs.When("when they attempt to delete a payment link while the paylink api is down")
	nexiMock.SimulateError(nexi.DownstreamError)
	response := tstPerformDelete("/api/rest/v1/paylinks/42", token)

	docs.Then("then the request fails with the appropriate error")
	tstRequireErrorResponse(t, response, http.StatusBadGateway, "paylink.downstream.error", nil)

	docs.Then("and the expected email notifications have been sent")
	expNotif := tstExpectedMailNotification("delete-pay-link", "downstream unavailable - see log for details")
	expNotif.Variables["referenceId"] = "paylink id 42"
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{expNotif})

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "42",
		Kind:        "error",
		Message:     "delete-pay-link failed",
		Details:     "downstream unavailable - see log for details",
	})
}
