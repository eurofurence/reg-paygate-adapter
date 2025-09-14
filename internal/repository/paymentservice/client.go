package paymentservice

import (
	"context"
	"fmt"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/middleware"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctxvalues"
	"net/http"
	"net/url"
	"time"

	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/media"
)

type Impl struct {
	client  aurestclientapi.Client
	baseUrl string
}

func requestManipulator(ctx context.Context, r *http.Request) {
	r.Header.Add(media.HeaderXApiKey, config.FixedApiToken())
	r.Header.Add(middleware.TraceIdHeader, ctxvalues.RequestId(ctx))
}

func newClient() (PaymentService, error) {
	httpClient, err := auresthttpclient.New(0, nil, requestManipulator)
	if err != nil {
		return nil, err
	}

	requestLoggingClient := aurestlogging.New(httpClient)

	circuitBreakerClient := aurestbreaker.New(requestLoggingClient,
		"payment-service-breaker",
		10,
		2*time.Minute,
		30*time.Second,
		15*time.Second,
	)

	return &Impl{
		client:  circuitBreakerClient,
		baseUrl: config.PaymentServiceBaseUrl(),
	}, nil
}

func errByStatus(err error, status int) error {
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return NotFoundError
	}
	if status >= 300 {
		return DownstreamError
	}
	return nil
}

func (i *Impl) AddTransaction(ctx context.Context, transaction Transaction) error {
	url := fmt.Sprintf("%s/api/rest/v1/transactions", i.baseUrl)
	response := aurestclientapi.ParsedResponse{}
	err := i.client.Perform(ctx, http.MethodPost, url, transaction, &response)
	return errByStatus(err, response.Status)
}

func (i *Impl) UpdateTransaction(ctx context.Context, transaction Transaction) error {
	url := fmt.Sprintf("%s/api/rest/v1/transactions/%s", i.baseUrl, url.PathEscape(transaction.ID))
	response := aurestclientapi.ParsedResponse{}
	err := i.client.Perform(ctx, http.MethodPut, url, transaction, &response)
	return errByStatus(err, response.Status)
}

func (i *Impl) GetTransactionByReferenceId(ctx context.Context, reference_id string) (Transaction, error) {
	url := fmt.Sprintf("%s/api/rest/v1/transactions?transaction_identifier=%s", i.baseUrl, url.QueryEscape(reference_id))
	bodyDto := TransactionResponse{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, url, nil, &response)

	err = errByStatus(err, response.Status)
	if len(bodyDto.Payload) == 0 {
		if err == nil {
			err = NotFoundError
		}
		return Transaction{}, err
	}
	return bodyDto.Payload[0], err
}
