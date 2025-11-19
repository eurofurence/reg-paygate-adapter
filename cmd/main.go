package main

import (
	"os"

	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/app"
)

func main() {
	os.Exit(app.New().Run())
}
