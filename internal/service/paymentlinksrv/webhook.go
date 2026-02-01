package paymentlinksrv

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
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
	aulogging.Logger.Ctx(ctx).Info().Printf("webhook id=%s tx=%s status=%s responsecode=%s", webhook.PayId, webhook.TransId, webhook.Status, webhook.ResponseCode)

	if webhook.Status == "OK" || webhook.Status == "AUTHORIZED" {
		return i.success(ctx, webhook)
	} else {
		return i.unexpected(ctx, webhook)
	}
}

func (i *Impl) success(ctx context.Context, webhook nexiapi.WebhookDto) error {
	// validate or create (pending!!) payment with given reference id, we only trust webhooks so much
	prefix := config.TransactionIDPrefix()
	if prefix != "" && !strings.HasPrefix(webhook.TransId, prefix) {
		aulogging.Logger.Ctx(ctx).Warn().Printf("webhook with wrong ref id prefix, ref=%s", webhook.TransId)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: webhook.TransId,
			ApiId:       webhook.PayId,
			Kind:        "error",
			Message:     fmt.Sprintf("webhook %s ref-id-prefix wrong", webhook.Status),
			Details:     fmt.Sprintf("expecting prefix %s", prefix),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", webhook.TransId, "ref-id-prefix-mismatch")
		// report success so they don't retry, it's not a big problem after all
		return nil
	}

	// trust the webhook if no api url configured
	upstreamPayment := nexi.NexiPaymentQueryResponse{
		PayId:   webhook.PayId,
		TransId: webhook.TransId,
		Status:  webhook.Status,
	}
	// if url available, read from API
	if config.NexiDownstreamBaseUrl() != "" {
		var err error
		upstreamPayment, err = nexi.Get().QueryPaymentLink(ctx, webhook.TransId)
		if err != nil {
			aulogging.Logger.Ctx(ctx).Warn().Printf("failed to get payment info from upstream, ref=%s", webhook.TransId)
			db := database.GetRepository()
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: webhook.TransId,
				ApiId:       webhook.PayId,
				Kind:        "error",
				Message:     fmt.Sprintf("webhook %s failed to read payment - continuing with AUTHORIZED only", webhook.Status),
				Details:     err.Error(),
				RequestId:   ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "webhook", webhook.TransId, "failed to read payment from upstream - set to pending - manual intervention needed")
			// reset to minimal info
			upstreamPayment = nexi.NexiPaymentQueryResponse{
				PayId:   webhook.PayId,
				TransId: webhook.TransId,
				Status:  "unknown",
			}
		}
	}

	// fetch transaction data from payment service
	transaction, err := paymentservice.Get().GetTransactionByReferenceId(ctx, webhook.TransId)
	if err != nil {
		if errors.Is(err, paymentservice.NotFoundError) {
			// transaction not found in the payment service -> create one.
			// Note: this should never happen, but we try to recover because someone paid us money for something.
			aulogging.Logger.Ctx(ctx).Error().Printf("webhook ref not found in payment service. Creating new pending transaction and flagging for manual review. ref=%s", webhook.TransId)

			return i.createTransaction(ctx, webhook, upstreamPayment)
		} else {
			aulogging.Logger.Ctx(ctx).Error().Printf("error fetching transaction from payment service. err=%s", err.Error())
			return err
		}
	}

	// matching transaction was found in the payment service database.
	// update the status if applicable.
	return i.updateTransaction(ctx, webhook, transaction, upstreamPayment)
}

func (i *Impl) unexpected(ctx context.Context, webhook nexiapi.WebhookDto) error {
	aulogging.Logger.Ctx(ctx).Error().Printf("unexpected webhook status %s - skipped processing", webhook.Status)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: webhook.TransId,
		ApiId:       webhook.PayId,
		Kind:        "error",
		Message:     fmt.Sprintf("webhook %s unknown status", webhook.Status),
		Details:     fmt.Sprintf("code=%s desc=%s", webhook.ResponseCode, webhook.ResponseDescription),
		RequestId:   ctxvalues.RequestId(ctx),
	})
	_ = i.SendErrorNotifyMail(ctx, "webhook", fmt.Sprintf("unknown status: %s", webhook.Status), "unexpected-status")

	// confirm with 200 so we do not keep receiving the event - we've done all we can
	return nil
}

func (i *Impl) createTransaction(ctx context.Context, data nexiapi.WebhookDto, upstream nexi.NexiPaymentQueryResponse) error {
	debitor_id, err := debitorIdFromReferenceID(data.TransId)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Warn().Printf("webhook couldn't parse debitor_id from transId '%s'", data.TransId)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.TransId,
			ApiId:       data.PayId,
			Kind:        "error",
			Message:     "webhook cannot determine debitor from reference id and payment not found",
			Details:     fmt.Sprintf("amount=%d currency=%s", data.Amount.Value, data.Amount.Currency),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "parse-refid-err")
		// do not continue - we wouldn't know which attendee to associate the payment with. Needs manual investigation
		// confirm with 200 so we do not keep receiving the event - we've done all we can
		return nil
	}

	effective := i.effectiveToday()
	comment := "CC paymentId " + data.PayId + " (auto created - please check and maybe fix tax rate)"

	// no need to validate against API, we're creating pending only, so will be manually checked anyway.
	if upstream.Amount != nil {
		// but warn about different amount / currency:
		if data.Amount.Currency != upstream.Amount.Currency || data.Amount.Value != upstream.Amount.Value {
			comment += " - verified amount/currency differs"
			db := database.GetRepository()
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: data.TransId,
				ApiId:       data.PayId,
				Kind:        "warning",
				Message:     "verified amount/currency differs",
				Details:     fmt.Sprintf("amount=%d %s verified=%d %s", data.Amount.Value, data.Amount.Currency, upstream.Amount.Value, upstream.Amount.Currency),
				RequestId:   ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "amount-or-currency-did-not-verify please manually check")
		}

		comment += fmt.Sprintf(" - paygate status=%s", upstream.Status)
	}

	transaction := paymentservice.Transaction{
		ID:        data.TransId,
		DebitorID: debitor_id,
		Type:      paymentservice.Payment,
		Method:    paymentservice.Credit, // we don't know at this point
		Amount: paymentservice.Amount{
			GrossCent: data.Amount.Value,
			Currency:  data.Amount.Currency,
			VatRate:   0.0,
		},
		Comment:       comment,
		Status:        paymentservice.Pending,
		EffectiveDate: effective,
		DueDate:       effective,
		// omitting Deletion
	}

	err = paymentservice.Get().AddTransaction(ctx, transaction)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf(
			"webhook could not create transaction in payment service! (we don't know why we received this money, and we couldn't add the transaction to the database either!) reference_id=%s",
			data.TransId,
		)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.TransId,
			ApiId:       data.PayId,
			Kind:        "error",
			Message:     "webhook failed to create transaction in payment service",
			Details:     fmt.Sprintf("amount=%d currency=%s error=%s", data.Amount.Value, data.Amount.Currency, err.Error()),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "create-missing-err")
		return err
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("webhook OK amount=%d currency=%s id=%s ref=%s",
		data.Amount.Value,
		data.Amount.Currency,
		data.PayId,
		data.TransId,
	)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: data.TransId,
		ApiId:       data.PayId,
		Kind:        "warning",
		Message:     "webhook created PENDING payment - did not exist - needs review",
		Details:     fmt.Sprintf("amount=%d currency=%s", data.Amount.Value, data.Amount.Currency),
		RequestId:   ctxvalues.RequestId(ctx),
	})
	_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "create-missing-pending-success (needs review and tax rate fix)")
	return nil
}

func (i *Impl) updateTransaction(ctx context.Context, data nexiapi.WebhookDto, transaction paymentservice.Transaction, upstream nexi.NexiPaymentQueryResponse) error {
	if transaction.Status == paymentservice.Valid {
		aulogging.Logger.Ctx(ctx).Warn().Printf(
			"aborting transaction update - already in status %s! reference_id=%s",
			transaction.Status, data.TransId,
		)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.TransId,
			ApiId:       data.PayId,
			Kind:        "warning",
			Message:     fmt.Sprintf("webhook payment already in status %s", transaction.Status),
			Details: fmt.Sprintf("existing_amount=%d ignored_amount=%d existing_currency=%s ignored_currency=%s webhook_status=%s upstream_status=%s",
				transaction.Amount.GrossCent,
				data.Amount.Value,
				transaction.Amount.Currency,
				data.Amount.Currency,
				data.Status,
				upstream.Status),
			RequestId: ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, fmt.Sprintf("abort-update-for-%s-%s", transaction.Status, upstream.Status))
		return nil // not an error
	}

	effective := i.effectiveToday()
	comment := "CC paymentId " + data.PayId

	forcePending := false
	if upstream.Amount != nil {
		// warn about different amount / currency:
		if data.Amount.Currency != upstream.Amount.Currency || data.Amount.Value != upstream.Amount.Value {
			_ = database.GetRepository().WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: data.TransId,
				ApiId:       data.PayId,
				Kind:        "warning",
				Message:     "verified amount/currency differs",
				Details:     fmt.Sprintf("amount=%d %s verified=%d %s", data.Amount.Value, data.Amount.Currency, upstream.Amount.Value, upstream.Amount.Currency),
				RequestId:   ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "amount-or-currency-upstream-difference-kept-pending-please-check")

			comment += " - upstream differences"

			forcePending = true
		}

		if upstream.Status != "OK" {
			_ = database.GetRepository().WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: data.TransId,
				ApiId:       data.PayId,
				Kind:        "warning",
				Message:     "verified status not OK",
				Details:     fmt.Sprintf("webhook=%s verified=%s", data.Status, upstream.Status),
				RequestId:   ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "upstream-status-not-OK-kept-pending-please-check")

			forcePending = true
		}
		comment += fmt.Sprintf(" - status %s", upstream.Status)
	} else {
		// only trust webhook status if upstream not available - means we're using mock/simulator
		if data.Status != "OK" {
			_ = database.GetRepository().WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: data.TransId,
				ApiId:       data.PayId,
				Kind:        "warning",
				Message:     "webhook status not OK",
				Details:     fmt.Sprintf("webhook=%s verified=%s", data.Status, upstream.Status),
				RequestId:   ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "webhook-status-not-OK-kept-pending-please-check")

			forcePending = true
		}
	}

	if transaction.Amount.GrossCent != data.Amount.Value || transaction.Amount.Currency != data.Amount.Currency {
		aulogging.Logger.Ctx(ctx).Warn().Printf("transaction update changes amount or currency! reference_id=%s", data.TransId)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.TransId,
			ApiId:       data.PayId,
			Kind:        "warning",
			Message:     "webhook payment amount differs",
			Details: fmt.Sprintf("old_amount=%d amount=%d old_currency=%s currency=%s",
				transaction.Amount.GrossCent,
				data.Amount.Value,
				transaction.Amount.Currency,
				data.Amount.Currency),
			RequestId: ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "amount-difference-kept-pending-please-check")
		// continue, but keep in pending!

		transaction.Amount.GrossCent = data.Amount.Value
		transaction.Amount.Currency = data.Amount.Currency
		forcePending = true
	}

	if forcePending {
		transaction.Status = paymentservice.Pending
	} else {
		transaction.Status = paymentservice.Valid
	}

	transaction.EffectiveDate = effective
	transaction.Comment = comment

	err := paymentservice.Get().UpdateTransaction(ctx, transaction)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("webhook unable to update upstream transaction. reference_id=%s", data.TransId)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: data.TransId,
			ApiId:       data.PayId,
			Kind:        "error",
			Message:     "webhook failed to update transaction",
			Details:     fmt.Sprintf("amount=%d currency=%s error=%s", data.Amount.Value, data.Amount.Currency, err.Error()),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "webhook", data.TransId, "update-tx-err")
		return err
	}

	aulogging.Logger.Ctx(ctx).Info().Printf("successfully updated upstream transaction to valid. reference_id=%s", data.TransId)
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: data.TransId,
		ApiId:       data.PayId,
		Kind:        "success",
		Message:     "transaction updated successfully",
		Details:     fmt.Sprintf("amount=%d currency=%s", data.Amount.Value, data.Amount.Currency),
		RequestId:   ctxvalues.RequestId(ctx),
	})

	return nil
}

func (i *Impl) effectiveToday() string {
	return i.Now().Format(isoDateFormat)
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
