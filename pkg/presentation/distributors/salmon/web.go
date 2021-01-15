package salmon

import (
	"fmt"
	"log"
	"net/http"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/persistence/file"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/common"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors/salmon"
)

var dist *salmon.SalmonDistributor

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
	proxies, err := dist.GetProxies(secretId[0], rType[0])
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
	secretId, err := dist.Register()
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
	token, err := dist.CreateInvite(secretId[0])
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

	secretId, err := dist.RedeemInvite(token[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "new user secret-id: %s", secretId)
}

// InitFrontend is the entry point to Salmon's Web frontend.  It spins up the
// Web server and then waits until it receives a SIGINT.
func InitFrontend(cfg *internal.Config) {

	dist = salmon.NewSalmonDistributor()

	pMech := file.New(salmon.DistName, cfg.Distributors.Salmon.WorkingDir)
	if err := pMech.Load(dist); err != nil {
		// It's best to fail here, and encourage the operator to fix whatever
		// went wrong with our persistence mechanism.  If we continue despite
		// the error, we may end up overwriting important data.
		log.Fatalf("Failed to load persistent data: %s", err)
	}
	log.Printf("Distributor state: %s", dist)
	defer func() {
		if err := pMech.Save(dist); err != nil {
			log.Printf("Failed to save state: %s", err)
		}
	}()
	handlers := map[string]http.HandlerFunc{
		"/proxies": http.HandlerFunc(ProxiesHandler),
		"/account": http.HandlerFunc(AccountHandler),
		"/invite":  http.HandlerFunc(InviteHandler),
		"/redeem":  http.HandlerFunc(RedeemHandler),
	}

	common.StartWebServer(
		&cfg.Distributors.Salmon.WebApi,
		cfg,
		dist,
		handlers,
	)
}
