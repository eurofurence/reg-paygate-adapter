package main

import (
	"os"

	"github.com/eurofurence/reg-paygate-adapter/internal/web/app"
)

func main() {
	os.Exit(app.New().Run())
}
