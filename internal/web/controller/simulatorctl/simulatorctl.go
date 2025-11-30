// Package simulatorctl implements a local simulator for the api.
//
// If you do not set the url of the payment provider in the configuration, this will be what the service
// talks to instead. Generated simulated pay links will be handled by this implementation as well, and
// lead to the expected webhook call.
package simulatorctl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/self"
	"github.com/eurofurence/reg-paygate-adapter/internal/service/paymentlinksrv"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/media"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
	"github.com/google/uuid"
)

var paymentLinkService paymentlinksrv.PaymentLinkService

func Create(server chi.Router, paymentLinkSrv paymentlinksrv.PaymentLinkService) {
	paymentLinkService = paymentLinkSrv
	server.Get("/simulator/{id}", useSimulator)
}

func useSimulator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := idFromVars(ctx, w, r)
	if err != nil {
		return
	}

	paylink, err := paymentLinkService.GetPaymentLink(ctx, id)
	if err != nil {
		errorHandler(ctx, w, http.StatusNotFound, "not found", fmt.Sprintf("simulator paylink with id %s not found (may be lost from memory after restart)", id))
		return
	}

	mock := nexi.Get().(nexi.Mock)
	tx := nexi.TransactionData{
		UUID:        uuid.New().String(),
		Amount:      paylink.AmountDue,
		Status:      "confirmed",
		Time:        "2022-10-15 15:50:20",
		Lang:        "DEU",
		PageUUID:    "",
		Payment:     nexi.Payment{Brand: "visa"},
		Psp:         "ConCardis_PayEngine_3",
		PspID:       29,
		Mode:        "TEST",
		ReferenceID: paylink.ReferenceId,
		Invoice: nexi.Invoice{
			Currency: "EUR",
		},
	}
	mock.InjectTransaction(tx)

	selfCaller := self.Get()
	// For simulator, use a dummy id since webhook expects int
	var paymentReqId int64 = 42
	if parsed, err := strconv.ParseInt(id, 10, 64); err == nil {
		paymentReqId = parsed
	}
	event := nexiapi.WebhookEventDto{
		Transaction: nexiapi.WebhookEventTransaction{
			Id: tx.ID,
			Invoice: nexiapi.WebhookEventTransactionInvoice{
				ReferenceId:      paylink.ReferenceId,
				PaymentRequestId: paymentReqId,
			},
		},
	}
	err = selfCaller.CallWebhook(ctx, event)
	if err != nil {
		errorHandler(ctx, w, http.StatusInternalServerError,
			"failed to report to local webhook - see log for details",
			fmt.Sprintf("failed to report %s to webhook: %s", paylink.ReferenceId, err.Error()),
		)
		return
	}

	successHandler(ctx, w,
		fmt.Sprintf("paid refId %s for %0.2f %s", paylink.ReferenceId, float64(tx.Amount)/100.0, tx.Invoice.Currency),
		fmt.Sprintf("simulator paid refId %s for %0.2f %s", paylink.ReferenceId, float64(tx.Amount)/100.0, tx.Invoice.Currency),
	)
}

func idFromVars(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		errorHandler(ctx, w, http.StatusBadRequest, "bad id", fmt.Sprintf("simulator received invalid id '%s'", url.QueryEscape(idStr)))
		return "", fmt.Errorf("empty id")
	}
	return idStr, nil
}

func successHandler(ctx context.Context, w http.ResponseWriter, message string, logmessage string) {
	aulogging.Logger.Ctx(ctx).Info().Print(logmessage)
	w.Header().Set(headers.ContentType, media.ContentTypeTextPlain)
	_, _ = w.Write([]byte(message))
}

func errorHandler(ctx context.Context, w http.ResponseWriter, status int, message string, logmessage string) {
	aulogging.Logger.Ctx(ctx).Warn().Print(logmessage)
	w.Header().Set(headers.ContentType, media.ContentTypeTextPlain)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}
