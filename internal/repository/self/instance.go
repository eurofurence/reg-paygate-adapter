package self

import (
	"errors"
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
)

var activeInstance Self

func Create() (err error) {
	if config.ServicePublicURL() != "" {
		aulogging.Logger.NoCtx().Warn().Printf("created local webhook self caller (not useful for production!)")
		activeInstance, err = newClient()
		return err
	} else {
		return errors.New("cannot create instance of self downstream - functionality is part of the simulator - this is a bug")
	}
}

func Get() Self {
	return activeInstance
}
