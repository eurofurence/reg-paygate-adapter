package fallbackctl

import (
	"net/http"

	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/util/ctlutil"
	"github.com/go-chi/chi/v5"
)

func Create(server chi.Router) {
	server.HandleFunc("/*", fallbackErrorHandler)
}

func fallbackErrorHandler(w http.ResponseWriter, r *http.Request) {
	ctlutil.ErrorHandler(r.Context(), w, r, "not.found", http.StatusNotFound, nil)
}
