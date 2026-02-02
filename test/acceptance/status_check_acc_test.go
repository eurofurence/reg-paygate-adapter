package acceptance

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/eurofurence/reg-paygate-adapter/docs"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/mailservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
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

func TestStatusCheck_NonPaygateOKNoChange(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a transaction in status pending and matching payment in status AUTHORIZED")
	id := "EF1995-000001-230001-122218-5555" // set up in paygate mock as AUTHORIZED 390.00 EUR
	_, payment := tstInjectCreditPaymentTransaction(t, id, 39000, "pending")

	docs.When("when a status check is triggered")
	response := tstTriggerStatusCheck(t, id, tstValidApiToken())

	docs.Then("then the request is successful")
	tstRequirePaymentResponse(t, response, http.StatusOK, payment)

	docs.Then("and no unexpected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "success",
		Message:     "get-payment",
		Details:     "",
	})

	docs.Then("and no error notification emails have been sent")
	tstRequireMailServiceRecording(t, nil)

	docs.Then("and the transaction is unchanged")
	tstRequirePaymentServiceRecording(t, nil)
}

func TestStatusCheck_Error_PaygateOKOnValid(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a transaction in status valid and matching payment in status OK")
	id := "EF1995-000001-221216-122218-4132" // set up in paygate mock as OK 185.00 EUR
	_, payment := tstInjectCreditPaymentTransaction(t, id, 18500, "valid")

	docs.When("when a status check is triggered")
	response := tstTriggerStatusCheck(t, id, tstValidApiToken())

	docs.Then("then the request fails with the expected error")
	tstRequireErrorResponse(t, response, http.StatusConflict, "payment.update.conflict", "transaction status blocks update")

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "success",
		Message:     "get-payment",
		Details:     "",
	}, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "warning",
		Message:     "status-check: payment in status valid - skipping update",
		Details:     "transaction_status=valid upstream_status=OK",
	})

	docs.Then("and the expected error notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{
		{
			CommonID: "payment-nexi-adapter-error",
			Lang:     "en-US",
			To: []string{
				"errors@example.com",
			},
			Variables: map[string]string{
				"status":      "abort-update-for-valid-OK",
				"operation":   "status-check",
				"referenceId": id,
			},
			Async: true,
		},
	})

	docs.Then("and the transaction is unchanged")
	tstRequirePaymentServiceRecording(t, nil)
}

func TestStatusCheck_Error_PaygateOKOnDifferences(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a transaction in status pending and non-matching payment in status OK")
	id := "EF1995-000001-221216-122218-4132" // set up in paygate mock as OK 185.00 EUR
	_, payment := tstInjectCreditPaymentTransaction(t, id, 20500, "pending")

	docs.When("when a status check is triggered")
	response := tstTriggerStatusCheck(t, id, tstValidApiToken())

	docs.Then("then the request fails with the expected error")
	tstRequireErrorResponse(t, response, http.StatusConflict, "payment.update.conflict", "transaction data mismatch")

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "success",
		Message:     "get-payment",
		Details:     "",
	}, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "warning",
		Message:     "status-check: amount or currency differs - skipping update",
		Details:     "tx_amount=20500 upstream_amount=18500 tx_currency=EUR upstream_currency=EUR transaction_status=pending upstream_status=OK",
	})

	docs.Then("and the expected error notification emails have been sent")
	tstRequireMailServiceRecording(t, []mailservice.MailSendDto{
		{
			CommonID: "payment-nexi-adapter-error",
			Lang:     "en-US",
			To: []string{
				"errors@example.com",
			},
			Variables: map[string]string{
				"status":      "abort-update-values-differ",
				"operation":   "status-check",
				"referenceId": id,
			},
			Async: true,
		},
	})

	docs.Then("and the transaction is unchanged")
	tstRequirePaymentServiceRecording(t, nil)
}

func TestStatusCheck_Success_PaygateOKOnPending(t *testing.T) {
	tstSetup(tstConfigFile)
	defer tstShutdown()

	docs.Given("given a transaction in status pending and matching payment in status OK")
	id := "EF1995-000001-221216-122218-4132" // set up in paygate mock as OK 185.00 EUR
	tx, payment := tstInjectCreditPaymentTransaction(t, id, 18500, "pending")

	docs.When("when a status check is triggered")
	response := tstTriggerStatusCheck(t, id, tstValidApiToken())

	docs.Then("then the request is successful")
	tstRequirePaymentResponse(t, response, http.StatusOK, payment)

	docs.Then("and the expected protocol entries have been written")
	tstRequireProtocolEntries(t, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "success",
		Message:     "get-payment",
		Details:     "",
	}, entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       payment.Id,
		Kind:        "success",
		Message:     "transaction updated successfully by status-check",
		Details:     "amount=18500 currency=EUR upstream=OK",
	})

	docs.Then("and no error notification emails have been sent")
	tstRequireMailServiceRecording(t, nil)

	docs.Then("and the transaction has been changed to valid")
	tx.Status = "valid"
	tx.Comment = "CC paymentId 42"
	tstRequirePaymentServiceRecording(t, []paymentservice.Transaction{tx})
}

// --- helpers ---

func tstInjectCreditPaymentTransaction(t *testing.T, refId string, amount int64, status paymentservice.TransactionStatus) (paymentservice.Transaction, nexiapi.PaymentDto) {
	t.Helper()

	injectedTx := paymentservice.Transaction{
		DebitorID: 1,
		ID:        refId,
		Type:      "payment",
		Method:    "credit",
		Amount: paymentservice.Amount{
			Currency:  "EUR",
			GrossCent: amount,
			VatRate:   19.0,
		},
		Comment:       "CC previously created",
		Status:        status,
		EffectiveDate: "2022-12-10", // may update to 2022-12-16 (mocked Now() date)
		DueDate:       "2022-12-10",
	}
	err := paymentMock.InjectTransaction(context.TODO(), injectedTx)
	require.NoError(t, err)

	payment, err := nexiMock.QueryPaymentLink(context.TODO(), refId)
	require.NoError(t, err)

	result := nexiapi.PaymentDto{
		Id:            payment.PayId,
		ReferenceId:   refId,
		AmountDue:     payment.Amount.Value,
		AmountPaid:    *payment.Amount.CapturedValue,
		Currency:      payment.Amount.Currency,
		Status:        payment.Status,
		ResponseCode:  payment.ResponseCode,
		PaymentMethod: payment.PaymentMethods.Type,
	}

	return injectedTx, result
}

func tstTriggerStatusCheck(t *testing.T, refId string, token string) tstWebResponse {
	t.Helper()

	url := fmt.Sprintf("/api/rest/v1/paylinks/%s/status-check", refId)
	return tstPerformPost(url, "", token)
}
