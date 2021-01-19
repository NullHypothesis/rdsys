package stub

import (
	"errors"
	"log"
	"sync"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery/mechanisms"
)

const (
	DistName = "stub"
)

// StubDistributor contains the context that the distributor needs.  This
// structure must implement the Distributor interface.
type StubDistributor struct {
	// ring contains the resources that we are going to distribute.
	ring *core.Hashring
	// ipc represents the IPC mechanism that we use to talk to the backend.
	ipc delivery.Mechanism
	// cfg represents our configuration file.
	cfg *internal.Config
	// shutdown is used to let housekeeping know when it's time to finish.
	shutdown chan bool
	// wg is used to figure out when our housekeeping method is finished.
	wg sync.WaitGroup
}

// housekeeping keeps track of periodic tasks.
func (d *StubDistributor) housekeeping(rStream chan *core.ResourceDiff) {

	defer d.wg.Done()
	defer close(rStream)
	defer d.ipc.StopStream()

	for {
		select {
		case diff := <-rStream:
			// We got a resource update from the backend.  Let's add it to our
			// hashring.
			d.ring.ApplyDiff(diff)
		case <-d.shutdown:
			// We are told to shut down.
			log.Printf("Shutting down housekeeping.")
			return
		}
	}
}

// RequestBridges takes as input a hashkey (it is the frontend's responsibility
// to derive the hashkey) and uses it to return a slice of resources.
func (d *StubDistributor) RequestBridges(key core.Hashkey) ([]core.Resource, error) {

	if d.ring.Len() == 0 {
		return nil, errors.New("no bridges available")
	}

	r, err := d.ring.Get(key)
	return []core.Resource{r}, err
}

// Init initialises the distributor.  Along with Shutdown, it's the only method
// that a distributor must implement to satisfy the Distributor interface.
func (d *StubDistributor) Init(cfg *internal.Config) {
	log.Printf("Initialising %s distributor.", DistName)

	d.cfg = cfg
	d.shutdown = make(chan bool)
	d.ring = core.NewHashring()

	// Request resources from the backend.  The backend will respond with an
	// initial batch of resources and then follow up with incremental updates
	// as resources change (e.g. some resources may disappear, others appear,
	// and others may change their state).  We will receive resources at the
	// rStream channel.
	log.Printf("Initialising resource stream.")
	d.ipc = mechanisms.NewHttpsIpc("http://" + cfg.Backend.WebApi.ApiAddress + cfg.Backend.ResourceStreamEndpoint)
	rStream := make(chan *core.ResourceDiff)
	req := core.ResourceRequest{
		RequestOrigin: DistName,
		ResourceTypes: d.cfg.Distributors.Stub.Resources,
		BearerToken:   d.cfg.Backend.ApiTokens[DistName],
		Receiver:      rStream,
	}
	d.ipc.StartStream(&req)

	d.wg.Add(1)
	go d.housekeeping(rStream)
}

// Shutdown shuts down the distributor.  This method is required to satisfy the
// Distributor interface.
func (d *StubDistributor) Shutdown() {
	log.Printf("Shutting down %s distributor.", DistName)

	// Signal to housekeeping that it's time to stop and wait until the
	// goroutine is done.
	close(d.shutdown)
	d.wg.Wait()
}
