package paymentlinksrv

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/attendeeservice"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/web/util/ctxvalues"

	"github.com/eurofurence/reg-paygate-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
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
	taxAmountCents := int32(math.Round(float64(data.AmountDue) * data.VatRate / 100.0))

	request := nexi.NexiCreatePaymentRequest{
		Order: nexi.NexiOrder{
			Items: []nexi.NexiOrderItem{
				{
					Reference:        config.InvoicePurpose(), // SKU
					Name:             config.InvoiceTitle(),
					Quantity:         1.0,
					Unit:             "pcs",
					UnitPrice:        data.AmountDue - taxAmountCents,
					TaxRate:          int32(math.Round(data.VatRate * 100.0)), // in increments of 0.01%
					TaxAmount:        taxAmountCents,
					GrossTotalAmount: data.AmountDue,
					NetTotalAmount:   data.AmountDue - taxAmountCents,
					ImageUrl:         p(""),
				},
			},
			Amount:    data.AmountDue,
			Currency:  data.Currency,
			Reference: data.ReferenceId,
		},
		Checkout: nexi.NexiCheckout{
			Url:             p(""),
			IntegrationType: "HostedPaymentPage", // case-sensitive - was hostedPaymentPage?
			ReturnUrl:       config.SuccessRedirect(),
			CancelUrl:       config.FailureRedirect(),
			Consumer: &nexi.NexiConsumer{
				Email: p(attendee.Email),
			},
			TermsUrl:         config.TermsURL(),
			MerchantTermsUrl: p(""),
			//ShippingCountries: []nexi.NexiCountry{ // optional
			//	{CountryCode: "DEU"},
			//},
			//Shipping: nexi.NexiShipping{ // optional
			//	Countries: []nexi.NexiCountry{
			//		{CountryCode: "DEU"},
			//	},
			//	MerchantHandlesShippingCost: false,
			//	EnableBillingAddress:        true,
			//},
			//ConsumerType: &nexi.NexiConsumerType{
			//	Default:        "b2c",
			//	SupportedTypes: []string{"b2c", "b2b"},
			//},
			Charge:                      false,
			PublicDevice:                false,
			MerchantHandlesConsumerData: false,
			CountryCode:                 p("DEU"),
			Appearance: &nexi.NexiAppearance{
				DisplayOptions: nexi.NexiDisplayOptions{
					ShowMerchantName: true,
					ShowOrderSummary: true,
				},
				TextOptions: nexi.NexiTextOptions{
					CompletePaymentButtonText: "pay",
				},
			},
		},
	}
	if config.ServicePublicURL() != "" {
		url := config.ServicePublicURL() + "/api/rest/v1/webhook/" + config.WebhookSecret()
		request.Notifications = &nexi.NexiNotifications{
			Webhooks: []nexi.NexiWebhook{
				{
					EventName: nexiapi.EventPaymentCheckoutCompleted,
					Url:       url,
				},
				{
					EventName: nexiapi.EventPaymentCancelCreated,
					Url:       url,
				},
				{
					EventName: nexiapi.EventPaymentChargeCreated,
					Url:       url,
				},
				{
					EventName: nexiapi.EventPaymentChargeCreatedV2,
					Url:       url,
				},
				{
					EventName: nexiapi.EventPaymentChargeFailed,
					Url:       url,
				},
				{
					EventName: nexiapi.EventPaymentChargeFailedV2,
					Url:       url,
				},
				{
					EventName: nexiapi.EventPaymentCreated,
					Url:       url,
				},
			},
		}
	}
	return request
}

func (i *Impl) apiResponseFromNexiResponse(response nexi.NexiPaymentLinkCreated, request nexi.NexiCreatePaymentRequest) nexiapi.PaymentLinkDto {
	return nexiapi.PaymentLinkDto{
		Title:       config.InvoiceTitle(),
		Description: config.InvoiceDescription(),
		ReferenceId: request.Order.Reference,
		Purpose:     config.InvoicePurpose(),
		AmountDue:   int64(request.Order.Amount),
		AmountPaid:  0,
		Currency:    request.Order.Currency,
		VatRate:     float64(request.Order.Items[0].TaxRate) / 100.0,
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
		AmountDue:   int64(data.Amount),
		AmountPaid:  0, // TODO calculate paid amount from summary
		Currency:    data.Currency,
		Link:        data.Link,
	}
	if len(data.Order.Items) > 0 {
		result.VatRate = float64(data.Order.Items[0].TaxRate) / 100.0
		result.Title = data.Order.Items[0].Name
		result.Description = "" // TODO
	}

	return result, nil
}

func (i *Impl) DeletePaymentLink(ctx context.Context, id string) error {
	// Query to get the amount for cancel
	data, err := nexi.Get().QueryPaymentLink(ctx, id)
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

	amount := data.Amount

	err = nexi.Get().DeletePaymentLink(ctx, id, amount)
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

func p[T comparable](t T) *T {
	var nullValue T
	if t == nullValue {
		return nil
	}
	return &t
}
