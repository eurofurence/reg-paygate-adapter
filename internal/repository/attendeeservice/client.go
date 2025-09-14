package attendeeservice

import (
	"context"
	"fmt"
	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/middleware"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctxvalues"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/media"
	"net/http"
	"time"
)

type Impl struct {
	client  aurestclientapi.Client
	baseUrl string
}

func requestManipulator(ctx context.Context, r *http.Request) {
	r.Header.Add(media.HeaderXApiKey, config.FixedApiToken())
	r.Header.Add(middleware.TraceIdHeader, ctxvalues.RequestId(ctx))
}

func newClient() (AttendeeService, error) {
	httpClient, err := auresthttpclient.New(0, nil, requestManipulator)
	if err != nil {
		return nil, err
	}

	requestLoggingClient := aurestlogging.New(httpClient)

	circuitBreakerClient := aurestbreaker.New(requestLoggingClient,
		"attendee-service-breaker",
		10,
		2*time.Minute,
		30*time.Second,
		15*time.Second,
	)

	return &Impl{
		client:  circuitBreakerClient,
		baseUrl: config.AttendeeServiceBaseUrl(),
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

func (i *Impl) GetAttendee(ctx context.Context, id uint) (AttendeeDto, error) {
	url := fmt.Sprintf("%s/api/rest/v1/attendees/%d", i.baseUrl, id)
	bodyDto := AttendeeDto{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, url, nil, &response)

	err = errByStatus(err, response.Status)
	return bodyDto, err
}
