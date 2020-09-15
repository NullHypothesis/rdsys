package salmon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	rawId, ok := r.Form["id"]
	if !ok {
		http.Error(w, "no field 'id' given", http.StatusBadRequest)
		return
	} else if len(rawId) != 1 {
		http.Error(w, "need excactly one 'id' field", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(rawId[0])
	if err != nil {
		http.Error(w, "'id' field not a number", http.StatusBadRequest)
		return
	}
	proxies, err := salmon.GetProxies(id)
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
	id, err := salmon.Register()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "new user id: %d", id)
}

// InviteHandler handles requests for /invite.
func InviteHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	rawId, ok := r.Form["id"]
	if !ok {
		http.Error(w, "no field 'id' given", http.StatusBadRequest)
		return
	} else if len(rawId) != 1 {
		http.Error(w, "need excactly one 'id' field", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(rawId[0])
	if err != nil {
		http.Error(w, "'id' field not a number", http.StatusBadRequest)
		return
	}

	token, err := salmon.CreateInvite(id)
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

	id, err := salmon.RedeemInvite(token[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "new user id: %d", id)
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
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Web API shut down: %s", err)
	}
}
