package self

import (
	"context"
	"errors"

	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
)

type Self interface {
	CallWebhook(ctx context.Context, event nexiapi.WebhookDto) error
}

var (
	DownstreamError = errors.New("downstream unavailable - see log for details")
)
