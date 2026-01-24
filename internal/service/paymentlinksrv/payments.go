package paymentlinksrv

import (
	"context"
	"fmt"

	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"

	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
)

func (i *Impl) GetPayment(ctx context.Context, id string) (nexiapi.PaymentDto, error) {
	data, err := nexi.Get().QueryPaymentLink(ctx, id)
	if err != nil {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: id,
			Kind:        "error",
			Message:     "get-payment failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "get-payment", fmt.Sprintf("reference id %s", id), err.Error())
		return nexiapi.PaymentDto{}, err
	}

	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: id,
		ApiId:       data.PayId,
		Kind:        "success",
		Message:     "get-payment",
		Details:     "",
		RequestId:   ctxvalues.RequestId(ctx),
	})

	amountDue := int64(0)
	amountPaid := int64(0)
	currency := ""
	if data.Amount != nil {
		amountDue = data.Amount.Value
		if data.Amount.CapturedValue != nil {
			amountPaid = *data.Amount.CapturedValue
		}
		currency = data.Amount.Currency
	}

	method := ""
	if data.PaymentMethods != nil {
		method = data.PaymentMethods.Type
	}

	result := nexiapi.PaymentDto{
		Id:            data.PayId,
		ReferenceId:   id,
		AmountDue:     amountDue,
		AmountPaid:    amountPaid,
		Currency:      currency,
		Status:        data.Status,
		ResponseCode:  data.ResponseCode,
		PaymentMethod: method,
	}

	return result, nil
}
