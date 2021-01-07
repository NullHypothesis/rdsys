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
	FarInTheFuture = time.Hour * 24 * 365 * 100
	// MaxResources determines the maximum number of resources that we're
	// willing to buffer before sending a request to bridgestrap.
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
	Functional bool      `json:"functional"`
	LastTested time.Time `json:"last_tested"`
	Error      string    `json:"error,omitempty"`
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
	flushTimeout time.Duration
	shutdown     chan bool
	pending      chan core.Resource
	ipc          delivery.Mechanism
	inProgress   map[string]bool
}

// NewResourceTestPool returns a new resource test pool.
func NewResourceTestPool(apiEndpoint string) *ResourceTestPool {
	p := &ResourceTestPool{}
	p.flushTimeout = time.Minute
	p.shutdown = make(chan bool)
	p.pending = make(chan core.Resource)
	p.ipc = mechanisms.NewHttpsIpc(apiEndpoint)
	p.inProgress = make(map[string]bool)
	go p.dispatch()

	return p
}

// GetTestFunc returns a function that's executed when a new resource is added
// to rdsys's backend.  The function takes as input a resource and submits it
// to our testing pool.
func (p *ResourceTestPool) GetTestFunc() core.TestFunc {
	return func(r core.Resource) {
		p.pending <- r
	}
}

// Stop stops the test pool by signalling to the dispatcher that it's time to
// shut down.
func (p *ResourceTestPool) Stop() {
	close(p.shutdown)
}

// alreadyInProgress returns 'true' if the given bridge line is being tested
// right now.
func (p *ResourceTestPool) alreadyInProgress(bridgeLine string) bool {
	p.Lock()
	defer p.Unlock()

	if _, exists := p.inProgress[bridgeLine]; exists {
		return true
	}
	p.inProgress[bridgeLine] = true
	return false
}

// dispatch handles the following requests:
// 1) Incoming resources to be tested
// 2) A timer whose expiry signals that it's time to test bridges
// 3) A shutdown signal, indicating that the function should return
func (p *ResourceTestPool) dispatch() {
	defer log.Printf("Shutting down resource pool ticker.")
	log.Printf("Starting resource pool ticker.")

	ticker := time.NewTicker(FarInTheFuture)
	rMap := make(map[string]core.Resource)
	for {
		select {
		case <-ticker.C:
			log.Println("Test pool timer expired.  Testing resources.")
			go p.testResources(rMap)
			rMap = make(map[string]core.Resource)
		case r := <-p.pending:
			if p.alreadyInProgress(r.String()) {
				break
			}

			// We got a new resource to test.  Start timer if our pool was
			// empty.
			if len(rMap) == 0 {
				log.Println("Starting test pool timer.")
				ticker.Reset(p.flushTimeout)
			}
			rMap[r.String()] = r

			// Test resources if our pool is full.
			if len(rMap) == MaxResources {
				log.Println("Test pool reached capacity.  Resetting timer and testing resources.")
				ticker.Reset(FarInTheFuture)
				go p.testResources(rMap)
				rMap = make(map[string]core.Resource)
			}
		case <-p.shutdown:
			return
		}
	}
}

// testResources puts all resources that are currently in our pool into a
// bridgestrap requests and sends them to our bridgestrap instance for testing.
// The testing results are then added to each resource's state.
func (p *ResourceTestPool) testResources(rMap map[string]core.Resource) {
	defer func() {
		p.Lock()
		for bridgeLine, _ := range rMap {
			delete(p.inProgress, bridgeLine)
		}
		p.Unlock()
	}()

	if len(rMap) == 0 {
		return
	}

	req := BridgestrapRequest{}
	resp := BridgestrapResponse{}
	for bridgeLine, _ := range rMap {
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
		r, exists := rMap[bridgeLine]
		if !exists {
			log.Printf("Bug: %q not in our resource test pool.", bridgeLine)
			continue
		}

		rTest := r.Test()
		rTest.LastTested = bridgeTest.LastTested
		rTest.Error = bridgeTest.Error
		if bridgeTest.Functional {
			numFunctional++
			rTest.State = core.StateFunctional
		} else {
			numDysfunctional++
			rTest.State = core.StateDysfunctional
		}
	}
	log.Printf("Tested %d resources: %d functional and %d dysfunctional.",
		len(resp.Bridges), numFunctional, numDysfunctional)
}
