package nexi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"

	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/go-http-utils/headers"
)

type Impl struct {
	client       aurestclientapi.Client
	baseUrl      string
	instanceName string
}

func requestManipulator(ctx context.Context, r *http.Request) {
	// New Nexi API uses JSON and headers
	r.Header.Set(headers.ContentType, aurestclientapi.ContentTypeApplicationJson)
	merchantID := config.NexiMerchantID()
	apiKey := config.NexiAPIKey()
	if merchantID != "" && apiKey != "" {
		credentials := merchantID + ":" + apiKey
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		r.Header.Set(headers.Authorization, "Basic "+encoded)
	}
}

func newClient() (NexiDownstream, error) {
	httpClient, err := auresthttpclient.New(0, nil, requestManipulator)
	if err != nil {
		return nil, err
	}

	requestLoggingClient := aurestlogging.New(httpClient)

	circuitBreakerClient := aurestbreaker.New(requestLoggingClient,
		"nexi-downstream-breaker",
		10,
		2*time.Minute,
		30*time.Second,
		15*time.Second,
	)

	return &Impl{
		client:       circuitBreakerClient,
		baseUrl:      config.NexiDownstreamBaseUrl(),
		instanceName: config.NexiInstanceName(),
	}, nil
}

func NewTestingClient(verifierClient aurestclientapi.Client) NexiDownstream {
	return &Impl{
		client:       verifierClient,
		baseUrl:      config.NexiDownstreamBaseUrl(),
		instanceName: config.NexiInstanceName(),
	}
}

type NexiCreateLowlevelResponseBody struct {
	PaymentId            string `json:"paymentId"`
	HostedPaymentPageUrl string `json:"hostedPaymentPageUrl"`
}

func (i *Impl) CreatePaymentLink(ctx context.Context, request NexiCreateCheckoutSessionRequest) (NexiCreateCheckoutSessionResponse, error) {
	requestUrl := fmt.Sprintf("%s/v2/payments/sessions", i.baseUrl)
	requestBody, err := json.Marshal(request)
	if err != nil {
		return NexiCreateCheckoutSessionResponse{}, fmt.Errorf("failed to marshal request: %v", err)
	}
	if config.LogFullRequests() {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: request.TransId,
			Kind:        "raw",
			Message:     "nexi create request",
			Details:     string(requestBody),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		aulogging.Logger.Ctx(ctx).Info().Print("nexi create request: " + string(requestBody))
	}
	var responseRaw *[]byte
	response := aurestclientapi.ParsedResponse{
		Body: &responseRaw,
	}
	if err := i.client.Perform(ctx, http.MethodPost, requestUrl, string(requestBody), &response); err != nil {
		return NexiCreateCheckoutSessionResponse{}, err
	}
	if responseRaw == nil {
		return NexiCreateCheckoutSessionResponse{}, fmt.Errorf("response body is empty")
	}
	if response.Status >= 300 {
		if config.LogFullRequests() {
			db := database.GetRepository()
			bodyStr := string(*responseRaw)
			bodyStr = strings.ReplaceAll(bodyStr, "\r", "")
			bodyStr = strings.ReplaceAll(bodyStr, "\n", "")
			bodyStr = strings.ReplaceAll(bodyStr, " ", "")
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: request.TransId,
				Kind:        "raw",
				Message:     "nexi create error response",
				Details:     bodyStr,
				RequestId:   ctxvalues.RequestId(ctx),
			})
			aulogging.Logger.Ctx(ctx).Info().Printf("nexi create error response (status %d): %s", response.Status, string(*responseRaw))
		}
		return NexiCreateCheckoutSessionResponse{}, fmt.Errorf("unexpected response status %d", response.Status)
	}
	responseBody := NexiCreateCheckoutSessionResponse{}
	if err := json.Unmarshal(*responseRaw, &responseBody); err != nil {
		return NexiCreateCheckoutSessionResponse{}, fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	if config.LogFullRequests() {
		// Log response
		if response.Body != nil {
			aulogging.Logger.Ctx(ctx).Info().Print("nexi create success response: " + string(*responseRaw))
			// Also write to protocol with ApiId
			db := database.GetRepository()
			bodyStr := string(*responseRaw)
			bodyStr = strings.ReplaceAll(bodyStr, "\r", "")
			bodyStr = strings.ReplaceAll(bodyStr, "\n", "")
			bodyStr = strings.ReplaceAll(bodyStr, " ", "")
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: request.TransId,
				ApiId:       responseBody.PaymentId,
				Kind:        "raw",
				Message:     "nexi create success response",
				Details:     bodyStr,
				RequestId:   ctxvalues.RequestId(ctx),
			})
		}
	}
	return NexiCreateCheckoutSessionResponse{
		ID:   responseBody.PaymentId,
		Link: responseBody.HostedPaymentPageUrl,
	}, nil
}

func (i *Impl) QueryPaymentLink(ctx context.Context, paymentId string) (NexiPaymentQueryResponse, error) {
	requestUrl := fmt.Sprintf("%s/v1/payments/%s", i.baseUrl, paymentId)
	response := aurestclientapi.ParsedResponse{
		Body: &NexiPaymentQueryResponse{},
	}
	if err := i.client.Perform(ctx, http.MethodGet, requestUrl, "", &response); err != nil {
		return NexiPaymentQueryResponse{}, err
	}
	if response.Status >= 300 {
		return NexiPaymentQueryResponse{}, fmt.Errorf("unexpected response status %d", response.Status)
	}
	result := response.Body.(*NexiPaymentQueryResponse)
	return *result, nil
}

func determineStatusFromSummary(summary NexiSummary) string {
	// Simple status determination based on amounts
	if summary.ChargedAmount > 0 {
		return "confirmed"
	} else if summary.CancelledAmount > 0 {
		return "cancelled"
	} else {
		return "waiting"
	}
}

func parseCreatedDate(created string) int64 {
	// Parse ISO date to int64 timestamp
	if t, err := time.Parse(time.RFC3339, created); err == nil {
		return t.Unix()
	}
	return 0
}

// delete does not return a response body?
type deleteLowlevelResponseBody struct {
	Status string `json:"status"`
}

func (i *Impl) DeletePaymentLink(ctx context.Context, paymentId string, amount int32) error {
	requestUrl := fmt.Sprintf("%s/v1/payments/%s/cancels", i.baseUrl, paymentId)
	cancelPayload := map[string]int32{"amount": amount}
	requestBody, err := json.Marshal(cancelPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal cancel request: %v", err)
	}
	response := aurestclientapi.ParsedResponse{
		Body: &map[string]interface{}{}, // cancels returns errors or empty
	}
	if err := i.client.Perform(ctx, http.MethodPost, requestUrl, string(requestBody), &response); err != nil {
		return err
	}
	if response.Status >= 300 {
		return fmt.Errorf("unexpected response status %d", response.Status)
	}
	return nil
}

func (i *Impl) QueryTransactions(ctx context.Context, timeGreaterThan time.Time, timeLessThan time.Time) ([]TransactionData, error) {
	//TODO implement me
	panic("implement me")
}
