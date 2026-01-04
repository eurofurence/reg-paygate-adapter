package acceptance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database/inmemorydb"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/mailservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"

	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/media"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/require"
)

// placing these here because they are package global

type tstWebResponse struct {
	status      int
	body        string
	contentType string
	location    string
}

func tstWebResponseFromResponse(response *http.Response) tstWebResponse {
	status := response.StatusCode
	ct := ""
	if val, ok := response.Header[headers.ContentType]; ok {
		ct = val[0]
	}
	loc := ""
	if val, ok := response.Header[headers.Location]; ok {
		loc = val[0]
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = response.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	return tstWebResponse{
		status:      status,
		body:        string(body),
		contentType: ct,
		location:    loc,
	}
}

func tstPerformGet(relativeUrlWithLeadingSlash string, apiToken string) tstWebResponse {
	request, err := http.NewRequest(http.MethodGet, ts.URL+relativeUrlWithLeadingSlash, nil)
	if err != nil {
		log.Fatal(err)
	}
	if apiToken != "" {
		request.Header.Set(media.HeaderXApiKey, apiToken)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return tstWebResponseFromResponse(response)
}

func tstPerformPost(relativeUrlWithLeadingSlash string, requestBody string, apiToken string) tstWebResponse {
	request, err := http.NewRequest(http.MethodPost, ts.URL+relativeUrlWithLeadingSlash, strings.NewReader(requestBody))
	if err != nil {
		log.Fatal(err)
	}
	if apiToken != "" {
		request.Header.Set(media.HeaderXApiKey, apiToken)
	}
	request.Header.Set(headers.ContentType, media.ContentTypeApplicationJson)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return tstWebResponseFromResponse(response)
}

func tstPerformDelete(relativeUrlWithLeadingSlash string, apiToken string) tstWebResponse {
	request, err := http.NewRequest(http.MethodDelete, ts.URL+relativeUrlWithLeadingSlash, nil)
	if err != nil {
		log.Fatal(err)
	}
	if apiToken != "" {
		request.Header.Set(media.HeaderXApiKey, apiToken)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	return tstWebResponseFromResponse(response)
}

func tstRenderJson(v interface{}) string {
	representationBytes, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	return string(representationBytes)
}

// tip: dto := &XyzDto{}
func tstParseJson(body string, dto interface{}) {
	err := json.Unmarshal([]byte(body), dto)
	if err != nil {
		log.Fatal(err)
	}
}

func tstRequireErrorResponse(t *testing.T, response tstWebResponse, expectedStatus int, expectedMessage string, expectedDetails interface{}) {
	require.Equal(t, expectedStatus, response.status, "unexpected http response status")
	errorDto := nexiapi.ErrorDto{}
	tstParseJson(response.body, &errorDto)
	require.Equal(t, expectedMessage, errorDto.Message, "unexpected error code")
	expectedDetailsStr, ok := expectedDetails.(string)
	if ok && expectedDetailsStr != "" {
		require.EqualValues(t, url.Values{"details": []string{expectedDetailsStr}}, errorDto.Details, "unexpected error details")
	}
	expectedDetailsUrlValues, ok := expectedDetails.(url.Values)
	if ok {
		require.EqualValues(t, expectedDetailsUrlValues, errorDto.Details, "unexpected error details")
	}
}

func tstRequirePaymentLinkResponse(t *testing.T, response tstWebResponse, expectedStatus int, expectedBody nexiapi.PaymentLinkDto) {
	require.Equal(t, expectedStatus, response.status, "unexpected http response status")
	actualBody := nexiapi.PaymentLinkDto{}
	tstParseJson(response.body, &actualBody)
	require.EqualValues(t, expectedBody, actualBody)
}

func tstRequireNexiRecording(t *testing.T, expectedEntries ...string) {
	actual := nexiMock.Recording()
	require.Equal(t, len(expectedEntries), len(actual))
	for i := range expectedEntries {
		require.Equal(t, expectedEntries[i], actual[i])
	}
}

func tstRequireMailServiceRecording(t *testing.T, expectedEntries []mailservice.MailSendDto) {
	actual := mailMock.Recording()
	require.Equal(t, len(expectedEntries), len(actual))
	for i := range expectedEntries {
		require.Equal(t, expectedEntries[i], actual[i])
	}
}

func tstRequirePaymentServiceRecording(t *testing.T, expectedEntries []paymentservice.Transaction) {
	actual := paymentMock.Recording()
	require.Equal(t, len(expectedEntries), len(actual))
	for i := range expectedEntries {
		require.Equal(t, expectedEntries[i], actual[i])
	}
}

// --- data ---

func tstBuildValidPaymentLinkRequest() nexiapi.PaymentLinkRequestDto {
	return nexiapi.PaymentLinkRequestDto{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		DebitorId:   1,
		AmountDue:   390,
		Currency:    "EUR",
		VatRate:     19.0,
	}
}

func tstBuildValidPaymentLink() nexiapi.PaymentLinkDto {
	return nexiapi.PaymentLinkDto{
		Title:       "some page title",
		Description: "some page description",
		ReferenceId: "EF1995-000001-221216-122218-4132",
		Purpose:     "some payment purpose",
		AmountDue:   390,
		AmountPaid:  0,
		Currency:    "EUR",
		VatRate:     19.0,
		Link:        "http://localhost:1111/some/paylink/EF1995-000001-221216-122218-4132",
	}
}

func tstBuildValidPaymentLinkGetResponse() nexiapi.PaymentLinkDto {
	return nexiapi.PaymentLinkDto{
		ReferenceId: "EF1995-000001-221216-122218-4132",
		Purpose:     "some payment purpose",
		AmountDue:   390,
		AmountPaid:  0,
		Currency:    "EUR",
		Link:        "http://localhost:1111/some/paylink/42",
	}
}

func tstBuildValidWebhookRequest(t *testing.T, event string) string {
	re := regexp.MustCompile(`(\r\s*|\n\s*)`)
	// for each event, hardcode one realistic webhook body
	if event == nexiapi.EventPaymentCreated {
		return re.ReplaceAllString(`
  {
    "id": "01234567890abcdef0123456789abcde",
    "merchantId": 123456789,
    "timestamp": "2025-12-15T13:24:15.9680+00:00",
    "event": "payment.created",
    "data": {
      "order": {
        "amount": {
          "amount": 18500,
          "currency": "EUR",
          "other": "some extra stuff to test tolerant reader pattern"
        },
        "reference": "EF1995-000001-221216-122218-4132"
      },
      "paymentId": "ef00000000000000000000000000cafe"
    }
  }
`, "")
	} else if event == nexiapi.EventPaymentChargeCreatedV2 {
		return re.ReplaceAllString(`
  {
    "id": "01234567890abcdef0123456789abcde",
    "timestamp": "2025-12-15T13:24:23.0175+00:00",
    "merchantNumber": 123456789,
    "event": "payment.charge.created.v2",
    "data": {
      "paymentMethod": "Visa",
      "paymentType": "CARD",
      "amount": {
        "amount": 18500,
        "currency": "EUR"
      },
      "surchargeAmount": 0,
      "paymentId": "ef00000000000000000000000000cafe"
    }
  }
`, "")
	} else if event == nexiapi.EventPaymentCheckoutCompleted {
		return re.ReplaceAllString(`
  {
    "id": "01234567890abcdef0123456789abcde",
    "merchantId": 123456789,
    "timestamp": "2025-12-15T13:24:23.0175+00:00",
    "event": "payment.checkout.completed",
    "data": {
      "order": {
        "amount": {
          "amount": 18500,
          "currency": "EUR"
        },
        "reference": "EF1995-000001-221216-122218-4132"
      },
      "paymentId": "ef00000000000000000000000000cafe"
    }
  }
`, "")
	} else {
		return re.ReplaceAllString(fmt.Sprintf(`
  {
    "id": "01234567890abcdef0123456789abcde",
    "merchantId": 123456789,
    "timestamp": "2025-12-15T13:24:23.0175+00:00",
    "event": "%s",
    "data": {
      "paymentId": "ef00000000000000000000000000cafe"
    }
  }
`, event), "")
	}
}

func tstExpectedMailNotification(operation string, status string) mailservice.MailSendDto {
	return mailservice.MailSendDto{
		CommonID: "payment-nexi-adapter-error",
		Lang:     "en-US",
		To: []string{
			"errors@example.com",
		},
		Variables: map[string]string{
			"status":      status,
			"operation":   operation,
			"referenceId": "EF1995-000001-221216-122218-4132",
		},
		Async: true,
	}
}

func tstClearDatabase() {
	db := database.GetRepository().(*inmemorydb.InMemoryRepository)
	db.Clear()
}

func tstRequireProtocolEntries(t *testing.T, expectedProtocol ...entity.ProtocolEntry) {
	db := database.GetRepository().(*inmemorydb.InMemoryRepository)
	actualProtocol := db.ProtocolEntries()
	require.Equal(t, len(expectedProtocol), len(actualProtocol))
	for i, expected := range expectedProtocol {
		actual := *(actualProtocol[i])
		require.Equal(t, expected.ReferenceId, actual.ReferenceId)
		require.Equal(t, expected.ApiId, actual.ApiId)
		require.Equal(t, expected.Kind, actual.Kind)
		require.Equal(t, expected.Message, actual.Message)
		require.Equal(t, expected.Details, actual.Details)
	}
}
