package core

import (
	"errors"
	"sort"
)

type Hashkey uint64

type Hashnode struct {
	Hashkey Hashkey
	Elem    interface{}
}

type Hashring struct {
	Hashnodes []*Hashnode
}

// Len implements the sort interface.
func (h *Hashring) Len() int {
	return len(h.Hashnodes)
}

// Less implements the sort interface.
func (h *Hashring) Less(i, j int) bool {
	return h.Hashnodes[i].Hashkey < h.Hashnodes[j].Hashkey
}

// Swap implements the Swap interface.
func (h *Hashring) Swap(i, j int) {
	h.Hashnodes[i], h.Hashnodes[j] = h.Hashnodes[j], h.Hashnodes[i]
}

// Add adds the given resource to the hashring.  If the resource is already
// present, an error is returned.
func (h *Hashring) Add(r Resource) error {

	// Does the hashring already have the resource?
	if _, err := h.getIndex(r.Hash()); err == nil {
		return errors.New("resource already present in hashring")
	}

	n := &Hashnode{r.Hash(), r}
	h.Hashnodes = append(h.Hashnodes, n)
	sort.Sort(h)
	return nil
}

// Remove removes the given resource from the hashring.  If the hashring is
// empty or we cannot find the key, an error is returned.
func (h *Hashring) Remove(r Resource) error {

	i, err := h.getIndex(r.Hash())
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
func (h *Hashring) Get(k Hashkey) (interface{}, error) {

	i, err := h.getIndex(k)
	if err != nil && i == -1 {
		return nil, err
	}
	return h.Hashnodes[i].Elem, nil
}

// GetMany behaves like Get with the exception that it attempts to return the
// given number of elements.  If the number of desired elements exceeds the
// number of elements in the hashring, an error is returned.
func (h *Hashring) GetMany(k Hashkey, num int) ([]interface{}, error) {

	if num > h.Len() {
		return nil, errors.New("requested more elements than hashring has")
	}

	var r []interface{}
	i, err := h.getIndex(k)
	if err != nil && i == -1 {
		return nil, err
	}

	for j := i; j < num+i; j++ {
		r = append(r, h.Hashnodes[j%h.Len()].Elem)
	}

	return r, nil
}
