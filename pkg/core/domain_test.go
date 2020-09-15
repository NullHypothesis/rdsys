package core

import (
	"testing"
)

func TestBlockedIn(t *testing.T) {
	r := ResourceBase{}
	l := &Location{"DE", 1234}

	if r.IsBlockedIn(l) {
		t.Error("Falsely labeled resource as blocked.")
	}

	l = &Location{"AT", 1234}
	if r.IsBlockedIn(l) {
		t.Error("Falsely labeled resource as blocked.")
	}
}

func TestQueue(t *testing.T) {

	d0 := NewDummy(0, 0)
	d1 := NewDummy(1, 1)
	d2 := NewDummy(5, 5)
	d3 := NewDummy(6, 5) // Same UID but different OID as previous resource.
	q := ResourceQueue{}
	if _, err := q.Dequeue(); err == nil {
		t.Errorf("failed to return error for empty queue")
	}
	if err := q.Delete(d1); err == nil {
		t.Errorf("failed to return error for empty queue")
	}
	if err := q.Update(d1); err == nil {
		t.Errorf("failed to return error for empty queue")
	}

	q.Enqueue(d0)
	q.Enqueue(d1)
	if err := q.Enqueue(d1); err == nil {
		t.Errorf("failed to return error for duplicate resource")
	}
	q.Enqueue(d2)
	if len(q) != 3 {
		t.Errorf("expected queue length of 3 but got %d", len(q))
	}

	// Get d0 from the queue.
	r, err := q.Dequeue()
	if err != nil {
		t.Errorf("failed to dequeue resource: %s", err)
	}
	if r.Oid() != d0.Oid() {
		t.Errorf("got wrong resource from queue")
	}
	if len(q) != 2 {
		t.Errorf("expected queue length of 2 but got %d", len(q))
	}

	// Delete d1 from the queue.
	if err = q.Delete(d1); err != nil {
		t.Errorf("failed to delete existing resource")
	}
	if len(q) != 1 {
		t.Errorf("expected queue length of 1 but got %d", len(q))
	}

	// Only d2 remains in our queue.  Let's update it, so it resembles d3.
	if err = q.Update(d3); err != nil {
		t.Errorf("failed to update resource")
	}
	r, err = q.Dequeue()
	if err != nil {
		t.Errorf("failed to return existing resource")
	}
	if r.Uid() != d3.Uid() || r.Oid() != d3.Oid() {
		t.Errorf("returned resource was not updated correctly")
	}

	// Try deleting the only resource in a queue, to cover an edge case.
	q.Enqueue(d1)
	q.Delete(d1)
	if len(q) != 0 {
		t.Errorf("expected queue length of 0 but got %d", len(q))
	}
}
