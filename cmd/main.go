package main

import (
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/app"
	"os"
)

func main() {
	os.Exit(app.New().Run())
}
