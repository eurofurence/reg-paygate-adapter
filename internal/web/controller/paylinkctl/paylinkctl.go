package paylinkctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/attendeeservice"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/service/paymentlinksrv"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctlutil"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctxvalues"
	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
)

var paymentLinkService paymentlinksrv.PaymentLinkService

func Create(server chi.Router, paymentLinkSrv paymentlinksrv.PaymentLinkService) {
	paymentLinkService = paymentLinkSrv

	server.Post("/api/rest/v1/paylinks", createPaylinkHandler)
	server.Get("/api/rest/v1/paylinks/{id}", getPaylinkHandler)
	server.Delete("/api/rest/v1/paylinks/{id}", deletePaylinkHandler)
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

func getPaylinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ctxvalues.HasApiToken(ctx) {
		ctlutil.UnauthenticatedError(ctx, w, r, "you must be logged in for this operation", "anonymous access attempt")
		return
	}

	id, err := idFromVars(ctx, w, r)
	if err != nil {
		return
	}

	dto, err := paymentLinkService.GetPaymentLink(ctx, id)
	if err != nil {
		if errors.Is(err, nexi.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, "paylink", err)
		} else if errors.Is(err, nexi.NoSuchID404Error) {
			paylinkNotFoundErrorHandler(ctx, w, r, id)
		} else {
			ctlutil.UnexpectedError(ctx, w, r, err)
		}
		return
	}

	ctlutil.WriteJson(ctx, w, dto)
}

func deletePaylinkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ctxvalues.HasApiToken(ctx) {
		ctlutil.UnauthenticatedError(ctx, w, r, "you must be logged in for this operation", "anonymous access attempt")
		return
	}

	id, err := idFromVars(ctx, w, r)
	if err != nil {
		return
	}

	err = paymentLinkService.DeletePaymentLink(ctx, id)
	if err != nil {
		if errors.Is(err, nexi.DownstreamError) {
			downstreamErrorHandler(ctx, w, r, "paylink", err)
		} else if errors.Is(err, nexi.NoSuchID404Error) {
			paylinkNotFoundErrorHandler(ctx, w, r, id)
		} else {
			ctlutil.UnexpectedError(ctx, w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func idFromVars(ctx context.Context, w http.ResponseWriter, r *http.Request) (string, error) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		invalidPaylinkIdErrorHandler(ctx, w, r, idStr)
		return "", fmt.Errorf("empty id")
	}
	// Validate that id looks like a valid uint for backward compatibility
	if _, err := strconv.ParseUint(idStr, 10, 32); err != nil {
		invalidPaylinkIdErrorHandler(ctx, w, r, idStr)
		return "", fmt.Errorf("invalid id")
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

func invalidPaylinkIdErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	aulogging.Logger.Ctx(ctx).Warn().Printf("received invalid paylink id '%s'", url.QueryEscape(id))
	ctlutil.ErrorHandler(ctx, w, r, "paylink.id.invalid", http.StatusBadRequest, url.Values{})
}

func paylinkNotFoundErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	aulogging.Logger.Ctx(ctx).Warn().Printf("paylink id %s not found", id)
	ctlutil.ErrorHandler(ctx, w, r, "paylink.id.notfound", http.StatusNotFound, url.Values{})
}
