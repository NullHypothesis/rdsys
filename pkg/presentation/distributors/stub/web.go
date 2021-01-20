package stub

import (
	"fmt"
	"net/http"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/common"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors/stub"
)

var dist *stub.StubDistributor

// RequestHandler handles requests for /.
func RequestHandler(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)

	// Call our distributor backend to get bridges.
	resources, err := dist.RequestBridges(0)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	} else {
		for _, res := range resources {
			fmt.Fprintf(w, fmt.Sprintln(res.String()))
		}
	}
}

// InitFrontend is the entry point to stub's Web frontend.  It spins up a Web
// server and then waits until it receives a SIGINT.  Note that we can
// implement all sorts of user-facing frontends here.  It doesn't have to be a
// Web server.  It could be an SMTP server, BitTorrent tracker, message board,
// etc.
func InitFrontend(cfg *internal.Config) {

	// Start our distributor backend, which takes care of the distribution
	// logic.  This file implements the user-facing distribution code.
	dist = &stub.StubDistributor{}
	handlers := map[string]http.HandlerFunc{
		"/": http.HandlerFunc(RequestHandler),
	}

	common.StartWebServer(
		&cfg.Distributors.Stub.WebApi,
		cfg,
		dist,
		handlers,
	)
}
