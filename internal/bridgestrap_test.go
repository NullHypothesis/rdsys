package internal

import (
	"testing"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

// DummyDelivery is a drop-in replacement for our HTTPS interface and
// facilitates testing.
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

func TestInProgress(t *testing.T) {

	bridgeLine := "dummy"
	p := NewResourceTestPool("")

	if p.alreadyInProgress(bridgeLine) == true {
		t.Fatal("bridge line isn't currently being tested")
	}

	p.inProgress[bridgeLine] = true

	if p.alreadyInProgress(bridgeLine) != true {
		t.Fatal("bridge line is currently being tested")
	}
}

func TestDispatch(t *testing.T) {

	d := core.NewDummy(0, 0)
	p := NewResourceTestPool("")
	p.ipc = &DummyDelivery{}
	// Set flush timeout to a nanosecond, so it triggers practically instantly.
	p.flushTimeout = time.Nanosecond
	defer p.Stop()

	p.pending <- d
	d.Test().State = core.StateUntested
	p.pending <- d
	time.Sleep(time.Millisecond)

	if d.Test().State == core.StateUntested {
		t.Fatal("resource should not be untested")
	}
}

func TestTestFunc(t *testing.T) {

	p := NewResourceTestPool("")
	p.ipc = &DummyDelivery{}
	defer p.Stop()

	f := p.GetTestFunc()
	dummies := [25]*core.Dummy{}
	for i := 0; i < len(dummies); i++ {
		k := core.Hashkey(i)
		dummies[i] = core.NewDummy(k, k)
		f(dummies[i])
	}

	// Were all states set correctly?
	for i := 0; i < len(dummies); i++ {
		if dummies[i].Test().State != core.StateFunctional {
			t.Fatal("resource state was set incorrectly", dummies[i].Test().State)
		}
	}
}
