package internal

import (
	"testing"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

type DummyDelivery struct{}

func (d *DummyDelivery) StartStream(*core.ResourceRequest) {}
func (d *DummyDelivery) StopStream()                       {}
func (d *DummyDelivery) MakeJsonRequest(req interface{}, resp interface{}) error {
	resp.(*BridgestrapResponse).Bridges = make(map[string]*BridgeTest)
	for _, bridgeLine := range req.(BridgestrapRequest).BridgeLines {
		resp.(*BridgestrapResponse).Bridges[bridgeLine] = &BridgeTest{Functional: true}
	}
	return nil
}

func TestTestFunc(t *testing.T) {

	p := NewResourceTestPool("")
	// Replace our HTTPS IPC with our dummy to facilitate testing.
	p.ipc = &DummyDelivery{}
	f := p.GetTestFunc()
	dummies := [25]*core.Dummy{}
	for i := 0; i < len(dummies); i++ {
		k := core.Hashkey(i)
		dummies[i] = core.NewDummy(k, k)
	}

	f(dummies[0])
	if len(p.rMap) != 1 {
		t.Fatal("unexpected size of resource pool")
	}

	// Adding the resource again shouldn't affect the pool size.
	f(dummies[0])
	if len(p.rMap) != 1 {
		t.Fatal("unexpected size of resource pool")
	}

	for i := 0; i < len(dummies); i++ {
		f(dummies[i])
	}
	// Our resource pool should now be flushed.
	if len(p.rMap) != 0 {
		t.Fatal("unexpected size of resource pool: ", p.rMap)
	}

	f(dummies[0])
	// Trigger our timer.
	p.ticker.Reset(time.Nanosecond)
	time.Sleep(time.Millisecond * 100)
	if len(p.rMap) != 0 {
		t.Fatal("unexpected size of resource pool", len(p.rMap))
	}

	// Were all states set correctly?
	for i := 0; i < len(dummies); i++ {
		if dummies[i].Test().State != core.StateFunctional {
			t.Fatal("resource state was set incorrectly", dummies[i].Test().State)
		}
	}

	p.Stop()
}
