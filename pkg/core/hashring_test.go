package core

import (
	"testing"
)

func TestLen(t *testing.T) {
	d1 := &Dummy{1, 1}
	d2 := &Dummy{5, 5}
	h := NewHashring()

	if h.Len() != 0 {
		t.Errorf("expected length 0 but got %d", h.Len())
	}

	if err := h.Add(d1); err != nil {
		t.Error(err)
	}
	if h.Len() != 1 {
		t.Errorf("expected length 1 but got %d", h.Len())
	}

	if err := h.Add(d2); err != nil {
		t.Error(err)
	}
	if h.Len() != 2 {
		t.Errorf("expected length 2 but got %d", h.Len())
	}
}

func TestAdd(t *testing.T) {
	d1 := &Dummy{1, 1}
	d2 := &Dummy{2, 2}
	h := NewHashring()

	if err := h.Add(d1); err != nil {
		t.Error(err)
	}
	if err := h.Add(d1); err == nil {
		t.Error("adding duplicate element should result in error")
	}
	if err := h.Add(d2); err != nil {
		t.Error(err)
	}
	if err := h.Add(d2); err == nil {
		t.Error("adding duplicate element should result in error")
	}
}

func TestGet(t *testing.T) {
	d1 := &Dummy{5, 5}
	d2 := &Dummy{10, 10}
	h := NewHashring()

	if _, err := h.Get(d1.Uid()); err == nil {
		t.Error("retrieving element from empty hashring should result in error")
	}

	h.Add(d1)
	h.Add(d2)
	i, err := h.Get(0)
	if err != nil {
		t.Error(err)
	}
	if i.(*Dummy).UniqueId != 5 {
		t.Error("got wrong element")
	}

	i, err = h.Get(5)
	if err != nil {
		t.Error(err)
	}
	if i.(*Dummy).UniqueId != 5 {
		t.Error("got wrong element")
	}

	i, err = h.Get(9)
	if err != nil {
		t.Error(err)
	}
	if i.(*Dummy).UniqueId != 10 {
		t.Error("got wrong element")
	}

	i, err = h.Get(11)
	if err != nil {
		t.Error(err)
	}
	if i.(*Dummy).UniqueId != 5 {
		t.Error("got wrong element")
	}
}

func TestGetMany(t *testing.T) {
	d1 := &Dummy{5, 5}
	d2 := &Dummy{10, 10}
	d3 := &Dummy{15, 15}
	h := NewHashring()

	if _, err := h.GetMany(0, 0); err == nil {
		t.Error("requesting elements from empty hashring should result in error")
	}

	h.Add(d1)
	h.Add(d2)
	h.Add(d3)
	if _, err := h.GetMany(0, 4); err == nil {
		t.Error("requesting more elements than present should result in error")
	}

	numElems := 3
	elems, err := h.GetMany(11, numElems)
	if err != nil {
		t.Error(err)
	}
	if len(elems) != numElems {
		t.Errorf("got %d elements but expected 3", numElems)
	}
	if elems[0].(*Dummy).UniqueId != 15 {
		t.Error("got wrong element")
	}
	if elems[1].(*Dummy).UniqueId != 5 {
		t.Error("got wrong element")
	}
	if elems[2].(*Dummy).UniqueId != 10 {
		t.Error("got wrong element")
	}
}

func TestRemove(t *testing.T) {
	d1 := &Dummy{1, 1}
	d2 := &Dummy{2, 2}
	d3 := &Dummy{3, 3}
	h := NewHashring()

	// Add a single element and remove it.
	h.Add(d1)
	if err := h.Remove(d1); err != nil {
		t.Error(err)
	}
	if h.Len() != 0 {
		t.Errorf("expected length 0 but got %d", h.Len())
	}

	// Add two elements and remove one.
	h.Add(d1)
	h.Add(d2)
	if err := h.Remove(d1); err != nil {
		t.Error(err)
	}
	if h.Len() != 1 {
		t.Errorf("expected length 1 but got %d", h.Len())
	}

	// Add two elements and remove the other one.
	h.Add(d1)
	if err := h.Remove(d2); err != nil {
		t.Error(err)
	}
	if h.Len() != 1 {
		t.Errorf("expected length 1 but got %d", h.Len())
	}

	// Add three elements and remove the middle one.
	h.Add(d2)
	h.Add(d3)
	if err := h.Remove(d2); err != nil {
		t.Error(err)
	}
	if h.Len() != 2 {
		t.Errorf("expected length 2 but got %d", h.Len())
	}

	// Try removing an element that was already removed.
	if err := h.Remove(d2); err == nil {
		t.Error("removing a non-existing element should result in error")
	}

	// Make sure that d1 and d3 remain in the hashring.
	if _, err := h.Get(d1.Uid()); err != nil {
		t.Error(err)
	}
	if _, err := h.Get(d3.Uid()); err != nil {
		t.Error(err)
	}
}

func TestDiff(t *testing.T) {

	h1 := &Hashring{}
	h2 := &Hashring{}
	d1 := &Dummy{1, 1}
	d2 := &Dummy{2, 2}
	d3 := &Dummy{3, 2}
	d4 := &Dummy{4, 3}

	h1.Add(d1)
	h1.Add(d2)
	h2.Add(d3)
	h2.Add(d4)

	// We should be dealing with one new, one changed, and one gone resource.
	diff := h1.Diff(h2)
	if len(diff.New) != 1 {
		t.Error("incorrect number of new resources")
	}
	if diff.New["dummy"][0] != d1 {
		t.Error("failed to find new resources")
	}

	if len(diff.Changed) != 1 {
		t.Error("incorrect number of changed resources")
	}
	if diff.Changed["dummy"][0] != d2 {
		t.Errorf("failed to find changed resources")
	}

	if len(diff.Gone) != 1 {
		t.Error("incorrect number of gone resources")
	}
	if diff.Gone["dummy"][0] != d4 {
		t.Errorf("failed to find gone resources")
	}
}
