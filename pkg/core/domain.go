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
	Store(r Resource)
	Retrieve(id uint) *Resource
}

type Resource interface {
	Name() string
	String() string
	IsDepleted() bool
	IsPublic() bool
	IsValid() bool
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
	// Location     *Location
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

// CountryCode holds an ISO 3166-1 alpha-2 country code, e.g., "AR".
type CountryCode string

// ASN holds an autonomous system number, e.g., 1234.
type ASN uint32

// Location represents the physical and topological location of a resource or
// requester.
type Location struct {
	CountryCode CountryCode
	ASN         ASN
}

type ResourceBase struct {
	Type      string `json:"type"`
	Location  *Location
	Id        uint
	BlockedIn map[CountryCode]bool
	State     int

	Requesters []Requester
}

func (r *ResourceBase) SetState(state int) {
	r.State = state
}

func (r *ResourceBase) GetState() int {
	return r.State
}

func NewResourceBase() *ResourceBase {
	r := &ResourceBase{}
	r.BlockedIn = make(map[CountryCode]bool)
	return r
}

func (r *ResourceBase) IsBlockedIn(l *Location) bool {
	_, exists := r.BlockedIn[l.CountryCode]
	return exists
}

func (r *ResourceBase) SetBlockedIn(l *Location) {
	// Maybe update trust levels?
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
