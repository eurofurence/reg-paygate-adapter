package paymentlinksrv

import (
	"context"
	"fmt"
	"math"
	"net/url"

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
			ReferenceId: nexiRequest.TransId,
			Kind:        "error",
			Message:     "create-pay-link failed",
			Details:     err.Error(),
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "create-pay-link", data.ReferenceId, err.Error())
		return nexiapi.PaymentLinkDto{}, "", err
	}
	redirect := nexiResponse.Links.Redirect
	if redirect == nil || redirect.Href == "" {
		db := database.GetRepository()
		_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
			ReferenceId: nexiRequest.TransId,
			Kind:        "error",
			Message:     "create-pay-link empty",
			Details:     "response did not include a redirect link",
			RequestId:   ctxvalues.RequestId(ctx),
		})
		_ = i.SendErrorNotifyMail(ctx, "create-pay-link", data.ReferenceId, "response did not include a redirect link")
		return nexiapi.PaymentLinkDto{}, "", ReceivedEmptyPaylink
	}
	db := database.GetRepository()
	_ = db.WriteProtocolEntry(ctx, &entity.ProtocolEntry{
		ReferenceId: nexiRequest.TransId,
		Kind:        "success",
		Message:     "create-pay-link",
		Details:     redirect.Href,
		RequestId:   ctxvalues.RequestId(ctx),
	})
	output := i.apiResponseFromNexiResponse(nexiResponse, nexiRequest)
	return output, nexiRequest.TransId, nil
}

func (i *Impl) nexiCreateRequestFromApiRequest(data nexiapi.PaymentLinkRequestDto, attendee attendeeservice.AttendeeDto) nexi.NexiCreateCheckoutSessionRequest {
	amountDue := int64(data.AmountDue)
	taxAmountCents := int64(math.Round(float64(data.AmountDue) * data.VatRate / 100.0))
	netItemTotal := amountDue - taxAmountCents

	language := "en"
	if attendee.RegistrationLanguage == "de-DE" {
		language = "de"
	}

	webhook := ""
	if config.WebhookOverrideURL() != "" {
		webhook = config.WebhookOverrideURL() + "/api/rest/v1/weblogger/" + config.WebhookSecret()
	} else if config.ServicePublicURL() != "" {
		webhook = config.ServicePublicURL() + "/api/rest/v1/webhook/" + config.WebhookSecret()
	}

	request := nexi.NexiCreateCheckoutSessionRequest{
		TransId: data.ReferenceId,
		Amount: nexi.NexiAmount{
			Value:        amountDue,
			Currency:     data.Currency,
			TaxTotal:     &taxAmountCents,
			NetItemTotal: &netItemTotal,
		},
		Language: language,
		Urls: nexi.NexiPaymentUrlsRequest{
			Return:  config.SuccessRedirect(),
			Cancel:  config.FailureRedirect(),
			Webhook: webhook,
		},
		StatementDescriptor: config.InvoiceTitle(),
		Order: &nexi.NexiOrder{
			NumberOfArticles: 1,
			Items: []nexi.NexiOrderItem{
				{
					Name:    config.InvoicePurpose(),
					TaxRate: int64(math.Round(data.VatRate * 100.0)),
				},
			},
		},
		CustomerInfo: &nexi.NexiCustomerInfoRequest{
			Email: attendee.Email,
		},
		PaymentMethods: &nexi.NexiPaymentMethodsRequest{
			Card: &nexi.NexiCard{
				Template: &nexi.NexiCardTemplate{
					CustomFields: &nexi.NexiCardTemplateCustomFields{
						CustomField1: fmt.Sprintf("%.2f %s", float64(amountDue)/100.0, data.Currency), // Amount and currency of the transaction
						// CustomField3:  "",                      // Merchant’s logo, URL of the logo. Format: .png (any size)
						CustomField4: config.InvoicePurpose(), // order's descriptions
					},
				},
			},
		},
	}

	if config.NexiSimulationMode() {
		request.SimulationMode = "0000"
	}

	return request
}

func (i *Impl) apiResponseFromNexiResponse(response nexi.NexiCreateCheckoutSessionResponse, request nexi.NexiCreateCheckoutSessionRequest) nexiapi.PaymentLinkDto {
	vatRate := float64(0.0)
	if request.Order != nil && len(request.Order.Items) > 0 {
		// avoid panics, live with vatRate 0 in response if not available
		vatRate = float64(request.Order.Items[0].TaxRate) / 100.0
	}
	return nexiapi.PaymentLinkDto{
		Title:       config.InvoiceTitle(),
		Description: config.InvoiceDescription(),
		ReferenceId: request.TransId,
		Purpose:     config.InvoicePurpose(),
		AmountDue:   request.Amount.Value,
		AmountPaid:  0,
		Currency:    request.Amount.Currency,
		VatRate:     vatRate,
		Link:        response.Links.Redirect.Href,
	}
}

func p[T comparable](t T) *T {
	var nullValue T
	if t == nullValue {
		return nil
	}
	return &t
}
