package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/StephanHCB/go-autumn-logging-zerolog/loggermiddleware"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/config"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/repository/self"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/service/paymentlinksrv"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/controller/fallbackctl"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/controller/infoctl"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/controller/paylinkctl"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/controller/simulatorctl"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/controller/webhookctl"
	"github.com/eurofurence/reg-payment-nexi-adapter/internal/web/middleware"
	"github.com/go-chi/chi/v5"
)

func CreateRouter(ctx context.Context) (chi.Router, error) {
	aulogging.Logger.NoCtx().Debug().Print("Setting up router")
	server := chi.NewRouter()

	server.Use(middleware.AddRequestIdToContextAndResponse)
	server.Use(loggermiddleware.AddZerologLoggerToContext)
	server.Use(middleware.RequestLogger)
	server.Use(middleware.PanicRecoverer)
	server.Use(middleware.CorsHandling)
	server.Use(middleware.TokenValidator)

	// add your business logic services here
	paymentLinkService := paymentlinksrv.New()

	// add your controllers here
	paylinkctl.Create(server, paymentLinkService)
	webhookctl.Create(server, paymentLinkService)
	if config.NexiDownstreamBaseUrl() == "" {
		aulogging.Logger.NoCtx().Warn().Printf("service.nexi_downstream not configured. Enabling local paylink simulator at %s/simulator (not useful for production!)", config.ServicePublicURL())
		err := self.Create()
		if err != nil {
			return server, err
		}
		simulatorctl.Create(server, paymentLinkService)
	}
	infoctl.Create(server)
	fallbackctl.Create(server)
	return server, nil
}

func newServer(ctx context.Context, router chi.Router) *http.Server {
	aulogging.Logger.NoCtx().Debug().Print("setting up server")
	return &http.Server{
		Addr:         config.ServerAddr(),
		Handler:      router,
		ReadTimeout:  config.ServerReadTimeout(),
		WriteTimeout: config.ServerWriteTimeout(),
		IdleTimeout:  config.ServerIdleTimeout(),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
}

func runServerWithGracefulShutdown() error {
	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	handler, err := CreateRouter(ctx)
	if err != nil {
		cancel()
		return err
	}
	srv := newServer(ctx, handler)

	go func() {
		<-sig
		defer cancel()
		aulogging.Logger.NoCtx().Debug().Print("Stopping services now")

		tCtx, tcancel := context.WithTimeout(ctx, time.Second*5)
		defer tcancel()

		if err := srv.Shutdown(tCtx); err != nil {
			aulogging.Logger.NoCtx().Error().WithErr(err).Printf("Couldn't shutdown server gracefully: %s", err.Error())
			os.Exit(3)
		}
	}()

	aulogging.Logger.NoCtx().Info().Print("Running service on ", config.ServerAddr())
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		aulogging.Logger.NoCtx().Error().WithErr(err).Printf("Server closed unexpectedly: %s", err.Error())
		return err
	}

	return nil
}
