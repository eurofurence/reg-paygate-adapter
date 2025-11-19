package nexi

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
)

type Mock interface {
	NexiDownstream

	Reset()
	Recording() []string
	SimulateError(err error)
	InjectTransaction(tx TransactionData)
	ManipulateStatus(paylinkId string, status string)
}

type mockImpl struct {
	recording     []string
	simulateError error
	simulatorData map[string]NexiPaymentQueryResponse
	idSequence    uint32
	simulatorTx   []TransactionData
}

func newMock() Mock {
	simData := make(map[string]NexiPaymentQueryResponse)
	// used by some testcases
	simData["42"] = NexiPaymentQueryResponse{
		ID:          "42",
		Status:      "confirmed",
		ReferenceID: "221216-122218-000001",
		Link:        constructSimulatedPaylink("42"),
		Amount:      390,
		Currency:    "EUR",
		CreatedAt:   1673136000, // 2023-01-08
		VatRate:     19.0,
		Order: NexiOrderDetails{
			Reference: "221216-122218-000001",
			Amount:    390,
			Currency:  "EUR",
		},
		Summary: NexiSummary{
			ChargedAmount: 390,
		},
		Consumer: NexiConsumerFull{},
		Payments: []NexiPaymentDetails{},
		Refunds:  []NexiRefund{},
		Charges:  []NexiCharge{},
	}
	simData["4242"] = NexiPaymentQueryResponse{
		ID:          "4242",
		Status:      "confirmed",
		ReferenceID: "230001-122218-000001",
		Link:        constructSimulatedPaylink("4242"),
		Amount:      390,
		Currency:    "EUR",
		CreatedAt:   1418392958,
		VatRate:     19.0,
		Order: NexiOrderDetails{
			Reference: "230001-122218-000001",
			Amount:    390,
			Currency:  "EUR",
		},
		Summary: NexiSummary{
			ChargedAmount: 390,
		},
		Consumer: NexiConsumerFull{},
		Payments: []NexiPaymentDetails{},
		Refunds:  []NexiRefund{},
		Charges:  []NexiCharge{},
	}
	return &mockImpl{
		recording:     make([]string, 0),
		simulatorData: simData,
		simulatorTx:   make([]TransactionData, 0),
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

func (m *mockImpl) CreatePaymentLink(ctx context.Context, request NexiCreatePaymentRequest) (NexiPaymentLinkCreated, error) {
	if m.simulateError != nil {
		return NexiPaymentLinkCreated{}, m.simulateError
	}
	m.recording = append(m.recording, "CreatePaymentLink")

	newIdNum := atomic.AddUint32(&m.idSequence, 1)
	newId := fmt.Sprintf("mock-%d", newIdNum)
	response := NexiPaymentLinkCreated{
		ID:          newId,
		ReferenceID: request.Order.Reference,
		Link:        constructSimulatedPaylink(newId),
	}
	data := NexiPaymentQueryResponse{
		ID:          newId,
		Status:      "confirmed",
		ReferenceID: request.Order.Reference,
		Link:        response.Link,
		Amount:      request.Order.Amount,
		Currency:    request.Order.Currency,
		CreatedAt:   1418392958,
		VatRate:     19.0,
		Order: NexiOrderDetails{
			Reference: request.Order.Reference,
			Amount:    request.Order.Amount,
			Currency:  request.Order.Currency,
			Items:     request.Order.Items,
		},
		Summary: NexiSummary{
			ChargedAmount: request.Order.Amount,
		},
		Consumer: NexiConsumerFull{},
		Payments: []NexiPaymentDetails{},
		Refunds:  []NexiRefund{},
		Charges:  []NexiCharge{},
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("mock creating payment link id=%s amount=%d curr=%s", newId, request.Order.Amount, request.Order.Currency)

	m.simulatorData[newId] = data
	return response, nil
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
	tx.ID = newId
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
