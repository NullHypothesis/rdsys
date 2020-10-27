package https

import (
	"fmt"
	"hash/crc64"
	"log"
	"net/http"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/common"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors/https"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

var dist *https.HttpsDistributor

// mapRequestToHashkey maps the given HTTP request to a hash key.  It does so
// by taking the /16 of the client's IP address.  For example, if the client's
// address is 1.2.3.4, the function turns it into 1.2., computes its CRC64, and
// returns the resulting hash key.
func mapRequestToHashkey(r *http.Request) core.Hashkey {

	i := 0
	for numDots := 0; i < len(r.RemoteAddr) && numDots < 2; i++ {
		if r.RemoteAddr[i] == '.' {
			numDots++
		}
	}
	slash16 := r.RemoteAddr[:i]
	log.Printf("Using address prefix %q as hash key.", slash16)
	table := crc64.MakeTable(resources.Crc64Polynomial)

	return core.Hashkey(crc64.Checksum([]byte(slash16), table))
}

// RequestHandler handles requests for /.
func RequestHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	resources, err := dist.RequestBridges(mapRequestToHashkey(r))
	if err != nil {
		fmt.Fprintf(w, err.Error())
	} else {
		fmt.Fprintf(w, "Your %s bridge(s):<br>", resources[0].Type())
		for _, res := range resources {
			fmt.Fprintf(w, fmt.Sprintf("<tt>%s</tt><br>", res.String()))
		}
	}
}

// InitFrontend is the entry point to HTTPS's Web frontend.  It spins up the
// Web server and then waits until it receives a SIGINT.
func InitFrontend(cfg *internal.Config) {

	dist = &https.HttpsDistributor{}
	handlers := map[string]http.HandlerFunc{
		"/": http.HandlerFunc(RequestHandler),
	}

	common.StartWebServer(
		&cfg.Distributors.Https.WebApi,
		cfg,
		dist,
		handlers,
	)
}
