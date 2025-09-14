package paymentlinksrv

import (
	"context"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/mailservice"
)

func (i *Impl) SendErrorNotifyMail(ctx context.Context, operation string, referenceId string, status string) error {
	notifyMail := config.ErrorNotifyMail()
	if notifyMail == "" {
		aulogging.Logger.Ctx(ctx).Error().Printf("error notification mail cannot be sent - no address configured. Operation: %s, ReferenceId: %s, Status: %s", operation, referenceId, status)
		return nil
	} else {
		aulogging.Logger.Ctx(ctx).Warn().Printf("sending error notification mail - Operation: %s, ReferenceId: %s, Status: %s", operation, referenceId, status)
	}

	mailDto := mailservice.MailSendDto{
		CommonID: "payment-nexi-adapter-error",
		Lang:     "en-US",
		Variables: map[string]string{
			"operation":   operation,
			"referenceId": referenceId,
			"status":      status,
		},
		To: []string{
			notifyMail,
		},
	}

	err := mailservice.Get().SendEmail(ctx, mailDto)
	if err != nil {
		aulogging.Logger.Ctx(ctx).Error().WithErr(err).Printf("failed to send error notification mail to %s - Operation: %s, ReferenceId: %s, Status: %s - error was: %s", notifyMail, operation, referenceId, status, err.Error())
		return err
	}
	return nil
}
