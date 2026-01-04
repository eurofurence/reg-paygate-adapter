// Package simulatorctl implements a local simulator for the api.
//
// If you do not set the url of the payment provider in the configuration, this will be what the service
// talks to instead. Generated simulated pay links will be handled by this implementation as well, and
// lead to the expected webhook call.
package simulatorctl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/self"
	"github.com/eurofurence/reg-paygate-adapter/internal/service/paymentlinksrv"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/media"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
)

var paymentLinkService paymentlinksrv.PaymentLinkService

func Create(server chi.Router, paymentLinkSrv paymentlinksrv.PaymentLinkService) {
	paymentLinkService = paymentLinkSrv
	server.Get("/simulator/{id}", useSimulator)
}

func useSimulator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	referenceId, err := idFromVars(ctx, w, r)
	if err != nil {
		return
	}

	mock := nexi.Get().(nexi.Mock)

	event, err := mock.GetCachedWebhook(referenceId)
	if err != nil {
		errorHandler(ctx, w, http.StatusNotFound, "not found", fmt.Sprintf("simulator paylink with id %s not found (may be lost from memory after restart)", referenceId))
		return
	}
	var data nexiapi.DataPaymentCheckoutCompleted
	err = json.Unmarshal(event.Data, &data)
	if err != nil {
		errorHandler(ctx, w, http.StatusInternalServerError,
			"failed to report to local webhook - see log for details",
			fmt.Sprintf("failed to unmarshal cached webhook for %s: %s", referenceId, err.Error()),
		)
	}

	selfCaller := self.Get()
	err = selfCaller.CallWebhook(ctx, event)
	if err != nil {
		errorHandler(ctx, w, http.StatusInternalServerError,
			"failed to report to local webhook - see log for details",
			fmt.Sprintf("failed to report %s to webhook: %s", referenceId, err.Error()),
		)
		return
	}

	successHandler(ctx, w,
		fmt.Sprintf("paid refId %s for %0.2f %s", referenceId, float64(data.Order.Amount.Amount)/100.0, data.Order.Amount.Currency),
		fmt.Sprintf("simulator paid refId %s for %0.2f %s", referenceId, float64(data.Order.Amount.Amount)/100.0, data.Order.Amount.Currency),
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
