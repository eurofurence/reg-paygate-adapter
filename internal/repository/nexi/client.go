package nexi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/entity"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctxvalues"
	"net/http"
	"net/url"
	"strings"
	"time"

	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/go-http-utils/headers"
)

type Impl struct {
	client       aurestclientapi.Client
	baseUrl      string
	instanceName string
}

func requestManipulator(ctx context.Context, r *http.Request) {
	// even GET gets a body with the signature
	r.Header.Set(headers.ContentType, aurestclientapi.ContentTypeApplicationXWwwFormUrlencoded)
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

type createLowlevelResponseBody struct {
	Status string               `json:"status"`
	Data   []PaymentLinkCreated `json:"data"`
}

type createLowlevelRequestBody struct {
	PaymentLinkCreateRequest

	ApiSignature string `json:"ApiSignature"`
}

// FixedSignatureValue set for automated contract testing only
var FixedSignatureValue string

const signatureKey = "ApiSignature"

func pathEncode(key string, value string) string {
	return url.PathEscape(key) + "=" + url.PathEscape(value)
}

func queryEncode(key string, value string) string {
	return url.QueryEscape(key) + "=" + url.QueryEscape(value)
}

func signRequest(unsignedRequest string, instanceApiSecret string) string {
	// request parameters have to be in order for signature
	authenticator := hmac.New(sha256.New, []byte(instanceApiSecret))
	authenticator.Write([]byte(unsignedRequest))

	hashValue := authenticator.Sum([]byte{})
	signature := base64.StdEncoding.EncodeToString(hashValue)

	if FixedSignatureValue != "" {
		return FixedSignatureValue
	} else {
		return signature
	}
}

func constructBufferWithEncoding(request PaymentLinkCreateRequest, encode func(key string, value string) string) string {
	var buf strings.Builder
	buf.WriteString(encode("title", request.Title) + "&")
	buf.WriteString(encode("description", request.Description) + "&")
	buf.WriteString(encode("psp", fmt.Sprintf("%d", request.PSP)) + "&")
	buf.WriteString(encode("referenceId", request.ReferenceId) + "&")
	// leaving this one out because it leads to duplicate problems when re-trying a payment after a failure (which we can't really prevent)
	// buf.WriteString(encode("nexiOrderId", request.OrderId) + "&")
	buf.WriteString(encode("purpose", request.Purpose) + "&")
	buf.WriteString(encode("amount", fmt.Sprintf("%d", request.Amount)) + "&")
	buf.WriteString(encode("vatRate", fmt.Sprintf("%.1f", request.VatRate)) + "&")
	buf.WriteString(encode("currency", request.Currency) + "&")
	buf.WriteString(encode("sku", request.SKU) + "&")
	buf.WriteString(encode("preAuthorization", "0") + "&")
	buf.WriteString(encode("reservation", "0") + "&")
	// fields are:
	//  "title", "forename", "surname",
	//  "company", "street", "postcode", "place",
	//  "country", "phone", "email", "date_of_birth",
	//  "terms", "privacy_policy"
	// terms is selected by default if not specified
	buf.WriteString(encode("fields[email][mandatory]", "1") + "&")
	buf.WriteString(encode("fields[email][defaultValue]", request.Email))
	if request.SuccessRedirectUrl != "" {
		buf.WriteString("&" + encode("successRedirectUrl", request.SuccessRedirectUrl))
	}
	if request.FailedRedirectUrl != "" {
		buf.WriteString("&" + encode("failedRedirectUrl", request.FailedRedirectUrl))
	}
	return buf.String()
}

func buildCreateRequestBody(ctx context.Context, request PaymentLinkCreateRequest) string {
	// Note: the Nexi PayLink API uses PathEncoding for the Body,
	// but QueryEncoding to calculate the signature. (don't ask)
	// This is relevant for values with spaces, question marks, etc.
	pathEncodedPayload := constructBufferWithEncoding(request, pathEncode)
	queryEncodedPayloadForSigning := constructBufferWithEncoding(request, queryEncode)

	if config.LogFullRequests() {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: request.ReferenceId,
			Kind:        "raw",
			Message:     "nexi request",
			Details:     pathEncodedPayload,
			RequestId:   ctxvalues.RequestId(ctx),
		})
		aulogging.Logger.Ctx(ctx).Info().Print("nexi request: " + pathEncodedPayload)
	}

	signature := signRequest(queryEncodedPayloadForSigning, config.NexiInstanceApiSecret())
	return pathEncodedPayload + "&" + queryEncode(signatureKey, signature)
}

func (i *Impl) performWithRawResponseLogging(ctx context.Context, logName string, refId string, apiId uint, method string, requestUrl string, requestBody string, response *aurestclientapi.ParsedResponse) error {
	rawResponseBody := &([]byte{}) // prevent nil in case request fails before parsing response

	responseRaw := aurestclientapi.ParsedResponse{
		Body: &rawResponseBody,
	}
	if err := i.client.Perform(ctx, method, requestUrl, requestBody, &responseRaw); err != nil {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("%s http request failed: %s", logName, err.Error())
		return err
	}
	if rawResponseBody == nil || len(*rawResponseBody) == 0 {
		err := fmt.Errorf("%s failed, did not obtain a response body", logName)
		aulogging.Logger.Ctx(ctx).Warn().Print(err.Error())
		return err
	}

	if config.LogFullRequests() && len(*rawResponseBody) > 0 {
		bodyStr := string(*rawResponseBody)
		bodyStr = strings.ReplaceAll(bodyStr, "\r", "")
		bodyStr = strings.ReplaceAll(bodyStr, "\n", "\\n")
		aulogging.Logger.Ctx(ctx).Info().Print("nexi response: " + bodyStr)

		bodyStrDB := string(*rawResponseBody)
		bodyStrDB = strings.ReplaceAll(bodyStrDB, "\r", "")
		bodyStrDB = strings.ReplaceAll(bodyStrDB, "\n", "")
		bodyStrDB = strings.ReplaceAll(bodyStrDB, " ", "")
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: refId,
			ApiId:       apiId,
			Kind:        "raw",
			Message:     "nexi response",
			Details:     bodyStrDB,
			RequestId:   ctxvalues.RequestId(ctx),
		})
	}

	if err := json.Unmarshal(*rawResponseBody, response.Body); err != nil {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("%s response parse failed: %s", logName, err.Error())
		return err
	}

	response.Status = responseRaw.Status
	response.Time = responseRaw.Time
	response.Header = responseRaw.Header

	return nil
}

func (i *Impl) CreatePaymentLink(ctx context.Context, request PaymentLinkCreateRequest) (PaymentLinkCreated, error) {
	requestUrl := fmt.Sprintf("%s/v1.0/Invoice/?instance=%s", i.baseUrl, url.QueryEscape(i.instanceName))
	requestBody := buildCreateRequestBody(ctx, request)
	bodyDto := createLowlevelResponseBody{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	if err := i.performWithRawResponseLogging(ctx, "CreatePaymentLink", request.ReferenceId, 0, http.MethodPost, requestUrl, requestBody, &response); err != nil {
		return PaymentLinkCreated{}, err
	}
	if response.Status >= 300 {
		return PaymentLinkCreated{}, fmt.Errorf("unexpected response status %d", response.Status)
	}
	if bodyDto.Status != "success" {
		return PaymentLinkCreated{}, NotSuccessful
	}
	if len(bodyDto.Data) != 1 {
		return PaymentLinkCreated{}, fmt.Errorf("unexpected number of response body data array entries %d", len(bodyDto.Data))
	}
	return bodyDto.Data[0], nil
}

type queryLowlevelResponseBody struct {
	Status string                     `json:"status"`
	Data   []PaymentLinkQueryResponse `json:"data"`
}

func buildEmptyRequestBody() string {
	signature := signRequest("", config.NexiInstanceApiSecret())
	return queryEncode(signatureKey, signature)
}

func (i *Impl) QueryPaymentLink(ctx context.Context, id uint) (PaymentLinkQueryResponse, error) {
	requestUrl := fmt.Sprintf("%s/v1.0/Invoice/%d/?instance=%s", i.baseUrl, id, url.QueryEscape(i.instanceName))
	requestBody := buildEmptyRequestBody()
	bodyDto := queryLowlevelResponseBody{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	if err := i.performWithRawResponseLogging(ctx, "QueryPaymentLink", "", id, http.MethodGet, requestUrl, requestBody, &response); err != nil {
		return PaymentLinkQueryResponse{}, err
	}
	if response.Status >= 300 {
		return PaymentLinkQueryResponse{}, fmt.Errorf("unexpected response status %d", response.Status)
	}
	if bodyDto.Status != "success" {
		return PaymentLinkQueryResponse{}, NotSuccessful
	}
	if len(bodyDto.Data) != 1 {
		return PaymentLinkQueryResponse{}, fmt.Errorf("unexpected number of response body data array entries %d", len(bodyDto.Data))
	}
	return bodyDto.Data[0], nil
}

// delete does not return a response body?
type deleteLowlevelResponseBody struct {
	Status string `json:"status"`
}

func (i *Impl) DeletePaymentLink(ctx context.Context, id uint) error {
	requestUrl := fmt.Sprintf("%s/v1.0/Invoice/%d/?instance=%s", i.baseUrl, id, url.QueryEscape(i.instanceName))
	requestBody := buildEmptyRequestBody()
	bodyDto := deleteLowlevelResponseBody{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	if err := i.client.Perform(ctx, http.MethodDelete, requestUrl, requestBody, &response); err != nil {
		return err
	}
	if response.Status >= 300 {
		return fmt.Errorf("unexpected response status %d", response.Status)
	}
	//if bodyDto.Status != "success" {
	//	return NotSuccessful
	//}
	return nil
}

func (i *Impl) QueryTransactions(ctx context.Context, timeGreaterThan time.Time, timeLessThan time.Time) ([]TransactionData, error) {
	//TODO implement me
	panic("implement me")
}
