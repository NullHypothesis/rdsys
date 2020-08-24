package core

import (
	"errors"
	"log"
	"sort"
	"sync"
)

type HashringDiff struct {
	New     []Resource
	Changed []Resource
	Gone    []Resource
}

type Hashkey uint64

type Hashnode struct {
	Hashkey Hashkey
	Elem    Resource
}

type Hashring struct {
	Hashnodes []*Hashnode
	OnAddFunc OnAddFunc
	sync.RWMutex
}

type FilterFunc func(r Resource) bool
type OnAddFunc func(r Resource)

func NewHashring() *Hashring {

	h := &Hashring{}
	h.OnAddFunc = func(r Resource) {}
	return h
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

// Diff determines the resources that are 1) in h1 but not h2 (new), 2) in both
// h1 and h2 but changed, and 3) in h2 but not h1 (gone).
func (h1 *Hashring) Diff(h2 *Hashring) *HashringDiff {

	diff := &HashringDiff{}

	for _, n := range h1.Hashnodes {
		r1 := n.Elem

		index, err := h2.getIndex(r1.Uid())
		// The given resource is not present in h2, so it must be new.
		if err != nil {
			diff.New = append(diff.New, r1)
			continue
		}

		// The given resource is present.  Did it change, though?
		r2 := h2.Hashnodes[index].Elem
		if r1.Oid() != r2.Oid() {
			diff.Changed = append(diff.Changed, r1)
		}
	}

	// Finally, find resources that are gone.
	for _, n := range h2.Hashnodes {
		r2 := n.Elem

		_, err := h1.getIndex(r2.Uid())
		// The given resource is not present in h1, so it must be gone.
		if err != nil {
			diff.Gone = append(diff.Gone, r2)
		}
	}

	return diff
}

// Add adds the given resource to the hashring.  If the resource is already
// present, an error is returned.
func (h *Hashring) Add(r Resource) error {
	h.Lock()
	defer h.Unlock()

	// Does the hashring already have the resource?
	if _, err := h.getIndex(r.Uid()); err == nil {
		return errors.New("resource already present in hashring")
	}
	// Run our "on-add" hook.
	if h.OnAddFunc != nil {
		go h.OnAddFunc(r)
	}

	n := &Hashnode{r.Uid(), r}
	h.Hashnodes = append(h.Hashnodes, n)
	sort.Sort(h)
	return nil
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

	for j := i; j < num+i; {
		r := h.Hashnodes[j%h.Len()].Elem
		if r.GetState() != StateFunctional {
			log.Printf("Skipping %q because its state is %d.", r.String(), r.GetState())
			continue
		}
		resources = append(resources, h.Hashnodes[j%h.Len()].Elem)
		j++
	}

	return resources, nil
}

func (h *Hashring) GetAll() []Resource {
	var elems []Resource
	for _, node := range h.Hashnodes {
		elems = append(elems, node.Elem)
	}
	return elems
}

func (h *Hashring) Filter(f FilterFunc) *Hashring {

	r := &Hashring{}
	for _, n := range h.Hashnodes {
		if f(n.Elem.(Resource)) {
			r.Add(n.Elem.(Resource))
		}
	}
	return r
}
