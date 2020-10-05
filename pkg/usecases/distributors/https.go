package distributors

import (
	"context"
	"fmt"
	"hash/crc64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery/mechanisms"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

const (
	HttpsDistName        = "https"
	BridgeReloadInterval = time.Minute * 10
)

type HttpsDistributor struct {
	srv  http.Server
	ring *core.Hashring
	// ipc  core.IpcMechanism
	ipc delivery.Mechanism
	cfg *internal.Config
}

// requestResources periodically requests updated bridges from our backend.
func (d *HttpsDistributor) periodicTasks(wg *sync.WaitGroup) {

	defer wg.Done()
	ticker := time.NewTicker(BridgeReloadInterval)
	defer ticker.Stop()
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	log.Printf("Initialising resource stream.")
	stream := make(chan *core.ResourceDiff)
	defer close(stream)
	req := core.ResourceRequest{
		RequestOrigin: HttpsDistName,
		ResourceTypes: d.cfg.Distributors.Https.Resources,
		BearerToken:   d.cfg.Backend.ApiTokens["https"],
		Receiver:      stream,
	}

	d.ipc.(*mechanisms.HttpsIpcContext).StartStream(&req)
	for {
		select {
		case diff := <-stream:
			log.Printf("Got diff with %d new, %d changed, and %d gone resources.",
				len(diff.New), len(diff.Changed), len(diff.Gone))
			d.ring.ApplyDiff(diff)
			log.Printf("Done applying update; hashring length is %d.", d.ring.Len())

		case <-ticker.C:
			log.Printf("Ticker is ticking.")
			// TODO: Anything that we need to do periodically?

		case <-sigint:
			log.Println("Caught SIGINT.")
			//client.Stop()
			d.ipc.(*mechanisms.HttpsIpcContext).StopStream()
			if err := d.Shutdown(); err != nil {
				log.Printf("Error while shutting down: %s", err)
			}
			return
		}
	}
}

// mapRequestToHashkey maps the given HTTP request to a hash key.  It does so
// by taking the /16 of the client's IP address.  For example, if the client's
// address is 1.2.3.4, the function turns it into 1.2., computes its CRC64, and
// returns the resulting hash key.
func (d *HttpsDistributor) mapRequestToHashkey(r *http.Request) core.Hashkey {

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

func (d *HttpsDistributor) handleBridgeRequest(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if d.ring.Len() == 0 {
		fmt.Fprintf(w, "No bridges available.")
	} else {
		r, err := d.ring.Get(d.mapRequestToHashkey(r))
		if err != nil {
			fmt.Fprintf(w, "Error while fetching bridge: %s", err)
		} else {
			fmt.Fprintf(w, "%s bridge: <tt>%s</tt>", r.Type(), r.String())
		}
	}
}

// Init starts our Web server.
func (d *HttpsDistributor) Init(cfg *internal.Config) error {
	log.Printf("Initialising %s distributor.", HttpsDistName)

	var wg sync.WaitGroup
	d.cfg = cfg
	d.ring = core.NewHashring()
	d.ipc = mechanisms.NewHttpsIpc("http://" + cfg.Backend.ApiAddress + cfg.Backend.ResourceStreamEndpoint)

	wg.Add(1)
	go d.periodicTasks(&wg)

	http.HandleFunc("/", d.handleBridgeRequest)
	d.srv.Addr = cfg.Distributors.Https.ApiAddress
	log.Printf("Starting Web server at %s.", d.srv.Addr)
	if err := d.srv.ListenAndServe(); err != nil {
		log.Printf("Web server terminated: %s", err)
	}
	wg.Wait()

	return nil
}

// Shutdown shuts down our Web server.
func (d *HttpsDistributor) Shutdown() error {
	log.Printf("Shutting down %s distributor.", HttpsDistName)

	// Give our Web server five seconds to shut down.
	t := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), t)
	defer cancel()
	return d.srv.Shutdown(ctx)
}
