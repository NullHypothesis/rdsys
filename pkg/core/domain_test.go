package core

import (
	"testing"
)

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
	if _, err := q.Search(0); err == nil {
		t.Errorf("failed to return error when searching empty queue")
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

	// At this point, only d2 remains in the queue.
	if _, err = q.Search(0); err == nil {
		t.Errorf("failed to return error for non-existing unique ID")
	}
	if _, err = q.Search(5); err != nil {
		t.Errorf("returned error for existing unique ID")
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

func TestLocationString(t *testing.T) {

	l1 := &Location{CountryCode: "FI", ASN: 9123}
	l2 := &Location{CountryCode: "AT"}

	if l1.String() != "FI (9123)" {
		t.Errorf("got incorrect string representation")
	}
	if l2.String() != "AT" {
		t.Errorf("got incorrect string representation")
	}
}

func TestLocationSetString(t *testing.T) {

	s := LocationSet{"BY (1234)": true, "BE (4321)": true}
	// Maps are orderless, so we can end up with either one of the following
	// two representations.
	p1 := "BY (1234), BE (4321)"
	p2 := "BE (4321), BY (1234)"

	str := s.String()
	if str != p1 && str != p2 {
		t.Errorf("incorrect representation of location set")
	}
}

func TestHasLocationNotIn(t *testing.T) {

	s1 := LocationSet{"BY (1234)": true, "BE (4321)": true}

	if s1.HasLocationsNotIn(s1) {
		t.Errorf("failed to determine set relationship")
	}

	if s1.HasLocationsNotIn(LocationSet{"BY (1234)": true, "BE (4321)": true, "CA (1111)": true}) {
		t.Errorf("failed to determine set relationship")
	}

	if !s1.HasLocationsNotIn(LocationSet{"FR (2222)": true}) {
		t.Errorf("failed to determine set relationship")
	}

	if !s1.HasLocationsNotIn(LocationSet{}) {
		t.Errorf("failed to determine set relationship")
	}
}

func TestResourceBase(t *testing.T) {

	b := NewResourceBase()

	if b.Test().State != StateUntested {
		t.Errorf("resource base has wrong default state")
	}

	b.Test().State = StateFunctional
	if b.Test().State != StateFunctional {
		t.Errorf("failed to update resource base state")
	}

	ls := make(LocationSet)
	l1 := &Location{CountryCode: "DE", ASN: 1122}
	ls[l1.String()] = true
	b.SetBlockedIn(ls)
	l := b.BlockedIn()
	if len(l) != 1 {
		t.Errorf("location set has incorrect length")
	}
	if _, exists := l["DE (1122)"]; !exists {
		t.Errorf("failed to retrieve blocked location set from resource base")
	}

	rType := "foobar"
	b.SetType(rType)
	if b.Type() != rType {
		t.Errorf("failed to retrieve the resource's type")
	}
}

func TestHasResourceType(t *testing.T) {

	rr := ResourceRequest{ResourceTypes: []string{"obfs3", "obfs4"}}
	if rr.HasResourceType("foo") {
		t.Errorf("failed to return 'false' for non-existing type")
	}
	if !rr.HasResourceType("obfs4") {
		t.Errorf("failed to return 'true' for existing type")
	}
}

func TestResourceMapString(t *testing.T) {

	m := make(ResourceMap)
	if m.String() != "empty" {
		t.Errorf("got incorrect string representation of resource map")
	}

	m["dummy"] = []Resource{NewDummy(0, 0)}
	if m.String() != "dummy: 1" {
		t.Errorf("got incorrect string representation of resource map")
	}
}

func TestApplyDiff(t *testing.T) {

	m := make(ResourceMap)

	diff := NewResourceDiff()
	diff.New["dummy"] = []Resource{NewDummy(0, 0)}
	m.ApplyDiff(diff)
	if len(m["dummy"]) != 1 || m["dummy"][0].Uid() != 0 {
		t.Errorf("failed to add resource from diff")
	}

	diff = NewResourceDiff()
	diff.Changed["dummy"] = []Resource{NewDummy(1, 0)}
	m.ApplyDiff(diff)
	if len(m["dummy"]) != 1 || m["dummy"][0].Oid() != 1 {
		t.Errorf("failed to update resource from diff")
	}

	diff = NewResourceDiff()
	diff.Gone["dummy"] = []Resource{NewDummy(1, 0)}
	m.ApplyDiff(diff)
	if len(m["dummy"]) != 0 {
		t.Errorf("failed to remove resource from diff")
	}
}

func TestResourceTests(t *testing.T) {

	b := NewResourceBase()
	if b.Test().State != StateUntested {
		t.Fatal("unexpected resource state")
	}

	test := &ResourceTest{
		State: StateDysfunctional,
		Error: "something went wrong",
	}
	b.SetTest(test)
	if b.Test().State != StateDysfunctional {
		t.Fatal("unexpected resource state")
	}
}
