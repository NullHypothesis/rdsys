package internal

import (
	"log"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery/mechanisms"
)

const (
	// FarInTheFuture determines a time span that's far enough in the future to
	// practically count as infinity.
	FarInTheFuture = time.Hour * 24 * 365
	// FlushTimeout determines the time span after the first resource was added
	// to the pool when we issue a bridgestrap request.
	FlushTimeout = time.Minute
	// MaxResources determines the number of resources at which we issue a
	// bridgestrap request.
	MaxResources = 25
)

// BridgestrapRequest represents a request for bridgestrap.  Here's what its
// API look like: https://gitlab.torproject.org/phw/bridgestrap#input
type BridgestrapRequest struct {
	BridgeLines []string `json:"bridge_lines"`
}

// BridgeTest represents the status of a single bridge in bridgestrap's
// response.
type BridgeTest struct {
	Functional bool   `json:"functional"`
	Error      string `json:"error,omitempty"`
}

// BridgestrapResponse represents bridgestrap's response.
type BridgestrapResponse struct {
	Bridges map[string]*BridgeTest `json:"bridge_results"`
	Time    float64                `json:"time"`
	Error   string                 `json:"error,omitempty"`
}

// ResourceTestPool implements a pool to which we add resources until it's time
// to send them to bridgestrap for testing.
type ResourceTestPool struct {
	sync.Mutex
	rMap     map[string]core.Resource
	ticker   *time.Ticker
	shutdown chan bool
	ipc      delivery.Mechanism
}

// NewResourceTestPool returns a new resource test pool.
func NewResourceTestPool(apiEndpoint string) *ResourceTestPool {
	p := &ResourceTestPool{}
	p.shutdown = make(chan bool)
	p.ipc = mechanisms.NewHttpsIpc(apiEndpoint)
	p.rMap = make(map[string]core.Resource)
	p.ticker = time.NewTicker(FarInTheFuture)
	go p.waitForTicker()

	return p
}

// AddFunc returns a function that's executed when a new resource is added to
// rdsys's backend.  The function takes as input a resource and adds it to our
// testing pool.
func (p *ResourceTestPool) AddFunc() core.OnAddFunc {
	return func(r core.Resource) {
		p.Lock()
		defer p.Unlock()

		// Start timer if our pool was empty.
		if len(p.rMap) == 0 {
			p.ticker.Reset(FlushTimeout)
		}
		p.rMap[r.String()] = r

		// Test resources if our pool is full.
		if len(p.rMap) == MaxResources {
			p.ticker.Reset(FarInTheFuture)
			p.testResources()
		}
	}
}

// Stop stops the test pool.
func (p *ResourceTestPool) Stop() {
	close(p.shutdown)
}

// waitForTicker tests the resources that are currently in the pool when our
// ticker fires.
func (p *ResourceTestPool) waitForTicker() {
	defer log.Printf("Shutting down resource pool ticker.")
	log.Printf("Starting resource pool ticker.")
	for {
		select {
		case <-p.ticker.C:
			p.Lock()
			p.testResources()
			p.Unlock()
		case <-p.shutdown:
			return
		}
	}
}

// testResources puts all resources that are currently in our pool into a
// bridgestrap requests and sends them to our bridgestrap instance for testing.
// The testing results are then added to each resource's state.
func (p *ResourceTestPool) testResources() {
	if len(p.rMap) == 0 {
		return
	}
	defer func() {
		// We're done testing; time to reset our pool and ticker.
		p.rMap = make(map[string]core.Resource)
		p.ticker.Reset(FarInTheFuture)
	}()

	req := BridgestrapRequest{}
	resp := BridgestrapResponse{}
	for bridgeLine, _ := range p.rMap {
		req.BridgeLines = append(req.BridgeLines, bridgeLine)
	}

	if err := p.ipc.MakeJsonRequest(req, &resp); err != nil {
		log.Printf("Bridgestrap request failed: %s", err)
		return
	}
	if resp.Error != "" {
		log.Printf("Bridgestrap test failed: %s", resp.Error)
		return
	}

	numFunctional, numDysfunctional := 0, 0
	for bridgeLine, bridgeTest := range resp.Bridges {
		r, exists := p.rMap[bridgeLine]
		if !exists {
			log.Printf("Bug: %q not in our resource pool.", bridgeLine)
			continue
		}

		if bridgeTest.Functional {
			numFunctional++
			r.SetState(core.StateFunctional)
		} else {
			numDysfunctional++
			r.SetState(core.StateNotFunctional)
		}
	}
	log.Printf("Tested %d resources: %d functional and %d dysfunctional.",
		len(resp.Bridges), numFunctional, numDysfunctional)
}
