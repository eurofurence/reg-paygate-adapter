package nexi

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
)

type Mock interface {
	NexiDownstream

	Reset()
	Recording() []string
	SimulateError(err error)
	InjectTransaction(tx TransactionData)
	ManipulateStatus(paylinkId string, status string)

	GetCachedWebhook(referenceId string) (nexiapi.WebhookDto, error)
}

type mockImpl struct {
	recording     []string
	simulateError error
	simulatorData map[string]NexiPaymentQueryResponse
	webhookCache  map[string]nexiapi.WebhookDto
	idSequence    uint32
	simulatorTx   []TransactionData
}

func newMock() Mock {
	// not actually queried, but currently used by some test cases
	simData := make(map[string]NexiPaymentQueryResponse)
	// used by some testcases
	//simData["42"] = NexiPaymentQueryResponse{
	//	ID:          "42",
	//	Status:      "confirmed",
	//	ReferenceID: "EF1995-000001-221216-122218-4132",
	//	Link:        constructSimulatedPaylink("42"),
	//	Amount:      390,
	//	Currency:    "EUR",
	//	CreatedAt:   1673136000, // 2023-01-08
	//	VatRate:     19.0,
	//	Order: NexiOrderDetails{
	//		Reference: "EF1995-000001-221216-122218-4132",
	//		Amount:    390,
	//		Currency:  "EUR",
	//	},
	//	Summary: NexiSummary{
	//		ChargedAmount: 390,
	//	},
	//	Consumer: NexiConsumerFull{},
	//	Payments: []NexiPaymentDetails{},
	//	Refunds:  []NexiRefund{},
	//	Charges:  []NexiCharge{},
	//}
	//simData["4242"] = NexiPaymentQueryResponse{
	//	ID:          "4242",
	//	Status:      "confirmed",
	//	ReferenceID: "EF1995-000001-230001-122218-5555",
	//	Link:        constructSimulatedPaylink("5555"),
	//	Amount:      390,
	//	Currency:    "EUR",
	//	CreatedAt:   1418392958,
	//	VatRate:     19.0,
	//	Order: NexiOrderDetails{
	//		Reference: "EF1995-000001-230001-122218-5555",
	//		Amount:    390,
	//		Currency:  "EUR",
	//	},
	//	Summary: NexiSummary{
	//		ChargedAmount: 390,
	//	},
	//	Consumer: NexiConsumerFull{},
	//	Payments: []NexiPaymentDetails{},
	//	Refunds:  []NexiRefund{},
	//	Charges:  []NexiCharge{},
	//}

	webhookCache := make(map[string]nexiapi.WebhookDto)
	return &mockImpl{
		recording:     make([]string, 0),
		simulatorData: simData,
		simulatorTx:   make([]TransactionData, 0),
		webhookCache:  webhookCache,
		idSequence:    100,
	}
}

func constructSimulatedPaylink(referenceId string) string {
	baseUrl := config.ServicePublicURL()
	if baseUrl == "" {
		return "http://localhost:1111/some/paylink/" + referenceId
	} else {
		return baseUrl + "/simulator/" + referenceId
	}
}

func (m *mockImpl) CreatePaymentLink(ctx context.Context, request NexiCreateCheckoutSessionRequest) (NexiCreateCheckoutSessionResponse, error) {
	if m.simulateError != nil {
		return NexiCreateCheckoutSessionResponse{}, m.simulateError
	}
	m.recording = append(m.recording, "CreatePaymentLink")

	newIdNum := atomic.AddUint32(&m.idSequence, 1)
	newId := fmt.Sprintf("mock-%d", newIdNum)
	response := NexiCreateCheckoutSessionResponse{
		Links: NexiCheckoutSessionResponseLinks{Redirect: &RedirectLink{
			Href: constructSimulatedPaylink(request.TransId),
			Type: "", // TODO
		}},
	}

	webhook := nexiapi.WebhookDto{
		PayId:               "42",
		TransId:             request.TransId,
		Status:              "OK",
		ResponseCode:        "00000000",
		ResponseDescription: "success",
		Amount: nexiapi.WebhookAmount{
			Value:    request.Amount.Value,
			Currency: request.Amount.Currency,
		},
		PaymentMethods: nexiapi.WebhookPaymentMethod{
			Type: "CARD",
		},
		CreationDate: "2025-09-01T14:00:00Z",
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("mock creating payment link id=%s amount=%d curr=%s", newId, request.Amount.Value, request.Amount.Currency)
	m.webhookCache[request.TransId] = webhook
	return response, nil
}

func (m *mockImpl) GetCachedWebhook(referenceId string) (nexiapi.WebhookDto, error) {
	webhook, ok := m.webhookCache[referenceId]
	if !ok {
		return nexiapi.WebhookDto{}, errors.New("webhook not found")
	}
	return webhook, nil
}

func (m *mockImpl) QueryPaymentLink(ctx context.Context, id string) (NexiPaymentQueryResponse, error) {
	if m.simulateError != nil {
		return NexiPaymentQueryResponse{}, m.simulateError
	}
	m.recording = append(m.recording, fmt.Sprintf("QueryPaymentLink %s", id))

	copiedData, ok := m.simulatorData[id]
	if !ok {
		return NexiPaymentQueryResponse{}, NoSuchID404Error
	}
	return copiedData, nil
}

func (m *mockImpl) DeletePaymentLink(ctx context.Context, id string, amount int64) error {
	if m.simulateError != nil {
		return m.simulateError
	}
	m.recording = append(m.recording, fmt.Sprintf("DeletePaymentLink %s", id))

	_, ok := m.simulatorData[id]
	if !ok {
		return NoSuchID404Error
	}
	delete(m.simulatorData, id)
	return nil
}

func (m *mockImpl) QueryTransactions(ctx context.Context, timeGreaterThan time.Time, timeLessThan time.Time) ([]TransactionData, error) {
	if m.simulateError != nil {
		return []TransactionData{}, m.simulateError
	}
	m.recording = append(m.recording, fmt.Sprintf("QueryTransactions %v <= t <= %v", timeGreaterThan, timeLessThan))

	copiedTransactions := make([]TransactionData, len(m.simulatorTx))
	for k, v := range m.simulatorTx {
		// time matching not implemented because it interferes with our tests
		copiedTransactions[k] = v
	}
	return copiedTransactions, nil
}

func (m *mockImpl) Reset() {
	m.recording = make([]string, 0)
	m.simulateError = nil
}

func (m *mockImpl) Recording() []string {
	return m.recording
}

func (m *mockImpl) SimulateError(err error) {
	m.simulateError = err
}

func (m *mockImpl) InjectTransaction(tx TransactionData) {
	newId := int64(atomic.AddUint32(&m.idSequence, 1))
	tx.Id = newId
	m.simulatorTx = append(m.simulatorTx, tx)

	// TODO: adapt to new NexiPaymentQueryResponse structure if needed
	// For now, just add to the transaction list
}

func (m *mockImpl) ManipulateStatus(paylinkId string, status string) {
	copiedData, ok := m.simulatorData[paylinkId]
	if !ok {
		return
	}
	copiedData.Status = status
	m.simulatorData[paylinkId] = copiedData
}
