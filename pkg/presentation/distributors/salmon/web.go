package salmon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors"
)

var salmon *distributors.SalmonDistributor

// ProxiesHandler handles requests for /proxies.
func ProxiesHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	secretId, ok := r.Form["secret-id"]
	if !ok {
		http.Error(w, "no field 'secret-id' given", http.StatusBadRequest)
		return
	} else if len(secretId) != 1 {
		http.Error(w, "need excactly one 'secret-id' field", http.StatusBadRequest)
		return
	}
	rType, ok := r.Form["type"]
	if !ok {
		http.Error(w, "no field 'type' given", http.StatusBadRequest)
		return
	}
	proxies, err := salmon.GetProxies(secretId[0], rType[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		if len(proxies) == 0 {
			fmt.Fprintf(w, "currently no proxies available")
		} else {
			var str string
			for _, proxy := range proxies {
				if proxy != nil {
					str += proxy.String()
				}
			}
			fmt.Fprintf(w, "proxies:%s\n", str)
		}
	}
}

// AccountHandler handles requests for /account.
func AccountHandler(w http.ResponseWriter, r *http.Request) {
	secretId, err := salmon.Register()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "new user secret-id: %s", secretId)
}

// InviteHandler handles requests for /invite.
func InviteHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	secretId, ok := r.Form["secret-id"]
	if !ok {
		http.Error(w, "no field 'secret-id' given", http.StatusBadRequest)
		return
	} else if len(secretId) != 1 {
		http.Error(w, "need excactly one 'secret-id' field", http.StatusBadRequest)
		return
	}
	token, err := salmon.CreateInvite(secretId[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "give the following token to your friend:\n%s", token)
}

// RedeemHandler handles requests for /redeem.
func RedeemHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	token, ok := r.Form["token"]
	if !ok {
		http.Error(w, "no field 'token' given", http.StatusBadRequest)
		return
	} else if len(token) != 1 {
		http.Error(w, "need excactly one 'token' field", http.StatusBadRequest)
		return
	}

	secretId, err := salmon.RedeemInvite(token[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "new user secret-id: %s", secretId)
}

// Init is the entry point to Salmon's Web frontend.  It spins up the Web
// server and then waits until it receives a SIGINT.
func Init(cfg *internal.Config) {

	var srv http.Server
	salmon = distributors.NewSalmonDistributor()
	salmon.Init(cfg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Printf("Caught SIGINT.")
		salmon.Shutdown()

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
	mux.Handle("/proxies", http.HandlerFunc(ProxiesHandler))
	mux.Handle("/account", http.HandlerFunc(AccountHandler))
	mux.Handle("/invite", http.HandlerFunc(InviteHandler))
	mux.Handle("/redeem", http.HandlerFunc(RedeemHandler))
	srv.Handler = mux

	srv.Addr = cfg.Distributors.Salmon.ApiAddress
	log.Printf("Starting Web server at %s.", srv.Addr)

	var err error
	if cfg.Distributors.Salmon.KeyFile != "" && cfg.Distributors.Salmon.CertFile != "" {
		err = srv.ListenAndServeTLS(cfg.Distributors.Salmon.CertFile,
			cfg.Distributors.Salmon.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Printf("Web API shut down: %s", err)
	}
}
