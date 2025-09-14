package nexi

import (
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
)

var activeInstance NexiDownstream

func Create() (err error) {
	if config.NexiDownstreamBaseUrl() != "" {
		activeInstance, err = newClient()
		return err
	} else {
		aulogging.Logger.NoCtx().Warn().Print("service.nexi_downstream not configured. Using in-memory simulator for nexi downstream (not useful for production!)")
		activeInstance = newMock()
		return nil
	}
}

func CreateMock() Mock {
	instance := newMock()
	activeInstance = instance
	return instance
}

func Get() NexiDownstream {
	return activeInstance
}
