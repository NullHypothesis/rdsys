package pkg

import (
	"errors"
)

type ResourceRepository interface {
	Store(r Resource)
	Retrieve(id uint) *Resource
}

type Resource interface {
	String() string
	IsDepleted() bool
	IsPublic() bool
}

type Requester interface {
	IsTransient() bool
}

type Distributor interface {
	RateLimitRequester(r *Requester)
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
	Location  *Location
	Id        uint
	BlockedIn map[CountryCode]bool

	Requesters []Requester
}

func NewResourceBase() *ResourceBase {
	r := &ResourceBase{}
	r.BlockedIn = make(map[CountryCode]bool)
	return r
}

type DistributorBase struct {
	Resources map[uint]*ResourceBase
}

func (d *DistributorBase) RequestResource(r *Requester) ([]*ResourceBase, error) {

	if d.RateLimitRequester(r) {
		return nil, errors.New("requester is rate limited")
	}

	// TODO: use bridgedb's hash ring idea, i.e., map requester's id (ip
	// address, email address, ...) to a point in the hashring of resource ids
}

func (r *ResourceBase) IsBlockedIn(l *Location) bool {
	_, exists := r.BlockedIn[l.CountryCode]
	return exists
}

func (r *ResourceBase) SetBlockedIn(l *Location) {
	// Maybe update trust levels?
}

type int TrustLevel

// TODO: Do we really need a separate Request object?
type Request struct {
	Trustability TrustLevel
}

type Requester struct {
	Trustability TrustLevel
	Location     *Location
}
