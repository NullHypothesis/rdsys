package core

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
}

type Requester interface {
	Hash()
	IsTransient() bool
	// Location     *Location
}

type IpcMechanism interface {
	// Allows distributors to periodically fetch updated resources.
	RequestResources(*ResourceRequest, interface{}) error
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
	RequestOrigin string   `json:"request_origin"`
	ResourceTypes []string `json:"resource_types"`
}

type Response struct {
}
