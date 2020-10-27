package https

import (
	"errors"
	"log"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery/mechanisms"
)

const (
	HttpsDistName        = "https"
	BridgeReloadInterval = time.Minute * 10
)

// HttpsDistributor contains all the context that the distributor needs to run.
type HttpsDistributor struct {
	ring     *core.Hashring
	ipc      delivery.Mechanism
	cfg      *internal.Config
	wg       sync.WaitGroup
	shutdown chan bool
}

// housekeeping keeps track of periodic tasks.
func (d *HttpsDistributor) housekeeping(rStream chan *core.ResourceDiff) {

	defer d.wg.Done()
	defer close(rStream)
	defer d.ipc.StopStream()

	for {
		select {
		case diff := <-rStream:
			d.ring.ApplyDiff(diff)
		case <-d.shutdown:
			log.Printf("Shutting down housekeeping.")
			return
		}
	}
}

// RequestBridges takes as input a hashkey (it is the frontend's responsibility
// to derive the hashkey) and uses it to return a slice of resources.
func (d *HttpsDistributor) RequestBridges(key core.Hashkey) ([]core.Resource, error) {

	if d.ring.Len() == 0 {
		return nil, errors.New("no bridges available")
	}

	r, err := d.ring.Get(key)
	return []core.Resource{r}, err
}

// Init initialises the given HTTPS distributor.
func (d *HttpsDistributor) Init(cfg *internal.Config) {
	log.Printf("Initialising %s distributor.", HttpsDistName)

	d.cfg = cfg
	d.shutdown = make(chan bool)
	d.ring = core.NewHashring()

	log.Printf("Initialising resource stream.")
	d.ipc = mechanisms.NewHttpsIpc("http://" + cfg.Backend.WebApi.ApiAddress + cfg.Backend.ResourceStreamEndpoint)
	rStream := make(chan *core.ResourceDiff)
	req := core.ResourceRequest{
		RequestOrigin: HttpsDistName,
		ResourceTypes: d.cfg.Distributors.Https.Resources,
		BearerToken:   d.cfg.Backend.ApiTokens[HttpsDistName],
		Receiver:      rStream,
	}
	d.ipc.StartStream(&req)

	d.wg.Add(1)
	go d.housekeeping(rStream)
}

// Shutdown shuts down the given HTTPS distributor.
func (d *HttpsDistributor) Shutdown() {
	log.Printf("Shutting down %s distributor.", HttpsDistName)

	// Signal to housekeeping that it's time to stop.
	close(d.shutdown)
	d.wg.Wait()
}
