package core

import (
	"errors"
	"log"
	"math/rand"
)

// Stencil is a list of intervals that implements a "view" that can be
// overlayed over a hashring.  Distributor-specific stencils make it easy to
// deterministically select non-overlapping subsets of a hashring that should
// be given to a distributor.
type Stencil struct {
	intervals []*Interval
}

// SplitHashring represents a hashring with a corresponding stencil.  The
// backend uses one SplitHashring per resource type to map resources to
// distributors.
type SplitHashring struct {
	*Hashring
	*Stencil
}

// Interval represents a numerical interval.
type Interval struct {
	Begin int
	End   int
	Name  string
}

// NewSplitHashring returns a new SplitHashring.
func NewSplitHashring() *SplitHashring {
	return &SplitHashring{NewHashring(), &Stencil{}}
}

// Contains returns 'true' if the given number n falls into the interval [a, b]
// so that a <= n <= b.
func (i *Interval) Contains(n int) bool {
	return i.Begin <= n && n <= i.End
}

// FindByValue attempts to return the interval that the given number falls into
// and an error otherwise.
func (s *Stencil) FindByValue(n int) (*Interval, error) {
	for _, interval := range s.intervals {
		if interval.Contains(n) {
			return interval, nil
		}
	}
	return nil, errors.New("no interval that contains given value")
}

// AddInterval adds the given interval to the stencil.
func (s *Stencil) AddInterval(i *Interval) {
	s.intervals = append(s.intervals, i)
}

// GetUpperEnd returns the the maximum of all intervals of the stencil.
func (s *Stencil) GetUpperEnd() (int, error) {

	if len(s.intervals) == 0 {
		return 0, errors.New("cannot determine upper end of empty stencil")
	}

	max := 0
	for _, interval := range s.intervals {
		if interval.End > max {
			max = interval.End
		}
	}
	return max, nil
}

// DoesDistOwnResource returns true if the given resource maps to the given
// distributor and false otherwise.
func (s *Stencil) DoesDistOwnResource(r Resource, distName string) bool {

	filterFunc, err := s.GetFilterFunc(distName)
	if err != nil {
		log.Printf("Bug: GetFilterFunc complained: %s", err)
		return false
	}
	return filterFunc(r)
}

// GetFilterFunc returns a hashring filter function which, when applied to a
// hashring, returns a subset of the hashring.  The idea is that the given
// distributor name results in a function that deterministically maps to a
// non-overlapping set of hashring resources.
//
// Consider the following example: we have three obfs4 resources (O1, O2, and
// O3) and two distributors (moat and https).  GetFilterFunc returns a filter
// function that deterministically maps O1 and O2 to moat, and O3 to https.
func (s *Stencil) GetFilterFunc(distName string) (FilterFunc, error) {

	upperEnd, err := s.GetUpperEnd()
	if err != nil {
		return nil, err
	}

	// This function returns 'true' if the given resource should be assigned to
	// the given distributor name.  The function uses a deterministic random
	// number generator to that end.
	f := func(r Resource) bool {

		// What interval does the resource's hash fall into?
		seed := r.Uid()
		rand.Seed(int64(seed))
		n := rand.Intn(upperEnd + 1)

		i, err := s.FindByValue(n)
		if err != nil {
			log.Printf("Bug: resource %q does not fall in any interval.", r.String())
			return false
		}
		return i.Name == distName
	}
	return f, nil
}

// GetForDist takes as input a distributor's name (e.g. "moat") and returns the
// resources that are allocated for the given distributor.
func (h *SplitHashring) GetForDist(distName string) ([]Resource, error) {

	filterFunc, err := h.Stencil.GetFilterFunc(distName)
	if err != nil {
		return []Resource{}, err
	}

	subHashring := h.Hashring.Filter(filterFunc)
	var resources []Resource
	for _, elem := range subHashring.GetAll() {
		resources = append(resources, elem.(Resource))
	}

	return resources, nil
}

// Len returns the length of the SplitHashring.
func (h *SplitHashring) Len() int {
	return h.Hashring.Len()
}
