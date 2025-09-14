package ctlutil

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctxvalues"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/media"
	"github.com/go-http-utils/headers"
	"net/http"
	"net/url"
	"time"
)

// --- common error handlers ---

// note, remember to bail out after calling these

func UnauthenticatedError(ctx context.Context, w http.ResponseWriter, r *http.Request, details string, logMessage string) {
	aulogging.Logger.Ctx(ctx).Warn().Print(logMessage)
	ErrorHandler(ctx, w, r, "auth.unauthorized", http.StatusUnauthorized, url.Values{"details": []string{details}})
}

func UnauthorizedError(ctx context.Context, w http.ResponseWriter, r *http.Request, details string, logMessage string) {
	aulogging.Logger.Ctx(ctx).Warn().Print(logMessage)
	ErrorHandler(ctx, w, r, "auth.forbidden", http.StatusForbidden, url.Values{"details": []string{details}})
}

func UnexpectedError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("unexpected error: %s", err.Error())
	ErrorHandler(ctx, w, r, "unexpected", http.StatusInternalServerError, nil)
}

func ErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, msg string, status int, details url.Values) {
	timestamp := time.Now().Format(time.RFC3339)
	response := nexiapi.ErrorDto{Message: msg, Timestamp: timestamp, Details: details, RequestId: ctxvalues.RequestId(ctx)}
	w.Header().Set(headers.ContentType, media.ContentTypeApplicationJson)
	w.WriteHeader(status)
	WriteJson(ctx, w, response)
}
