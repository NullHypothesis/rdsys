package salmon

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/internal"
	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/usecases/distributors"
)

var salmon *distributors.SalmonDistributor

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
				str += proxy.String()
			}
			fmt.Fprintf(w, "proxies:%s\n", str)
		}
	}
}

func AccountHandler(w http.ResponseWriter, r *http.Request) {
	id, err := salmon.Register()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "new user id: %d", id)
}

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

func Init(cfg *internal.Config) {

	salmon = distributors.NewSalmonDistributor()
	salmon.Init(cfg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		// if err := cache.WriteToDisk(cacheFile); err != nil {
		// 	log.Printf("Could not write cache because: %s", err)
		// }
		// TODO: wait for open scans to be done. maybe by doing proper web
		// server shutdown? it should wait for all connection to be terminated?
		os.Exit(1)
	}()

	// log.Printf("Starting service on port %s.", addr)
	// if certFilename != "" && keyFilename != "" {
	// 	log.Fatal(http.ListenAndServeTLS(addr, certFilename, keyFilename, router))
	// } else {
	addr := ":8000"
	mux := http.NewServeMux()

	mux.Handle("/proxies", http.HandlerFunc(ProxiesHandler))
	mux.Handle("/account", http.HandlerFunc(AccountHandler))
	mux.Handle("/invite", http.HandlerFunc(InviteHandler))
	mux.Handle("/redeem", http.HandlerFunc(RedeemHandler))

	log.Fatal(http.ListenAndServe(addr, mux))
	// }
}
