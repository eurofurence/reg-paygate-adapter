package config

import (
	"testing"
	"time"

	"github.com/eurofurence/reg-payment-nexi-adapter/docs"
	"github.com/stretchr/testify/require"
)

func TestServerAddrWithAddressAndPort(t *testing.T) {
	docs.Description("ensure ServerAddr() returns the correct server address string, with address specified")
	configurationData = &Application{Logging: LoggingConfig{Severity: "DEBUG"}, Server: ServerConfig{
		Address: "localhost",
		Port:    1234,
	}}
	require.Equal(t, "localhost:1234", ServerAddr(), "unexpected server address string")
}

func TestServerAddrWithOnlyPort(t *testing.T) {
	docs.Description("ensure ServerAddr() returns the correct server address string, with no address specified")
	configurationData = &Application{Logging: LoggingConfig{Severity: "DEBUG"}, Server: ServerConfig{
		Port: 1234,
	}}
	require.Equal(t, ":1234", ServerAddr(), "unexpected server address string")
}

func TestServerTimeouts(t *testing.T) {
	docs.Description("ensure ServerRead/Write/IdleTimout() return the correct timeouts")
	configurationData = &Application{Logging: LoggingConfig{Severity: "DEBUG"}, Server: ServerConfig{
		Address:      "localhost",
		Port:         1234,
		ReadTimeout:  13,
		WriteTimeout: 17,
		IdleTimeout:  23,
	}}
	require.Equal(t, 13*time.Second, ServerReadTimeout())
	require.Equal(t, 17*time.Second, ServerWriteTimeout())
	require.Equal(t, 23*time.Second, ServerIdleTimeout())
}
