package paymentlinksrv

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
)

const isoDateFormat = "2006-01-02"

func (i *Impl) HandleWebhook(ctx context.Context, webhook nexiapi.WebhookEventDto) error {
	aulogging.Logger.Ctx(ctx).Info().Printf("webhook id=%d invoice.paymentRequestId=%d invoice.referenceId=%s", webhook.Transaction.Id, webhook.Transaction.Invoice.PaymentRequestId, webhook.Transaction.Invoice.ReferenceId)

	paylinkId, err := idValidate(webhook.Transaction.Invoice.PaymentRequestId)
	if err != nil {
		if webhook.Transaction.Invoice.Number == "123456" && webhook.Transaction.Invoice.PaymentRequestId == 0 {
			// could be a test webhook invocation
			aulogging.Logger.Ctx(ctx).Warn().Printf("webhook called with invalid paylink ID 0 invoice number 123456 (probably someone clicked test button in UI)")
			_ = i.SendErrorNotifyMail(ctx, "webhook", "paylinkId 0 test button", "api-error")

			return nil
		}

		aulogging.Logger.Ctx(ctx).Error().Printf("webhook called with invalid paylink ID. id=%d", webhook.Transaction.Invoice.PaymentRequestId)
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("paylinkId: %d", webhook.Transaction.Invoice.PaymentRequestId), "api-error")

		return err
	}

	paylink, err := nexi.Get().QueryPaymentLink(ctx, fmt.Sprintf("%d", paylinkId)) // convert to string
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("can't query payment link from nexi. err=%s", err.Error())
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: webhook.Transaction.Invoice.ReferenceId,
			ApiId:       fmt.Sprintf("%d", paylinkId),
			Kind:        "error",
			Message:     "webhook query-pay-link failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("paylinkId: %d", paylinkId), "api-error")
		return err
	}

	if paylink.ReferenceID != webhook.Transaction.Invoice.ReferenceId {
		// webhook data claimed it was about ref_id A, but the paylink is for ref_id B
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook reference_id mismatch, ref_id in webhook=%s, ref_id in paylink data=%s", webhook.Transaction.Invoice.ReferenceId, paylink.ReferenceID)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: webhook.Transaction.Invoice.ReferenceId,
			ApiId:       fmt.Sprintf("%d", paylinkId),
			Kind:        "error",
			Message:     "webhook ref-id-mismatch",
			Details:     fmt.Sprintf("response ref-id=%s vs webhook ref-id=%s", paylink.ReferenceID, webhook.Transaction.Invoice.ReferenceId),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("paylinkId: %d", paylinkId), "ref-id-mismatch")
		return WebhookRefIdMismatchErr
	}

	prefix := config.TransactionIDPrefix()
	if prefix != "" && !strings.HasPrefix(paylink.ReferenceID, prefix) {
		aulogging.Logger.Ctx(ctx).Warn().Printf("webhook with wrong ref id prefix, ref_id=%s", paylink.ReferenceID)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: webhook.Transaction.Invoice.ReferenceId,
			ApiId:       fmt.Sprintf("%d", paylinkId),
			Kind:        "error",
			Message:     "webhook ref-id-prefix",
			Details:     fmt.Sprintf("ref-id=%s", paylink.ReferenceID),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", paylink.ReferenceID, "ref-id-prefix")
		// report success so they don't retry, it's not a big problem after all
		return nil
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("webhook call for paylink id=%d ref=%s status=%s amount=%d", paylink.ID, paylink.ReferenceID, paylink.Status, paylink.Amount)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: paylink.ReferenceID,
		ApiId:       fmt.Sprintf("%d", paylinkId),
		Kind:        "success",
		Message:     "webhook query-pay-link",
		Details:     fmt.Sprintf("status=%s amount=%d", paylink.Status, paylink.Amount),
		RequestId:   ctxvalues.RequestId(ctx),
	})

	if paylink.Status == "cancelled" || paylink.Status == "declined" {
		aulogging.Logger.Ctx(ctx).Info().Printf("irrelevant status, ignoring as successful")
		return nil
	}

	if paylink.Status != "confirmed" {
		_ = i.SendErrorNotifyMail(ctx, "webhook", paylink.ReferenceID, paylink.Status)
		// send 200 so nexi doesn't keep trying the webhook - we've done all we can
		return nil
	}

	// fetch transaction data from payment service
	transaction, err := paymentservice.Get().GetTransactionByReferenceId(ctx, paylink.ReferenceID)
	if err != nil {
		if err == paymentservice.NotFoundError {
			// transaction not found in the payment service -> create one.
			// Note: this should never happen, but we try to recover because someone paid us money for somthing.
			aulogging.Logger.Ctx(ctx).Error().Printf("webhook reference_id not found in payment service. Creating new transaction. reference_id=%s", paylink.ReferenceID)

			return i.createTransaction(ctx, paylink)
		} else {
			aulogging.Logger.Ctx(ctx).Error().Printf("error fetching transaction from payment service. err=%s", err.Error())
			return err
		}
	}

	// matching transaction was found in the payment service database.
	// update the values with data from Nexi.
	return i.updateTransaction(ctx, paylink, transaction)
}

func (i *Impl) createTransaction(ctx context.Context, paylink nexi.NexiPaymentQueryResponse) error {
	debitor_id, err := debitorIdFromReferenceID(paylink.ReferenceID)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Warn().Printf("webhook couldn't parse debitor_id from reference_id. reference_id=%s", paylink.ReferenceID)
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", paylink.ReferenceID), "parse-refid-err")
		// we log a warning, but we continue anyway
	}

	effective := i.effectiveISODateOrToday(paylink)
	comment := "CC orderId " + i.transactionUuid(paylink) + " (auto created)"

	transaction := paymentservice.Transaction{
		ID:        paylink.ReferenceID,
		DebitorID: debitor_id,
		Type:      paymentservice.Payment,
		Method:    paymentservice.Credit, // we use paylink for credit cards only, atm.
		Amount: paymentservice.Amount{
			GrossCent: int64(paylink.Amount),
			Currency:  paylink.Currency,
			VatRate:   paylink.VatRate,
		},
		Comment:       comment,
		Status:        paymentservice.Pending,
		EffectiveDate: effective,
		DueDate:       effective,
		// omitting Deletion
	}

	err = paymentservice.Get().AddTransaction(ctx, transaction)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook could not create transaction in payment service! (we don't know why we received this money, and we couldn't add the transaction to the database either!) reference_id=%s", paylink.ReferenceID)
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", paylink.ReferenceID), "create-missing-err")
	}
	return err
}

func (i *Impl) updateTransaction(ctx context.Context, paylink nexi.NexiPaymentQueryResponse, transaction paymentservice.Transaction) error {
	if transaction.Status == paymentservice.Valid {
		aulogging.Logger.Ctx(ctx).Warn().Printf("aborting transaction update - already in status valid! reference_id=%s", paylink.ReferenceID)
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", paylink.ReferenceID), "abort-update-for-valid")
		return nil // not an error
	}

	effective := i.effectiveISODateOrToday(paylink)
	comment := "CC orderId " + i.transactionUuid(paylink)

	transaction.Amount.GrossCent = int64(paylink.Amount)
	transaction.Amount.Currency = paylink.Currency
	transaction.Status = paymentservice.Valid
	transaction.EffectiveDate = effective
	transaction.Comment = comment

	err := paymentservice.Get().UpdateTransaction(ctx, transaction)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook unable to update upstream transaction. reference_id=%s", paylink.ReferenceID)
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", paylink.ReferenceID), "update-tx-err")
		return err
	}

	return nil
}

func (i *Impl) effectiveISODateOrToday(paylink nexi.NexiPaymentQueryResponse) string {
	today := time.Now().Format(isoDateFormat)
	effective := today

	// In the new structure, use CreatedAt or Charges[0].Created
	if paylink.CreatedAt > 0 {
		// Convert timestamp to date
		t := time.Unix(paylink.CreatedAt, 0)
		effective = t.Format(isoDateFormat)
	} else if len(paylink.Charges) > 0 {
		// Parse the Created time from the first charge
		if t, err := time.Parse(time.RFC3339, paylink.Charges[0].Created); err == nil {
			effective = t.Format(isoDateFormat)
		}
	}

	return effective
}

func (i *Impl) transactionUuid(paylink nexi.NexiPaymentQueryResponse) string {
	return paylink.ID
}

func debitorIdFromReferenceID(ref_id string) (uint, error) {
	// reference_id is generated internally in the payment service.
	// See  reg-payment-service/internal/interaction/transaction.go:generateTransactionID()

	// The format is:  "%s-%06d-%s-%s"
	// Fields:
	//   - prefix (hopefully without hyphens)
	//   - debitor_id
	//   - timestamp in format "0102-150405" (hyphen!)
	//   - random digits

	tokens := strings.Split(ref_id, "-")
	if len(tokens) != 5 {
		return 0, errors.New("error parsing reference_id")
	}

	debitor_id, err := strconv.ParseUint(tokens[1], 10, 32)
	if err != nil {
		return 0, errors.New("error parsing debitor_id as uint: " + err.Error())
	}

	return uint(debitor_id), nil
}

func idValidate(value int64) (uint, error) {
	if value < 1 {
		return 0, WebhookValidationErr
	}
	return uint(value), nil
}
