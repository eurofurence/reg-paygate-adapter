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
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/nexi"
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

func (i *Impl) CreatePaymentLink(ctx context.Context, data nexiapi.PaymentLinkRequestDto) (nexiapi.PaymentLinkDto, string, error) {
	attendee, err := attendeeservice.Get().GetAttendee(ctx, uint(data.DebitorId))
	if err != nil {
		return nexiapi.PaymentLinkDto{}, "", err
	}

	nexiRequest := i.nexiCreateRequestFromApiRequest(data, attendee)
	nexiResponse, err := nexi.Get().CreatePaymentLink(ctx, nexiRequest)
	if err != nil {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: nexiRequest.Order.Reference,
			Kind:        "error",
			Message:     "create-pay-link failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "create-pay-link", data.ReferenceId, err.Error())
		return nexiapi.PaymentLinkDto{}, "", err
	}
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: nexiRequest.Order.Reference,
		ApiId:       nexiResponse.ID,
		Kind:        "success",
		Message:     "create-pay-link",
		Details:     nexiResponse.Link,
		RequestId:   ctxvalues.RequestId(ctx),
	})
	output := i.apiResponseFromNexiResponse(nexiResponse, nexiRequest)
	return output, nexiResponse.ID, nil
}

func (i *Impl) nexiCreateRequestFromApiRequest(data nexiapi.PaymentLinkRequestDto, attendee attendeeservice.AttendeeDto) nexi.NexiCreatePaymentRequest {
	shortenedOrderId := strings.ReplaceAll(data.ReferenceId, "-", "")
	if len(shortenedOrderId) > 30 {
		shortenedOrderId = shortenedOrderId[:30]
	}
	return nexi.NexiCreatePaymentRequest{
		Order: nexi.NexiOrder{
			Items: []nexi.NexiOrderItem{
				{
					Reference:        config.InvoicePurpose(),
					Name:             config.InvoiceTitle(),
					Quantity:         1.0,
					Unit:             "qty",
					UnitPrice:        data.AmountDue,
					TaxRate:          data.VatRate,
					TaxAmount:        0, // calculate if needed
					GrossTotalAmount: data.AmountDue,
					NetTotalAmount:   data.AmountDue,
					ImageUrl:         "",
				},
			},
			Amount:    data.AmountDue,
			Currency:  data.Currency,
			Reference: data.ReferenceId,
		},
		Checkout: nexi.NexiCheckout{
			Url:             "",
			IntegrationType: "hostedPaymentPage",
			ReturnUrl:       config.SuccessRedirect(),
			CancelUrl:       config.FailureRedirect(),
			Consumer: nexi.NexiConsumer{
				Reference: attendee.Email,
				Email:     attendee.Email,
			},
			TermsUrl:         "",
			MerchantTermsUrl: "",
			ShippingCountries: []nexi.NexiCountry{
				{CountryCode: "DE"},
			},
			Shipping: nexi.NexiShipping{
				Countries: []nexi.NexiCountry{
					{CountryCode: "DE"},
				},
				MerchantHandlesShippingCost: false,
				EnableBillingAddress:        true,
			},
			ConsumerType: nexi.NexiConsumerType{
				Default:        "b2c",
				SupportedTypes: []string{"b2c", "b2b"},
			},
			Charge:                      true,
			PublicDevice:                false,
			MerchantHandlesConsumerData: false,
			Appearance: nexi.NexiAppearance{
				DisplayOptions: nexi.NexiDisplayOptions{
					ShowMerchantName: true,
					ShowOrderSummary: true,
				},
				TextOptions: nexi.NexiTextOptions{
					CompletePaymentButtonText: "Complete Payment",
				},
			},
			CountryCode: "DE",
		},
		MerchantNumber: config.NexiMerchantNumber(),
	}
}

func (i *Impl) apiResponseFromNexiResponse(response nexi.NexiPaymentLinkCreated, request nexi.NexiCreatePaymentRequest) nexiapi.PaymentLinkDto {
	return nexiapi.PaymentLinkDto{
		Title:       config.InvoiceTitle(),
		Description: config.InvoiceDescription(),
		ReferenceId: response.ReferenceID,
		Purpose:     config.InvoicePurpose(),
		AmountDue:   request.Order.Amount,
		AmountPaid:  0,
		Currency:    request.Order.Currency,
		VatRate:     request.Order.Items[0].TaxRate,
		Link:        response.Link,
	}
}

func (i *Impl) GetPaymentLink(ctx context.Context, id string) (nexiapi.PaymentLinkDto, error) {
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
		_ = i.SendErrorNotifyMail(ctx, "get-pay-link", fmt.Sprintf("paylink id %s", id), err.Error())
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
		Purpose:     config.InvoicePurpose(),
		AmountDue:   data.Amount,
		AmountPaid:  0, // TODO calculate paid amount from summary
		Currency:    data.Currency,
		Link:        data.Link,
	}
	if len(data.Order.Items) > 0 {
		result.VatRate = data.Order.Items[0].TaxRate
		result.Title = data.Order.Items[0].Name
		result.Description = "" // TODO
	}

	return result, nil
}

func (i *Impl) DeletePaymentLink(ctx context.Context, id string) error {
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
		_ = i.SendErrorNotifyMail(ctx, "delete-pay-link", fmt.Sprintf("paylink id %s", id), err.Error())
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
