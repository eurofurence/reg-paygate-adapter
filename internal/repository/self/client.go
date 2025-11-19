package self

import (
	"context"
	"fmt"
	"net/http"

	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
)

type Impl struct {
	client  aurestclientapi.Client
	baseUrl string
}

func newClient() (Self, error) {
	httpClient, err := auresthttpclient.New(0, nil, nil)
	if err != nil {
		return nil, err
	}

	requestLoggingClient := aurestlogging.New(httpClient)

	return &Impl{
		client:  requestLoggingClient,
		baseUrl: config.ServicePublicURL(),
	}, nil
}

func errByStatus(err error, status int) error {
	if err != nil {
		return err
	}
	if status >= 300 {
		return DownstreamError
	}
	return nil
}

func (i *Impl) CallWebhook(ctx context.Context, event nexiapi.WebhookEventDto) error {
	url := fmt.Sprintf("%s/api/rest/v1/webhook/%s", i.baseUrl, config.WebhookSecret())
	response := aurestclientapi.ParsedResponse{}
	err := i.client.Perform(ctx, http.MethodPost, url, event, &response)
	return errByStatus(err, response.Status)
}
