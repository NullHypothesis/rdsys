package core

import (
	"testing"
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
