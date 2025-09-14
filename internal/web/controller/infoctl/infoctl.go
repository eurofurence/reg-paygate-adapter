package infoctl

import (
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/api/v1/nexiapi"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctlutil"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func Create(server chi.Router) {
	server.Get("/", healthHandler)
	server.Get("/info/health", healthHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	dto := nexiapi.HealthReportDto{Status: "OK"}
	ctlutil.WriteJson(r.Context(), w, dto)
}
