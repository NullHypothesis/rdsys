package core

import (
	"testing"
	"time"
)

func TestAddCollection(t *testing.T) {
	d1 := NewDummy(1, 1)
	d2 := NewDummy(2, 2)
	d3 := NewDummy(3, 2)
	c := NewBackendResources([]string{d1.Name()}, &Stencil{})

	c.Add(d1)
	if c.Collection[d1.Name()].Len() != 1 {
		t.Errorf("expected length 1 but got %d", len(c.Collection))
	}
	c.Add(d2)
	if c.Collection[d1.Name()].Len() != 2 {
		t.Errorf("expected length 2 but got %d", len(c.Collection))
	}
	// d3 has the same unique ID as d2 but a different object ID.  Our
	// collection should update d2 but not create a new element.
	c.Add(d3)
	if c.Collection[d1.Name()].Len() != 2 {
		t.Errorf("expected length 2 but got %d", len(c.Collection))
	}

	elems, err := c.Collection[d3.Name()].GetMany(Hashkey(0), 2)
	if err != nil {
		t.Errorf(err.Error())
	}
	if elems[0] != d1 {
		t.Errorf("got unexpected element")
	}
	if elems[1] != d3 {
		t.Errorf("got unexpected element: %d", elems[1].Oid())
	}
}

func TestStringCollection(t *testing.T) {
	d := NewDummy(1, 1)
	c := NewBackendResources([]string{d.Name()}, &Stencil{})
	s := c.String()
	expected := "0 dummy"
	if s != expected {
		t.Errorf("expected %q but got %q", expected, s)
	}
}

func TestPruneCollection(t *testing.T) {
	d := NewDummy(1, 1)
	d.ExpiryTime = time.Minute * 10
	c := NewBackendResources([]string{d.Name()}, &Stencil{})
	c.Add(d)
	hLength := func() int { return c.Collection[d.Name()].Len() }

	// We should now have one element in the hashring.
	if hLength() != 1 {
		t.Fatalf("expectec hashring of length 1 but got %d", hLength())
	}

	// Expire the hashring node.
	i, err := c.Collection[d.Name()].getIndex(d.Uid())
	if err != nil {
		t.Errorf("failed to retrieve existing resource: %s", err)
	}
	node := c.Collection[d.Name()].Hashnodes[i]
	node.LastUpdate = time.Now().UTC().Add(-d.ExpiryTime - time.Minute)

	c.Prune()
	// Pruning should have left our hashring empty.
	if hLength() != 0 {
		t.Fatalf("expectec hashring of length 0 but got %d", hLength())
	}
}
