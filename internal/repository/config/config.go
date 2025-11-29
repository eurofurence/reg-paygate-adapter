// configuration management using a yaml configuration file
// You must have called LoadConfiguration() or otherwise set up the configuration before you can use these.
package config

import (
	"fmt"
	"strings"
	"time"
)

func UseEcsLogging() bool {
	return ecsLogging
}

func ServerAddr() string {
	c := Configuration()
	return fmt.Sprintf("%s:%d", c.Server.Address, c.Server.Port)
}

func ServerReadTimeout() time.Duration {
	return time.Second * time.Duration(Configuration().Server.ReadTimeout)
}

func ServerWriteTimeout() time.Duration {
	return time.Second * time.Duration(Configuration().Server.WriteTimeout)
}

func ServicePublicURL() string {
	return Configuration().Service.PublicURL
}

func ServerIdleTimeout() time.Duration {
	return time.Second * time.Duration(Configuration().Server.IdleTimeout)
}

func DatabaseUse() DatabaseType {
	return Configuration().Database.Use
}

func DatabaseMysqlConnectString() string {
	c := Configuration().Database
	return c.Username + ":" + c.Password + "@" +
		c.Database + "?" + strings.Join(c.Parameters, "&")
}

func MigrateDatabase() bool {
	return dbMigrate
}

func LoggingSeverity() string {
	return Configuration().Logging.Severity
}

func LogFullRequests() bool {
	return Configuration().Logging.FullRequests
}

func ErrorNotifyMail() string {
	return Configuration().Logging.ErrorNotifyMail
}

func FixedApiToken() string {
	return Configuration().Security.Fixed.Api
}

func IsCorsDisabled() bool {
	return Configuration().Security.Cors.DisableCors
}

func AttendeeServiceBaseUrl() string {
	return Configuration().Service.AttendeeService
}

func MailServiceBaseUrl() string {
	return Configuration().Service.MailService
}

func PaymentServiceBaseUrl() string {
	return Configuration().Service.PaymentService
}

func NexiDownstreamBaseUrl() string {
	return Configuration().Service.NexiDownstream
}

func NexiInstanceName() string {
	return Configuration().Service.NexiInstance
}

func NexiInstanceApiSecret() string {
	return Configuration().Service.NexiApiSecret
}

func WebhookSecret() string {
	return Configuration().Security.Fixed.Webhook
}

func TransactionIDPrefix() string {
	return Configuration().Service.TransactionIDPrefix
}

func InvoiceTitle() string {
	return Configuration().Invoice.Title
}

func InvoiceDescription() string {
	return Configuration().Invoice.Description
}

func InvoicePurpose() string {
	return Configuration().Invoice.Purpose
}

func SuccessRedirect() string {
	return Configuration().Service.SuccessRedirect
}

func FailureRedirect() string {
	return Configuration().Service.FailureRedirect
}

func CommercePlatformTag() string {
	return Configuration().Service.NexiCommercePlatformTag
}

func NexiMerchantNumber() string {
	return Configuration().Service.NexiMerchantNumber
}

func TermsURL() string {
	return Configuration().Service.TermsURL
}
