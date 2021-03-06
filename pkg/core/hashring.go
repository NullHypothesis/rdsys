package core

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// ResourceDiff represents a diff that contains new, changed, and gone
// resources.  A resource diff can be applied onto data structures that
// implement a collection of resources, e.g. a Hashring.
type ResourceDiff struct {
	New     ResourceMap `json:"new"`
	Changed ResourceMap `json:"changed"`
	Gone    ResourceMap `json:"gone"`
}

// Hashkey represents an index in a hashring.
type Hashkey uint64

// Hashnodes represents a node in a hashring.
type Hashnode struct {
	Hashkey    Hashkey
	Elem       Resource
	LastUpdate time.Time
}

// Hashring represents a hashring consisting of resources.
type Hashring struct {
	Hashnodes []*Hashnode
	TestFunc  TestFunc
	sync.RWMutex
}

// FilterFunc takes as input a resource and returns true or false, depending on
// its filtering criteria.
type FilterFunc func(r Resource) bool

// TestFunc takes as input a resource and tests it.
type TestFunc func(r Resource)

// NewResourceDiff returns a new ResourceDiff.
func NewResourceDiff() *ResourceDiff {
	return &ResourceDiff{
		New:     make(ResourceMap),
		Changed: make(ResourceMap),
		Gone:    make(ResourceMap),
	}
}

// NewHashnode returns a new hash node and sets its LastUpdate field to the
// current UTC time.
func NewHashnode(k Hashkey, r Resource) *Hashnode {
	return &Hashnode{Hashkey: k, Elem: r, LastUpdate: time.Now().UTC()}
}

// NewHashring returns a new hashring.
func NewHashring() *Hashring {

	h := &Hashring{}
	h.TestFunc = func(r Resource) {}
	return h
}

// String returns a string representation of ResourceDiff.
func (m *ResourceDiff) String() string {

	s := []string{}
	f := func(desc string, rMap ResourceMap) {
		for rType, rQueue := range rMap {
			s = append(s, fmt.Sprintf("%d %s %s", len(rQueue), desc, rType))
		}
	}
	f("new", m.New)
	f("changed", m.Changed)
	f("gone", m.Gone)

	return "Resource diff: " + strings.Join(s, ", ")
}

// Len implements the sort interface.
func (h *Hashring) Len() int {
	return len(h.Hashnodes)
}

// Less implements the sort interface.
func (h *Hashring) Less(i, j int) bool {
	return h.Hashnodes[i].Hashkey < h.Hashnodes[j].Hashkey
}

// Swap implements the sort interface.
func (h *Hashring) Swap(i, j int) {
	h.Hashnodes[i], h.Hashnodes[j] = h.Hashnodes[j], h.Hashnodes[i]
}

// ApplyDiff applies the given ResourceDiff to the hashring.  New resources are
// added, changed resources are updated, and gone resources are removed.
func (h *Hashring) ApplyDiff(d *ResourceDiff) {

	for rType, resources := range d.New {
		log.Printf("Adding %d resources of type %s.", len(resources), rType)
		for _, r := range resources {
			h.Add(r)
		}
	}
	for rType, resources := range d.Changed {
		log.Printf("Changing %d resources of type %s.", len(resources), rType)
		for _, r := range resources {
			h.AddOrUpdate(r)
		}
	}
	for rType, resources := range d.Gone {
		log.Printf("Removing %d resources of type %s.", len(resources), rType)
		for _, r := range resources {
			h.Remove(r)
		}
	}
}

// Diff determines the resources that are 1) in h1 but not h2 (new), 2) in both
// h1 and h2 but changed, and 3) in h2 but not h1 (gone).
func (h1 *Hashring) Diff(h2 *Hashring) *ResourceDiff {

	diff := NewResourceDiff()

	for _, n := range h1.Hashnodes {
		r1 := n.Elem

		index, err := h2.getIndex(r1.Uid())
		// The given resource is not present in h2, so it must be new.
		if err != nil {
			diff.New[r1.Type()] = append(diff.New[r1.Type()], r1)
			continue
		}

		// The given resource is present.  Did it change, though?
		r2 := h2.Hashnodes[index].Elem
		if r1.Oid() != r2.Oid() {
			diff.Changed[r1.Type()] = append(diff.Changed[r1.Type()], r1)
		}
	}

	// Finally, find resources that are gone.
	for _, n := range h2.Hashnodes {
		r2 := n.Elem

		_, err := h1.getIndex(r2.Uid())
		// The given resource is not present in h1, so it must be gone.
		if err != nil {
			diff.Gone[r2.Type()] = append(diff.Gone[r2.Type()], r2)
		}
	}

	return diff
}

// Add adds the given resource to the hashring.  If the resource is already
// present, we update its timestamp and return an error.
func (h *Hashring) Add(r Resource) error {
	h.Lock()
	defer h.Unlock()

	// Does the hashring already have the resource?
	if i, err := h.getIndex(r.Uid()); err == nil {
		h.Hashnodes[i].LastUpdate = time.Now().UTC()
		return errors.New("resource already present in hashring")
	}
	h.maybeTestResource(r)

	n := NewHashnode(r.Uid(), r)
	h.Hashnodes = append(h.Hashnodes, n)
	sort.Sort(h)
	return nil
}

// maybeTestResource may test the given resource.  The resource is *not* tested
// if all of the following conditions are met:
//
//   * The resource (as identified by its UID *and* OID) already exists.
//   * The resource has been last tested before the resource's expiry.
func (h *Hashring) maybeTestResource(r Resource) {

	var oldR Resource
	// Does the resource already exist in our hashring?
	if i, err := h.getIndex(r.Uid()); err == nil {
		oldR = h.Hashnodes[i].Elem
		// And is it exactly the same as the one we're dealing with?
		if oldR.Oid() == r.Oid() {
			rTest := oldR.Test()
			r = oldR
			// Is the resource already tested?
			if rTest != nil && rTest.State != StateUntested {
				// And if so, has it been tested recently?
				if time.Now().UTC().Sub(rTest.LastTested) < oldR.Expiry() {
					return
				}
			}
		}
	}
	if h.TestFunc != nil {
		go h.TestFunc(r)
	}
}

// AddOrUpdate attempts to add the given resource to the hashring.  If it
// already is in the hashring, we update it if (and only if) its object ID
// changed.
func (h *Hashring) AddOrUpdate(r Resource) {
	h.Lock()
	defer h.Unlock()

	h.maybeTestResource(r)
	// Does the hashring already have the resource?
	if i, err := h.getIndex(r.Uid()); err == nil {
		h.Hashnodes[i].LastUpdate = time.Now().UTC()
		// If so, we only update it if its object ID changed.
		if h.Hashnodes[i].Elem.Oid() != r.Oid() {
			h.Hashnodes[i].Elem = r
		}
	} else {
		n := NewHashnode(r.Uid(), r)
		h.Hashnodes = append(h.Hashnodes, n)
		sort.Sort(h)
	}
}

// Remove removes the given resource from the hashring.  If the hashring is
// empty or we cannot find the key, an error is returned.
func (h *Hashring) Remove(r Resource) error {
	h.Lock()
	defer h.Unlock()

	i, err := h.getIndex(r.Uid())
	if err != nil {
		return err
	}

	leftPart := h.Hashnodes[:i]
	rightPart := h.Hashnodes[i+1:]
	h.Hashnodes = append(leftPart, rightPart...)

	return nil
}

// getIndex attempts to return the index of the given hash key.  If the given
// hash key is present in the hashring, we return its index.  If the hashring
// is empty, an error is returned.  If the hash key cannot be found, an error
// is returned *and* the returned index is set to the *next* matching element
// in the hashring.
func (h *Hashring) getIndex(k Hashkey) (int, error) {

	if h.Len() == 0 {
		return -1, errors.New("hashring is empty")
	}

	i := sort.Search(h.Len(), func(i int) bool {
		return h.Hashnodes[i].Hashkey >= k
	})

	if i >= h.Len() {
		i = 0
	}

	if i < h.Len() && h.Hashnodes[i].Hashkey == k {
		return i, nil
	} else {
		return i, errors.New("could not find key in hashring")
	}
}

// Get attempts to retrieve the element identified by the given hash key.  If
// the hashring is empty, an error is returned.  If there is no exact match for
// the given hash key, we return the element whose hash key is the closest to
// the given hash key in descending direction.
func (h *Hashring) Get(k Hashkey) (Resource, error) {

	i, err := h.getIndex(k)
	if err != nil && i == -1 {
		return nil, err
	}
	return h.Hashnodes[i].Elem, nil
}

// GetExact attempts to retrieve the element identified by the given hash key.
// If we cannot find the element, an error is returned.
func (h *Hashring) GetExact(k Hashkey) (Resource, error) {

	i, err := h.getIndex(k)
	if err != nil {
		return nil, err
	}
	return h.Hashnodes[i].Elem, nil
}

// GetMany behaves like Get with the exception that it attempts to return the
// given number of elements.  If the number of desired elements exceeds the
// number of elements in the hashring, an error is returned.
func (h *Hashring) GetMany(k Hashkey, num int) ([]Resource, error) {

	if num > h.Len() {
		return nil, errors.New("requested more elements than hashring has")
	}

	var resources []Resource
	i, err := h.getIndex(k)
	if err != nil && i == -1 {
		return nil, err
	}

	for j := i; j < num+i; j++ {
		r := h.Hashnodes[j%h.Len()].Elem
		if r.Test().State != StateFunctional {
			log.Printf("Skipping %q because its state is %d.", r.String(), r.Test().State)
			continue
		}
		resources = append(resources, h.Hashnodes[j%h.Len()].Elem)
	}

	return resources, nil
}

// GetAll returns all of the hashring's resources.
func (h *Hashring) GetAll() []Resource {

	var elems []Resource
	for _, node := range h.Hashnodes {
		elems = append(elems, node.Elem)
	}
	return elems
}

// Filter filters the resources of this hashring with the given filter function
// and returns the remaining resources as another hashring.
func (h *Hashring) Filter(f FilterFunc) *Hashring {

	r := &Hashring{}
	for _, n := range h.Hashnodes {
		if f(n.Elem.(Resource)) {
			r.Add(n.Elem.(Resource))
		}
	}
	return r
}

// Prune prunes and returns expired resources from the hashring.
func (h *Hashring) Prune() []Resource {

	now := time.Now().UTC()
	pruned := []Resource{}

	for _, node := range h.Hashnodes {
		if now.Sub(node.LastUpdate) > node.Elem.Expiry() {
			pruned = append(pruned, node.Elem)
			h.Remove(node.Elem)
		}
	}

	return pruned
}
