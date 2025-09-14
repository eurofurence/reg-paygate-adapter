package nexi

import (
	"context"
	"fmt"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"sync/atomic"
	"time"
)

type Mock interface {
	NexiDownstream

	Reset()
	Recording() []string
	SimulateError(err error)
	InjectTransaction(tx TransactionData)
	ManipulateStatus(paylinkId uint, status string)
}

type mockImpl struct {
	recording     []string
	simulateError error
	simulatorData map[uint]PaymentLinkQueryResponse
	idSequence    uint32
	simulatorTx   []TransactionData
}

func newMock() Mock {
	simData := make(map[uint]PaymentLinkQueryResponse)
	// used by some testcases
	simData[42] = PaymentLinkQueryResponse{
		ID:          42,
		Status:      "confirmed",
		ReferenceID: "221216-122218-000001",
		Link:        constructSimulatedPaylink("42"),
		Name:        "Online-Shop payment #001",
		Purpose:     map[string]string{"1": "some payment purpose"},
		Amount:      390,
		Currency:    "EUR",
		CreatedAt:   1418392958,
		Invoices: []PaymentLinkInvoice{
			{
				Transactions: []TransactionData{
					{
						Time: "2023-01-08 12:22:58",
						UUID: "d3adb33f",
					},
				},
			},
		},
	}
	simData[4242] = PaymentLinkQueryResponse{
		ID:          4242,
		Status:      "confirmed",
		ReferenceID: "230001-122218-000001",
		Link:        constructSimulatedPaylink("4242"),
		Name:        "Online-Shop payment #002",
		Purpose:     map[string]string{"1": "some payment purpose"},
		Amount:      390,
		Currency:    "EUR",
		CreatedAt:   1418392958,
		Invoices: []PaymentLinkInvoice{
			{
				Transactions: []TransactionData{
					{
						Time: "2023-01-08 12:22:58",
						UUID: "d3adb33f",
					},
				},
			},
		},
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

func (m *mockImpl) CreatePaymentLink(ctx context.Context, request PaymentLinkCreateRequest) (PaymentLinkCreated, error) {
	if m.simulateError != nil {
		return PaymentLinkCreated{}, m.simulateError
	}
	m.recording = append(m.recording, fmt.Sprintf("CreatePaymentLink %v", request))

	newId := uint(atomic.AddUint32(&m.idSequence, 1))
	response := PaymentLinkCreated{
		ID:          newId,
		ReferenceID: request.ReferenceId,
		Link:        constructSimulatedPaylink(fmt.Sprintf("%d", newId)),
	}
	data := PaymentLinkQueryResponse{
		ID:          newId,
		Status:      "confirmed",
		ReferenceID: request.ReferenceId,
		Link:        response.Link,
		Name:        "Online-Shop payment #001",
		Purpose:     map[string]string{"1": "some payment purpose"},
		Amount:      request.Amount,
		Currency:    request.Currency,
		CreatedAt:   1418392958,
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("mock creating payment link id=%d amount=%d curr=%s email=%s", newId, request.Amount, request.Currency, request.Email)

	m.simulatorData[newId] = data
	return response, nil
}

func (m *mockImpl) QueryPaymentLink(ctx context.Context, id uint) (PaymentLinkQueryResponse, error) {
	if m.simulateError != nil {
		return PaymentLinkQueryResponse{}, m.simulateError
	}
	m.recording = append(m.recording, fmt.Sprintf("QueryPaymentLink %d", id))

	copiedData, ok := m.simulatorData[id]
	if !ok {
		return PaymentLinkQueryResponse{}, NoSuchID404Error
	}
	return copiedData, nil
}

func (m *mockImpl) DeletePaymentLink(ctx context.Context, id uint) error {
	if m.simulateError != nil {
		return m.simulateError
	}
	m.recording = append(m.recording, fmt.Sprintf("DeletePaymentLink %d", id))

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

	// add transaction to paylink
	for id, paylink := range m.simulatorData {
		if paylink.ReferenceID == tx.ReferenceID {
			paylink.Invoices = make([]PaymentLinkInvoice, 1)
			paylink.Invoices[0] = PaymentLinkInvoice{
				ReferenceID:      tx.ReferenceID,
				PaymentRequestId: id,
				Currency:         paylink.Currency,
				Amount:           paylink.Amount,
				Transactions:     []TransactionData{tx},
			}
			m.simulatorData[id] = paylink
		}
	}
}

func (m *mockImpl) ManipulateStatus(paylinkId uint, status string) {
	copiedData, ok := m.simulatorData[paylinkId]
	if !ok {
		return
	}
	copiedData.Status = status
	m.simulatorData[paylinkId] = copiedData
}
