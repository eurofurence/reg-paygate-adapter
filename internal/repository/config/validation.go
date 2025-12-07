package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
)

func setConfigurationDefaults(c *Application) {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout <= 0 {
		c.Server.ReadTimeout = 5
	}
	if c.Server.WriteTimeout <= 0 {
		c.Server.WriteTimeout = 5
	}
	if c.Server.IdleTimeout <= 0 {
		c.Server.IdleTimeout = 5
	}
	if c.Logging.Severity == "" {
		c.Logging.Severity = "INFO"
	}
}

const (
	envNexiApiSecret             = "REG_SECRET_NEXI_API_SECRET"
	envNexiIncomingWebhookSecret = "REG_SECRET_NEXI_INCOMING_WEBHOOK_SECRET"
	envApiToken                  = "REG_SECRET_API_TOKEN"
	envDbPassword                = "REG_SECRET_DB_PASSWORD"
)

func applyEnvVarOverrides(c *Application) {
	if nexiApiSecret := os.Getenv(envNexiApiSecret); nexiApiSecret != "" {
		c.Service.NexiApiSecret = nexiApiSecret
	}
	if nexiIncomingWebhookSecret := os.Getenv(envNexiIncomingWebhookSecret); nexiIncomingWebhookSecret != "" {
		c.Security.Fixed.Webhook = nexiIncomingWebhookSecret
	}
	if apiToken := os.Getenv(envApiToken); apiToken != "" {
		c.Security.Fixed.Api = apiToken
	}
	if dbPassword := os.Getenv(envDbPassword); dbPassword != "" {
		c.Database.Password = dbPassword
	}
}

func validateServerConfiguration(errs url.Values, c ServerConfig) {
	checkIntValueRange(&errs, 1024, 65535, "server.port", int(c.Port))
	checkIntValueRange(&errs, 1, 300, "server.read_timeout_seconds", c.ReadTimeout)
	checkIntValueRange(&errs, 1, 300, "server.write_timeout_seconds", c.WriteTimeout)
	checkIntValueRange(&errs, 1, 300, "server.idle_timeout_seconds", c.IdleTimeout)
}

var allowedDatabases = []DatabaseType{Mysql, Inmemory}

func validateDatabaseConfiguration(errs url.Values, c DatabaseConfig) {
	if notInAllowedValues(allowedDatabases, c.Use) {
		errs.Add("database.use", "must be one of mysql, inmemory")
	}
	if c.Use == Mysql {
		checkLength(&errs, 1, 256, "database.username", c.Username)
		checkLength(&errs, 1, 256, "database.password", c.Password)
		checkLength(&errs, 1, 256, "database.database", c.Database)
	}
}

var allowedSeverities = []string{"DEBUG", "INFO", "WARN", "ERROR"}

func validateLoggingConfiguration(errs url.Values, c LoggingConfig) {
	if notInAllowedValues(allowedSeverities[:], c.Severity) {
		errs.Add("logging.severity", "must be one of DEBUG, INFO, WARN, ERROR")
	}
}

func validateSecurityConfiguration(errs url.Values, c SecurityConfig) {
	checkLength(&errs, 16, 256, "security.fixed.api", c.Fixed.Api)
	checkLength(&errs, 8, 64, "security.fixed.webhook", c.Fixed.Webhook)
}

const urlPattern = "^https?://.*$"
const downstreamPattern = "^(|https?://.*[^/])$"

func validateServiceConfiguration(errs url.Values, c ServiceConfig) {
	if violatesPattern(downstreamPattern, c.AttendeeService) {
		errs.Add("service.attendee_service", "base url must be empty (enables in-memory simulator) or start with http:// or https:// and may not end in a /")
	}
	if violatesPattern(downstreamPattern, c.PaymentService) {
		errs.Add("service.payment_service", "base url must be empty (enables in-memory simulator) or start with http:// or https:// and may not end in a /")
	}
	if violatesPattern(downstreamPattern, c.NexiDownstream) {
		errs.Add("service.nexi_downstream", "base url must be empty (enables local simulator) or start with http:// or https:// and may not end in a /")
	}
	if violatesPattern(downstreamPattern, c.PublicURL) {
		errs.Add("service.public_url", "public url must be empty or start with http:// or https:// and may not end in a /")
	}
	if violatesPattern(urlPattern, c.TermsURL) {
		errs.Add("service.terms_url", "terms url must start with http:// or https://, and cannot be empty")
	}
	if c.NexiDownstream == "" && c.PublicURL == "" {
		errs.Add("service.public_url", "must set public_url (for simulated paylinks) if nexi_downstream (to talk to actual api) is blank")
	}
	checkLength(&errs, 1, 256, "service.nexi_instance", c.NexiInstance)
	checkLength(&errs, 1, 256, "service.nexi_api_secret", c.NexiApiSecret)
}

func validateInvoiceConfiguration(errs url.Values, c InvoiceConfig) {
	checkLength(&errs, 1, 256, "invoice.title", c.Title)
	checkLength(&errs, 1, 256, "invoice.purpose", c.Purpose)
	checkLength(&errs, 1, 256, "invoice.description", c.Description)
}

// -- helpers

func violatesPattern(pattern string, value string) bool {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return true
	}
	return !matched
}

func checkLength(errs *url.Values, min int, max int, key string, value string) {
	if len(value) < min || len(value) > max {
		errs.Add(key, fmt.Sprintf("%s field must be at least %d and at most %d characters long", key, min, max))
	}
}

func checkIntValueRange(errs *url.Values, min int, max int, key string, value int) {
	if value < min || value > max {
		errs.Add(key, fmt.Sprintf("%s field must be an integer at least %d and at most %d", key, min, max))
	}
}

func notInAllowedValues[T comparable](allowed []T, value T) bool {
	return !sliceContains(allowed, value)
}

func sliceContains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
