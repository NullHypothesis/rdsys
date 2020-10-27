package common

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors"
)

// StartWebServer helps distributor frontends start a Web server and configure
// handlers.  This function does not return until it receives a SIGINT or
// SIGTERM.  When that happens, the function calls the distributor's Shutdown
// method and shuts down the Web server.
func StartWebServer(apiCfg *internal.WebApiConfig, distCfg *internal.Config,
	dist distributors.Distributor, handlers map[string]http.HandlerFunc) {

	var srv http.Server
	dist.Init(distCfg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Printf("Caught SIGINT.")
		dist.Shutdown()

		log.Printf("Shutting down Web API.")
		// Give our Web server five seconds to shut down.
		t := time.Now().Add(5 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), t)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			log.Printf("Error shutting down Web API: %s", err)
		}
	}()

	mux := http.NewServeMux()
	for endpoint, handlerFunc := range handlers {
		mux.Handle(endpoint, handlerFunc)
	}
	srv.Handler = mux

	// srv.Addr = cfg.Distributors.Salmon.ApiAddress
	srv.Addr = apiCfg.ApiAddress
	log.Printf("Starting Web server at %s.", srv.Addr)

	var err error
	if apiCfg.KeyFile != "" && apiCfg.CertFile != "" {
		err = srv.ListenAndServeTLS(apiCfg.CertFile,
			apiCfg.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Printf("Web API shut down: %s", err)
	}
}
