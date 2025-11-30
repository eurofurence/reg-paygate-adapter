package attendeeservice

import (
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
)

var activeInstance AttendeeService

func Create() (err error) {
	if config.AttendeeServiceBaseUrl() != "" {
		activeInstance, err = newClient()
		return err
	} else {
		aulogging.Logger.NoCtx().Warn().Printf("service.attendee_service not configured. Using in-memory simulator for attendee service (not useful for production!)")
		activeInstance = newMock()
		return nil
	}
}

func CreateMock() Mock {
	instance := newMock()
	activeInstance = instance
	return instance
}

func Get() AttendeeService {
	return activeInstance
}
