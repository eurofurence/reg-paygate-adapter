package webhookctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/service/paymentlinksrv"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctlutil"
	"github.com/go-chi/chi/v5"
)

var paymentLinkService paymentlinksrv.PaymentLinkService

func Create(server chi.Router, paymentLinkSrv paymentlinksrv.PaymentLinkService) {
	paymentLinkService = paymentLinkSrv

	server.Post("/api/rest/v1/webhook/{secret}", webhookHandler)
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !secretFromVarsOk(ctx, w, r) {
		ctlutil.UnauthenticatedError(ctx, w, r, "invalid secret supplied", "invalid secret for webhook")
		return
	}

	request, err := parseBodyToWebhookDtoTolerant(ctx, w, r)
	if err != nil {
		return
	}

	err = paymentLinkService.HandleWebhook(ctx, request)
	if err != nil {
		if errors.Is(err, paymentlinksrv.WebhookValidationErr) {
			webhookRequestInvalidErrorHandler(ctx, w, r, err)
		} else if errors.Is(err, nexi.NoSuchID404Error) {
			paylinkNotFoundErrorHandler(ctx, w, r)
		} else if errors.Is(err, nexi.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, err)
		} else {
			ctlutil.UnexpectedError(ctx, w, r, err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func parseBodyToWebhookDtoTolerant(ctx context.Context, w http.ResponseWriter, r *http.Request) (nexiapi.WebhookDto, error) {
	dto := nexiapi.WebhookDto{}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		webhookRequestParseErrorHandler(ctx, w, r, err)
		return dto, err
	}

	if config.LogFullRequests() {
		bodyStr := string(bodyBytes)
		bodyStr = strings.ReplaceAll(bodyStr, "\r", "")
		bodyStr = strings.ReplaceAll(bodyStr, "\n", "\\n")
		aulogging.Logger.Ctx(ctx).Info().Print("webhook request: " + bodyStr)
	}

	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	err = decoder.Decode(&dto)
	if err != nil {
		webhookRequestParseErrorHandler(ctx, w, r, err)
		return dto, err
	}

	return dto, nil
}

func secretFromVarsOk(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	secretReceived := chi.URLParam(r, "secret")
	return secretReceived == config.WebhookSecret()
}

func webhookRequestParseErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("webhook body could not be parsed: %s", err.Error())
	ctlutil.ErrorHandler(ctx, w, r, "webhook.parse.error", http.StatusBadRequest, nil)
}

func webhookRequestInvalidErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("webhook data invalid: %s", err.Error())
	ctlutil.ErrorHandler(ctx, w, r, "webhook.data.invalid", http.StatusBadRequest, nil)
}

func downstreamErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("downstream error: %s", err.Error())
	ctlutil.ErrorHandler(ctx, w, r, "webhook.downstream.error", http.StatusBadGateway, nil)
}

func paylinkNotFoundErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	aulogging.Logger.Ctx(ctx).Warn().Print("paylink id not found")
	ctlutil.ErrorHandler(ctx, w, r, "paylink.id.notfound", http.StatusNotFound, nil)
}
