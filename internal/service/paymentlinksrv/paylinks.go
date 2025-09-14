package paymentlinksrv

import (
	"context"
	"fmt"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/entity"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/attendeeservice"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/database"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctxvalues"
	"net/url"
	"strings"

	"github.com/eurofurence/reg-payment-nexi-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/nexi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
)

func (i *Impl) ValidatePaymentLinkRequest(ctx context.Context, data nexiapi.PaymentLinkRequestDto) url.Values {
	errs := url.Values{}

	if data.DebitorId == 0 {
		errs.Add("debitor_id", "field must be a positive integer (the badge number to bill for)")
	}
	if data.AmountDue <= 0 {
		errs.Add("amount_due", "must be a positive integer (the amount to bill)")
	}
	if data.Currency != "EUR" {
		errs.Add("currency", "right now, only EUR is supported")
	}
	if data.VatRate < 0.0 || data.VatRate > 50.0 {
		errs.Add("vat_rate", "vat rate should be provided in percent and must be between 0.0 and 50.0")
	}

	if len(errs) == 0 {
		return nil
	} else {
		return errs
	}
}

func (i *Impl) CreatePaymentLink(ctx context.Context, data nexiapi.PaymentLinkRequestDto) (nexiapi.PaymentLinkDto, uint, error) {
	attendee, err := attendeeservice.Get().GetAttendee(ctx, uint(data.DebitorId))
	if err != nil {
		return nexiapi.PaymentLinkDto{}, 0, err
	}

	nexiRequest := i.nexiCreateRequestFromApiRequest(data, attendee)
	nexiResponse, err := nexi.Get().CreatePaymentLink(ctx, nexiRequest)
	if err != nil {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: nexiRequest.ReferenceId,
			ApiId:       nexiResponse.ID,
			Kind:        "error",
			Message:     "create-pay-link failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "create-pay-link", data.ReferenceId, err.Error())
		return nexiapi.PaymentLinkDto{}, 0, err
	}
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: nexiRequest.ReferenceId,
		ApiId:       nexiResponse.ID,
		Kind:        "success",
		Message:     "create-pay-link",
		Details:     nexiResponse.Link,
		RequestId:   ctxvalues.RequestId(ctx),
	})
	output := i.apiResponseFromNexiResponse(nexiResponse, nexiRequest)
	return output, nexiResponse.ID, nil
}

func (i *Impl) nexiCreateRequestFromApiRequest(data nexiapi.PaymentLinkRequestDto, attendee attendeeservice.AttendeeDto) nexi.PaymentLinkCreateRequest {
	shortenedOrderId := strings.ReplaceAll(data.ReferenceId, "-", "")
	if len(shortenedOrderId) > 30 {
		shortenedOrderId = shortenedOrderId[:30]
	}
	return nexi.PaymentLinkCreateRequest{
		Title:       config.InvoiceTitle(),
		Description: config.InvoiceDescription(),
		PSP:         1,
		ReferenceId: data.ReferenceId,
		OrderId:     shortenedOrderId,
		Purpose:     config.InvoicePurpose(),
		Amount:      data.AmountDue,
		VatRate:     data.VatRate,
		Currency:    data.Currency,
		SKU:         "registration",
		Email:       attendee.Email,

		SuccessRedirectUrl: config.SuccessRedirect(),
		FailedRedirectUrl:  config.FailureRedirect(),
	}
}

func (i *Impl) apiResponseFromNexiResponse(response nexi.PaymentLinkCreated, request nexi.PaymentLinkCreateRequest) nexiapi.PaymentLinkDto {
	return nexiapi.PaymentLinkDto{
		Title:       request.Title,
		Description: request.Description,
		ReferenceId: response.ReferenceID,
		Purpose:     request.Purpose,
		AmountDue:   request.Amount,
		AmountPaid:  0,
		Currency:    request.Currency,
		VatRate:     request.VatRate,
		Link:        response.Link,
	}
}

func (i *Impl) GetPaymentLink(ctx context.Context, id uint) (nexiapi.PaymentLinkDto, error) {
	data, err := nexi.Get().QueryPaymentLink(ctx, id)
	if err != nil {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: "",
			ApiId:       id,
			Kind:        "error",
			Message:     "get-pay-link failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "get-pay-link", fmt.Sprintf("paylink id %d", id), err.Error())
		return nexiapi.PaymentLinkDto{}, err
	}

	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: data.ReferenceID,
		ApiId:       id,
		Kind:        "success",
		Message:     "get-pay-link",
		Details:     data.Link,
		RequestId:   ctxvalues.RequestId(ctx),
	})

	// TODO lots of missing fields, can we get them from downstream?

	result := nexiapi.PaymentLinkDto{
		ReferenceId: data.ReferenceID,
		Purpose:     data.Purpose["1"],
		AmountDue:   data.Amount,
		AmountPaid:  0,
		Currency:    data.Currency,
		Link:        data.Link,
	}

	return result, nil
}

func (i *Impl) DeletePaymentLink(ctx context.Context, id uint) error {
	err := nexi.Get().DeletePaymentLink(ctx, id)
	if err != nil {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: "",
			ApiId:       id,
			Kind:        "error",
			Message:     "delete-pay-link failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "delete-pay-link", fmt.Sprintf("paylink id %d", id), err.Error())
		return err
	}

	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: "",
		ApiId:       id,
		Kind:        "success",
		Message:     "delete-pay-link",
		Details:     "",
		RequestId:   ctxvalues.RequestId(ctx),
	})

	return nil
}
