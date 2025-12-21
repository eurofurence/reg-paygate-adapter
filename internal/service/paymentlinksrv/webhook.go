package paymentlinksrv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"
)

const isoDateFormat = "2006-01-02"

func (i *Impl) LogRawWebhook(ctx context.Context, payload string) error {
	aulogging.Logger.Ctx(ctx).Info().Print("webhook request: " + payload)

	db := database.GetRepository()
	return db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "raw",
		Message:     "webhook request",
		Details:     payload,
		RequestId:   ctxvalues.RequestId(ctx),
	})
}

func (i *Impl) HandleWebhook(ctx context.Context, webhook nexiapi.WebhookDto) error {
	event := webhook.Event
	aulogging.Logger.Ctx(ctx).Info().Printf("webhook id=%s event=%s", webhook.Id, event)

	if event == nexiapi.EventPaymentCreated {
		return i.eventPaymentCreated(ctx, webhook)
	} else if event == nexiapi.EventPaymentChargeCreatedV2 {
		return i.eventPaymentChargeCreatedV2(ctx, webhook)
	} else if event == nexiapi.EventPaymentCheckoutCompleted {
		return i.eventPaymentCheckoutCompleted(ctx, webhook)
	} else {
		return i.eventUnexpected(ctx, webhook)
	}

}

func (i *Impl) eventPaymentCreated(ctx context.Context, webhook nexiapi.WebhookDto) error {
	data, err := parseUnion[nexiapi.DataPaymentCreated](ctx, i, webhook)
	if err != nil {
		return err
	}

	// we only log payment creation, it is a response to our API call, and may come intermittently, so we do not
	// even check reference ids
	aulogging.Logger.Ctx(ctx).Info().Printf("webhook event=%s ref=%s amount=%d currency=%s id=%s",
		nexiapi.EventPaymentCreated,
		data.Order.Reference,
		data.Order.Amount.Amount,
		data.Order.Amount.Currency,
		data.PaymentId,
	)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: data.Order.Reference,
		ApiId:       data.PaymentId,
		Kind:        "success",
		Message:     fmt.Sprintf("webhook %s", nexiapi.EventPaymentCreated),
		Details:     fmt.Sprintf("amount=%d currency=%s", data.Order.Amount.Amount, data.Order.Amount.Currency),
		RequestId:   ctxvalues.RequestId(ctx),
	})

	return nil
}

func (i *Impl) eventPaymentChargeCreatedV2(ctx context.Context, webhook nexiapi.WebhookDto) error {
	data, err := parseUnion[nexiapi.DataPaymentChargeCreatedV2](ctx, i, webhook)
	if err != nil {
		return err
	}

	// for now, we only log charge creation, we use checkout completed events instead, they include
	// the reference id
	aulogging.Logger.Ctx(ctx).Info().Printf("webhook event=%s method=%s type=%s amount=%d currency=%s id=%s",
		nexiapi.EventPaymentChargeCreatedV2,
		data.PaymentMethod,
		data.PaymentType,
		data.Amount.Amount,
		data.Amount.Currency,
		data.PaymentId,
	)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       data.PaymentId,
		Kind:        "success",
		Message:     fmt.Sprintf("webhook %s", nexiapi.EventPaymentChargeCreatedV2),
		Details:     fmt.Sprintf("method=%s type=%s amount=%d currency=%s", data.PaymentMethod, data.PaymentType, data.Amount.Amount, data.Amount.Currency),
		RequestId:   ctxvalues.RequestId(ctx),
	})

	return nil
}

func (i *Impl) eventPaymentCheckoutCompleted(ctx context.Context, webhook nexiapi.WebhookDto) error {
	data, err := parseUnion[nexiapi.DataPaymentCheckoutCompleted](ctx, i, webhook)
	if err != nil {
		return err
	}

	// validate or create (pending!!) payment with given reference id, we only trust webhooks so much
	prefix := config.TransactionIDPrefix()
	if prefix != "" && !strings.HasPrefix(data.Order.Reference, prefix) {
		aulogging.Logger.Ctx(ctx).Warn().Printf("webhook with wrong ref id prefix, ref=%s", data.Order.Reference)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.Order.Reference,
			ApiId:       data.PaymentId,
			Kind:        "error",
			Message:     fmt.Sprintf("webhook %s ref-id-prefix wrong", nexiapi.EventPaymentCheckoutCompleted),
			Details:     fmt.Sprintf("expecting prefix %s", prefix),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", data.Order.Reference, "ref-id-prefix mismatch")
		// report success so they don't retry, it's not a big problem after all
		return nil
	}

	// fetch transaction data from payment service
	transaction, err := paymentservice.Get().GetTransactionByReferenceId(ctx, data.Order.Reference)
	if err != nil {
		if err == paymentservice.NotFoundError {
			// transaction not found in the payment service -> create one.
			// Note: this should never happen, but we try to recover because someone paid us money for something.
			aulogging.Logger.Ctx(ctx).Error().Printf("webhook ref not found in payment service. Creating new pending transaction and flagging for manual review. ref=%s", data.Order.Reference)

			return i.createTransaction(ctx, data)
		} else {
			aulogging.Logger.Ctx(ctx).Error().Printf("error fetching transaction from payment service. err=%s", err.Error())
			return err
		}
	}

	// matching transaction was found in the payment service database.
	// update the status if applicable.
	return i.updateTransaction(ctx, data, transaction)
}

func (i *Impl) eventUnexpected(ctx context.Context, webhook nexiapi.WebhookDto) error {
	aulogging.Logger.Ctx(ctx).Error().Printf("unexpected webhook event %s - ignoring", webhook.Event)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       "",
		Kind:        "error",
		Message:     fmt.Sprintf("webhook %s unknown event", webhook.Event),
		Details:     "",
		RequestId:   ctxvalues.RequestId(ctx),
	})
	_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("unknown event: %s", webhook.Event), "unexpected-event")

	// confirm with 200 so we do not keep receiving the event - we've done all we can
	return nil
}

func parseUnion[D any](ctx context.Context, i *Impl, webhook nexiapi.WebhookDto) (D, error) {
	var data D

	err := json.Unmarshal(webhook.Data, &data)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook called with invalid data payload: %s", err.Error())
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("webhookId: %s", webhook.Id), "api-error")
	}

	return data, err
}

func (i *Impl) createTransaction(ctx context.Context, data nexiapi.DataPaymentCheckoutCompleted) error {
	debitor_id, err := debitorIdFromReferenceID(data.Order.Reference)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Warn().Printf("webhook couldn't parse debitor_id from reference_id. reference_id=%s", data.Order.Reference)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.Order.Reference,
			ApiId:       data.PaymentId,
			Kind:        "error",
			Message:     fmt.Sprintf("webhook %s cannot determine debitor from reference id and payment not found", nexiapi.EventPaymentCheckoutCompleted),
			Details:     fmt.Sprintf("amount=%d currency=%s", data.Order.Amount.Amount, data.Order.Amount.Currency),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", data.Order.Reference), "parse-refid-err")
		// do not continue - we wouldn't know which attendee to associate the payment with. Needs manual investigation
		return err
	}

	effective := i.effectiveToday()
	comment := "CC paymentId " + i.transactionUuid(data) + " (auto created)"
	var vatRate float64
	if len(data.Order.OrderItems) > 0 {
		vatRate = float64(data.Order.OrderItems[0].TaxRate) / 100.0
	}

	transaction := paymentservice.Transaction{
		ID:        data.Order.Reference,
		DebitorID: debitor_id,
		Type:      paymentservice.Payment,
		Method:    paymentservice.Credit, // we don't know at this point
		Amount: paymentservice.Amount{
			GrossCent: int64(data.Order.Amount.Amount),
			Currency:  data.Order.Amount.Currency,
			VatRate:   vatRate,
		},
		Comment:       comment,
		Status:        paymentservice.Pending,
		EffectiveDate: effective,
		DueDate:       effective,
		// omitting Deletion
	}

	err = paymentservice.Get().AddTransaction(ctx, transaction)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook could not create transaction in payment service! (we don't know why we received this money, and we couldn't add the transaction to the database either!) reference_id=%s", data.Order.Reference)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.Order.Reference,
			ApiId:       data.PaymentId,
			Kind:        "error",
			Message:     fmt.Sprintf("webhook %s failed to create transaction in payment service", nexiapi.EventPaymentCheckoutCompleted),
			Details:     fmt.Sprintf("amount=%d currency=%s error=%s", data.Order.Amount.Amount, data.Order.Amount.Currency, err.Error()),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", data.Order.Reference), "create-missing-err")
		return err
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("webhook event=%s amount=%d currency=%s id=%s",
		nexiapi.EventPaymentCheckoutCompleted,
		data.Order.Amount.Amount,
		data.Order.Amount.Currency,
		data.PaymentId,
	)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: data.Order.Reference,
		ApiId:       data.PaymentId,
		Kind:        "warning",
		Message:     fmt.Sprintf("webhook %s created PENDING payment - did not exist - needs review", nexiapi.EventPaymentCheckoutCompleted),
		Details:     fmt.Sprintf("amount=%d currency=%s", data.Order.Amount.Amount, data.Order.Amount.Currency),
		RequestId:   ctxvalues.RequestId(ctx),
	})
	_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", data.Order.Reference), "create-missing-pending-success (needs review)")
	return nil
}

func (i *Impl) updateTransaction(ctx context.Context, data nexiapi.DataPaymentCheckoutCompleted, transaction paymentservice.Transaction) error {
	if transaction.Status == paymentservice.Valid {
		aulogging.Logger.Ctx(ctx).Warn().Printf("aborting transaction update - already in status valid! reference_id=%s", data.Order.Reference)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.Order.Reference,
			ApiId:       data.PaymentId,
			Kind:        "warning",
			Message:     fmt.Sprintf("webhook %s payment already in status valid", nexiapi.EventPaymentCheckoutCompleted),
			Details:     fmt.Sprintf("amount=%d currency=%s", data.Order.Amount.Amount, data.Order.Amount.Currency),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", data.Order.Reference), "abort-update-for-valid")
		return nil // not an error
	}

	effective := i.effectiveToday()
	comment := "CC paymentId " + i.transactionUuid(data)

	if transaction.Amount.GrossCent != int64(data.Order.Amount.Amount) || transaction.Amount.Currency != data.Order.Amount.Currency {
		aulogging.Logger.Ctx(ctx).Warn().Printf("transaction update changes amount or currency! reference_id=%s", data.Order.Reference)
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", data.Order.Reference), "amount-difference-please-check")
		// continue!

		transaction.Amount.GrossCent = int64(data.Order.Amount.Amount)
		transaction.Amount.Currency = data.Order.Amount.Currency
	}

	transaction.Status = paymentservice.Valid
	transaction.EffectiveDate = effective
	transaction.Comment = comment

	err := paymentservice.Get().UpdateTransaction(ctx, transaction)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook unable to update upstream transaction. reference_id=%s", data.Order.Reference)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.Order.Reference,
			ApiId:       data.PaymentId,
			Kind:        "error",
			Message:     fmt.Sprintf("webhook %s failed to update payment", nexiapi.EventPaymentCheckoutCompleted),
			Details:     fmt.Sprintf("amount=%d currency=%s error=%s", data.Order.Amount.Amount, data.Order.Amount.Currency, err.Error()),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("refId: %s", data.Order.Reference), "update-tx-err")
		return err
	}

	return nil
}

func (i *Impl) effectiveToday() string {
	return time.Now().Format(isoDateFormat)
}

func (i *Impl) transactionUuid(data nexiapi.DataPaymentCheckoutCompleted) string {
	return data.PaymentId
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
