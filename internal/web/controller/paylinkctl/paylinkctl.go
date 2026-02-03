package paylinkctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/attendeeservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/service/paymentlinksrv"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctlutil"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
)

var paymentLinkService paymentlinksrv.PaymentLinkService

var refIdRegex *regexp.Regexp

func Create(server chi.Router, paymentLinkSrv paymentlinksrv.PaymentLinkService) {
	paymentLinkService = paymentLinkSrv

	server.Post("/api/rest/v1/paylinks", createPaylinkHandler)
	server.Get("/api/rest/v1/paylinks/{refid}", getPaymentHandler)
	server.Post("/api/rest/v1/paylinks/{refid}/status-check", checkPaymentStatusHandler)

	refIdRegex = regexp.MustCompile("^[A-Z0-9][A-Z0-9-]+[A-Z0-9]$")
}

func createPaylinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ctxvalues.HasApiToken(ctx) {
		ctlutil.UnauthenticatedError(ctx, w, r, "you must be logged in for this operation", "anonymous access attempt")
		return
	}

	request, err := parseBodyToPaymentLinkRequestDto(ctx, w, r)
	if err != nil {
		return
	}

	errs := paymentLinkService.ValidatePaymentLinkRequest(ctx, request)
	if errs != nil {
		paylinkRequestInvalidErrorHandler(ctx, w, r, errs)
		return
	}

	dto, id, err := paymentLinkService.CreatePaymentLink(ctx, request)
	if err != nil {
		if errors.Is(err, nexi.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, "paylink", err)
		} else if errors.Is(err, attendeeservice.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, "attsrv", err)
		} else {
			ctlutil.UnexpectedError(ctx, w, r, err)
		}
		return
	}

	w.Header().Set(headers.Location, fmt.Sprintf("/api/rest/v1/paylinks/%s", id))
	w.WriteHeader(http.StatusCreated)
	ctlutil.WriteJson(ctx, w, dto)
}

func getPaymentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ctxvalues.HasApiToken(ctx) {
		ctlutil.UnauthenticatedError(ctx, w, r, "you must be logged in for this operation", "anonymous access attempt")
		return
	}

	id, err := refidFromVars(ctx, w, r)
	if err != nil {
		return
	}

	dto, err := paymentLinkService.GetPayment(ctx, id)
	if err != nil {
		if errors.Is(err, nexi.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, "paylink", err)
		} else if errors.Is(err, nexi.NoSuchID404Error) {
			paymentNotFoundErrorHandler(ctx, w, r, id)
		} else {
			ctlutil.UnexpectedError(ctx, w, r, err)
		}
		return
	}

	ctlutil.WriteJson(ctx, w, dto)
}

func checkPaymentStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ctxvalues.HasApiToken(ctx) {
		ctlutil.UnauthenticatedError(ctx, w, r, "you must be logged in for this operation", "anonymous access attempt")
		return
	}

	id, err := refidFromVars(ctx, w, r)
	if err != nil {
		return
	}

	dto, err := paymentLinkService.CheckPaymentStatus(ctx, id)
	if err != nil {
		if errors.Is(err, nexi.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, "paylink", err)
		} else if errors.Is(err, nexi.NoSuchID404Error) {
			paymentNotFoundErrorHandler(ctx, w, r, id)
		} else if errors.Is(err, nexi.NotConfigured) {
			downstreamNotConfiguredErrorHandler(ctx, w, r, "paylink", err)
		} else if errors.Is(err, paymentservice.NotFoundError) {
			paymentNotFoundErrorHandler(ctx, w, r, id)
		} else if errors.Is(err, paymentlinksrv.TransactionStatusError) || errors.Is(err, paymentlinksrv.TransactionDataMismatchError) {
			cannotUpdatePaymentErrorHandler(ctx, w, r, id, err)
		} else {
			ctlutil.UnexpectedError(ctx, w, r, err)
		}
		return
	}

	ctlutil.WriteJson(ctx, w, dto)
}

func parseBodyToPaymentLinkRequestDto(ctx context.Context, w http.ResponseWriter, r *http.Request) (nexiapi.PaymentLinkRequestDto, error) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	dto := nexiapi.PaymentLinkRequestDto{}
	err := decoder.Decode(&dto)
	if err != nil {
		paylinkRequestParseErrorHandler(ctx, w, r, err)
	}
	return dto, err
}

func refidFromVars(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	idStr := chi.URLParam(r, "refid")
	// minimal validation to make sure the downstream api request will be valid
	if !refIdRegex.MatchString(idStr) || !strings.HasPrefix(idStr, config.TransactionIDPrefix()) || len(idStr) > 63 {
		invalidPaymentRefIdErrorHandler(ctx, w, r, idStr)
		return "", fmt.Errorf("invalid or empty id")
	}
	return idStr, nil
}

func paylinkRequestParseErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("paylink body could not be parsed: %s", err.Error())
	ctlutil.ErrorHandler(ctx, w, r, "paylink.parse.error", http.StatusBadRequest, nil)
}

func paylinkRequestInvalidErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, validationErrors url.Values) {
	// validation already logged each individual error
	ctlutil.ErrorHandler(ctx, w, r, "paylink.data.invalid", http.StatusBadRequest, validationErrors)
}

func downstreamErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, sysname string, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("%s downstream error: %s", sysname, err.Error())
	ctlutil.ErrorHandler(ctx, w, r, fmt.Sprintf("%s.downstream.error", sysname), http.StatusBadGateway, nil)
}

func downstreamNotConfiguredErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, sysname string, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("%s downstream not configured error: %s", sysname, err.Error())
	ctlutil.ErrorHandler(ctx, w, r, fmt.Sprintf("%s.downstream.noconfig", sysname), http.StatusBadGateway, nil)
}

func invalidPaymentRefIdErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	aulogging.Logger.Ctx(ctx).Warn().Printf("received invalid paylink id '%s'", url.QueryEscape(id))
	ctlutil.ErrorHandler(ctx, w, r, "payment.refid.invalid", http.StatusBadRequest, url.Values{})
}

func paymentNotFoundErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	aulogging.Logger.Ctx(ctx).Warn().Printf("paylink id %s not found", id)
	ctlutil.ErrorHandler(ctx, w, r, "payment.refid.notfound", http.StatusNotFound, url.Values{})
}

func cannotUpdatePaymentErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id string, err error) {
	aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("cannot update payment %s error: %s", id, err.Error())
	ctlutil.ErrorHandler(ctx, w, r, "payment.update.conflict", http.StatusConflict, url.Values{"details": {err.Error()}})
}
