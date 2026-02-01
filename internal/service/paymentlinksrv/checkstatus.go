package paymentlinksrv

import (
	"context"
	"fmt"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/paymentservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"
)

func (i *Impl) CheckPaymentStatus(ctx context.Context, id string) (nexiapi.PaymentDto, error) {
	if config.NexiDownstreamBaseUrl() == "" {
		return nexiapi.PaymentDto{}, nexi.NotConfigured
	}

	// check exists at Paygate
	nexiDto, err := i.GetPayment(ctx, id)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("error fetching payment from paygate API. err=%s", err.Error())
		return nexiapi.PaymentDto{}, err
	}

	// check exists in payment service
	transaction, err := paymentservice.Get().GetTransactionByReferenceId(ctx, id)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().Printf("error fetching transaction from payment service. err=%s", err.Error())
		return nexiapi.PaymentDto{}, err
	}

	if nexiDto.Status == "OK" && nexiDto.ResponseCode == "00000000" {
		aulogging.Logger.Ctx(ctx).Info().Printf("paygate status is OK, checking payment status. reference_id=%s", id)

		if transaction.Status != paymentservice.Pending && transaction.Status != paymentservice.Tentative {
			aulogging.Logger.Ctx(ctx).Warn().Printf(
				"aborting transaction update - currently in status %s! reference_id=%s", transaction.Status, id,
			)
			db := database.GetRepository()
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: id,
				ApiId:       nexiDto.Id,
				Kind:        "warning",
				Message:     fmt.Sprintf("status-check: payment in status %s - skipping update", transaction.Status),
				Details: fmt.Sprintf("transaction_status=%s upstream_status=%s",
					transaction.Status,
					nexiDto.Status),
				RequestId: ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "status-check", id, fmt.Sprintf("abort-update-for-%s-%s", transaction.Status, nexiDto.Status))
			return nexiDto, TransactionStatusError
		}

		if transaction.Amount.GrossCent != nexiDto.AmountPaid || transaction.Amount.Currency != nexiDto.Currency {
			aulogging.Logger.Ctx(ctx).Warn().Printf(
				"aborting transaction update - currency or amount differs - please check! reference_id=%s", id,
			)
			db := database.GetRepository()
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: id,
				ApiId:       nexiDto.Id,
				Kind:        "warning",
				Message:     fmt.Sprintf("status-check: amount or currency differs - skipping update"),
				Details: fmt.Sprintf("tx_amount=%d upstream_amount=%d tx_currency=%s upstream_currency=%s transaction_status=%s upstream_status=%s",
					transaction.Amount.GrossCent,
					nexiDto.AmountPaid,
					transaction.Amount.Currency,
					nexiDto.Currency,
					transaction.Status,
					nexiDto.Status),
				RequestId: ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "status-check", id, "abort-update-values-differ")
			return nexiDto, TransactionDataMismatchError
		}

		transaction.Status = paymentservice.Valid
		transaction.Comment = "CC paymentId " + nexiDto.Id

		err = paymentservice.Get().UpdateTransaction(ctx, transaction)
		if err != nil {
			aulogging.Logger.Ctx(ctx).Error().Printf("status-check unable to update upstream transaction. reference_id=%s", id)
			db := database.GetRepository()
			_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
				ReferenceId: id,
				ApiId:       nexiDto.Id,
				Kind:        "error",
				Message:     "status-check failed to update transaction",
				Details:     fmt.Sprintf("amount=%d currency=%s error=%s", transaction.Amount.GrossCent, transaction.Amount.Currency, err.Error()),
				RequestId:   ctxvalues.RequestId(ctx),
			})
			_ = i.SendErrorNotifyMail(ctx, "status-check", id, "update-tx-err")
			return nexiDto, err
		}

		aulogging.Logger.Ctx(ctx).Info().Printf("status-check: successfully updated upstream transaction to valid. reference_id=%s", id)
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: id,
			ApiId:       nexiDto.Id,
			Kind:        "success",
			Message:     "transaction updated successfully by status-check",
			Details:     fmt.Sprintf("amount=%d currency=%s", transaction.Amount.GrossCent, transaction.Amount.Currency),
			RequestId:   ctxvalues.RequestId(ctx),
		})
	}

	return nexiDto, nil
}
