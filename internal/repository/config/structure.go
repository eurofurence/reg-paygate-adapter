package config

type (
	DatabaseType string
)

const (
	Inmemory DatabaseType = "inmemory"
	Mysql    DatabaseType = "mysql"
)

// Application is the root configuration type
type Application struct {
	Service  ServiceConfig  `yaml:"service"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Security SecurityConfig `yaml:"security"`
	Invoice  InvoiceConfig  `yaml:"invoice"`
}

// ServerConfig contains all values for http configuration
type ServerConfig struct {
	Address      string `yaml:"address"`
	Port         uint16 `yaml:"port"`
	ReadTimeout  int    `yaml:"read_timeout_seconds"`
	WriteTimeout int    `yaml:"write_timeout_seconds"`
	IdleTimeout  int    `yaml:"idle_timeout_seconds"`
}

// ServiceConfig contains configuration values
// for service related tasks. E.g. URLs to downstream services
type ServiceConfig struct {
	Name                    string `yaml:"name"`
	PublicURL               string `yaml:"public_url"`                 // my own public base url, without a trailing slash
	AttendeeService         string `yaml:"attendee_service"`           // base url, usually http://localhost:nnnn, will use in-memory-mock if unset
	MailService             string `yaml:"mail_service"`               // base url, usually http://localhost:nnnn, will use in-memory-mock if unset
	PaymentService          string `yaml:"payment_service"`            // base url, usually http://localhost:nnnn, will use in-memory-mock if unset
	NexiDownstream          string `yaml:"nexi_downstream"`            // base url, usually https://api.dibspayment.eu for new API, will use in-memory-mock if unset
	NexiInstance            string `yaml:"nexi_instance"`              // for new API: NexiApiKey (not used?)
	NexiApiKey              string `yaml:"nexi_api_key"`               // for new API: authorization header (secret api key)
	NexiMerchantID          string `yaml:"nexi_merchant_id"`           // for new API: merchant id
	SuccessRedirect         string `yaml:"success_redirect"`
	FailureRedirect         string `yaml:"failure_redirect"`
	TransactionIDPrefix     string `yaml:"transaction_id_prefix"`
	TermsURL                string `yaml:"terms_url"` // our terms, required
}

// DatabaseConfig configures which db to use (mysql, inmemory)
// and how to connect to it (needed for mysql only)
type DatabaseConfig struct {
	Use        DatabaseType `yaml:"use"`
	Username   string       `yaml:"username"`
	Password   string       `yaml:"password"`
	Database   string       `yaml:"database"`
	Parameters []string     `yaml:"parameters"`
}

// SecurityConfig configures everything related to incoming request security
type SecurityConfig struct {
	Fixed FixedTokenConfig `yaml:"fixed_token"`
	Cors  CorsConfig       `yaml:"cors"`
}

type CorsConfig struct {
	DisableCors bool   `yaml:"disable"`
	AllowOrigin string `yaml:"allow_origin"`
}

type FixedTokenConfig struct {
	Api     string `yaml:"api"`     // shared-secret for server-to-server backend authentication
	Webhook string `yaml:"webhook"` // shared-secret for the webhook coming in from nexi
}

// LoggingConfig configures logging
type LoggingConfig struct {
	Severity        string `yaml:"severity"`
	FullRequests    bool   `yaml:"full_requests"`
	ErrorNotifyMail string `yaml:"error_notify_mail"`
}

// InvoiceConfig defines what the invoices should look like
type InvoiceConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Purpose     string `yaml:"purpose"`
}
