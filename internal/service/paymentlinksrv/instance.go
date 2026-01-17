package paymentlinksrv

import (
	"regexp"
	"time"
)

var NowFunc = time.Now

type Impl struct {
	Now               func() time.Time
	simulationMatcher *regexp.Regexp
}

func New() PaymentLinkService {
	return &Impl{
		Now:               NowFunc,
		simulationMatcher: regexp.MustCompile(`^[0-9]{4}$`),
	}
}
