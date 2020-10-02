package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StateUntested = iota
	StateFunctional
	StateNotFunctional
)

type ResourceRepository interface {
	Serialise(string) error
	Deserialise(string) error

	Len()
	ApplyDiff()
	Get(Hashkey) (Resource, error)
}

type Resource interface {
	Name() string
	String() string
	IsDepleted() bool
	IsPublic() bool
	IsValid() bool
	BlockedIn() LocationSet
	SetBlockedIn(LocationSet)
	// Uid returns the resource's unique identifier.  Bridges with different
	// fingerprints have different unique identifiers.
	Uid() Hashkey
	// Oid returns the resource's object identifier.  Bridges with the *same*
	// fingerprint but different, say, IP addresses have different object
	// identifiers.  If two resources have the same Oid, they must have the
	// same Uid but not vice versa.
	Oid() Hashkey
	SetState(int)
	GetState() int
	// Expiry returns the duration after which the resource should be deleted
	// from the backend (if the backend hasn't received an update).
	Expiry() time.Duration
}

type Requester interface {
	Hash()
	IsTransient() bool
}

// ResourceMap maps a resource type to a slice of respective resources.
type ResourceMap map[string]ResourceQueue

// ResourceQueue implements a queue of resources.
type ResourceQueue []Resource

// Enqueue adds a resource to the queue.  The function returns an error if the
// resource already exists in the queue.
func (q *ResourceQueue) Enqueue(r1 Resource) error {
	for _, r2 := range *q {
		if r1.Uid() == r2.Uid() {
			return errors.New("resource already exists")
		}
	}
	*q = append(*q, r1)
	return nil
}

// Dequeue return and removes the oldest resource in the queue.  If the queue
// is empty, the function returns an error.
func (q *ResourceQueue) Dequeue() (Resource, error) {
	if len(*q) == 0 {
		return nil, errors.New("queue is empty")
	}

	r := (*q)[0]
	if len(*q) > 1 {
		*q = (*q)[1:]
	} else {
		*q = []Resource{}
	}

	return r, nil
}

// Delete removes the resource from the queue.  If the queue is empty, the
// function returns an error.
func (q *ResourceQueue) Delete(r1 Resource) error {
	if len(*q) == 0 {
		return errors.New("queue is empty")
	}

	// See the following article on why this works:
	// https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	new := (*q)[:0]
	for _, r2 := range *q {
		if r1.Uid() != r2.Uid() {
			new = append(new, r2)
		}
	}

	*q = new
	return nil
}

// Update updates an existing resource if its unique ID matches the unique ID
// of the given resource.  If the queue is empty, the function returns an
// error.
func (q *ResourceQueue) Update(r1 Resource) error {
	if len(*q) == 0 {
		return errors.New("queue is empty")
	}

	for i, r2 := range *q {
		if r1.Uid() == r2.Uid() {
			(*q)[i] = r1
		}
	}

	return nil
}

func (m ResourceMap) String() string {
	s := []string{}
	for rType, queue := range m {
		s = append(s, fmt.Sprintf("%s: %d", rType, len(queue)))
	}
	return strings.Join(s, ", ")
}

// ApplyDiff applies the given HashringDiff to the ResourceMap.  New resources
// are added, changed resources are updated, and gone resources are removed.
func (m ResourceMap) ApplyDiff(d *HashringDiff) {

	for rType, resources := range d.New {
		for _, r := range resources {
			q := m[rType]
			q.Enqueue(r)
			m[rType] = q
		}
	}

	for rType, resources := range d.Changed {
		for _, r := range resources {
			q := m[rType]
			q.Update(r)
			m[rType] = q
		}
	}

	for rType, resources := range d.Gone {
		for _, r := range resources {
			q := m[rType]
			q.Delete(r)
			m[rType] = q
		}
	}
}

// Location represents the physical and topological location of a resource or
// requester.
type Location struct {
	CountryCode string // ISO 3166-1 alpha-2 country code, e.g. "AR".
	ASN         uint32 // Autonomous system number, e.g. 1234.
}

// String returns the string representation of the given location, e.g. "RU
// 1234".
func (l *Location) String() string {
	if l.ASN == 0 {
		return fmt.Sprintf("%s", l.CountryCode)
	} else {
		return fmt.Sprintf("%s (%d)", l.CountryCode, l.ASN)
	}
}

// LocationSet maps the string representation of locations (because we cannot
// use structs as map keys) to 'true'.
type LocationSet map[string]bool

// HasLocationsNotIn returns true if s1 contains at least one location that is
// not in s2.
func (s1 LocationSet) HasLocationsNotIn(s2 LocationSet) bool {
	for key, _ := range s1 {
		if _, exists := s2[key]; !exists {
			return true
		}
	}
	return false
}

type ResourceBase struct {
	Type      string `json:"type"`
	Location  *Location
	Id        uint
	blockedIn LocationSet
	State     int

	Requesters []Requester
}

func NewResourceBase() *ResourceBase {
	r := &ResourceBase{}
	r.blockedIn = make(LocationSet)
	return r
}

func (r *ResourceBase) SetState(state int) {
	r.State = state
}

func (r *ResourceBase) GetState() int {
	return r.State
}

func (r *ResourceBase) BlockedIn() LocationSet {
	return r.blockedIn
}

func (r *ResourceBase) SetBlockedIn(l *Location) {
	r.blockedIn[l.String()] = true
}

type ResourceRequest struct {
	// Name of requesting distributor.
	RequestOrigin string             `json:"request_origin"`
	ResourceTypes []string           `json:"resource_types"`
	BearerToken   string             `json:"-"`
	Receiver      chan *HashringDiff `json:"-"`
}

func (r *ResourceRequest) HasResourceType(rType1 string) bool {

	for _, rType2 := range r.ResourceTypes {
		if rType1 == rType2 {
			return true
		}
	}
	return false
}

type Response struct {
}
