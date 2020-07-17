package internal

// Resource represents a (potentially scarce) resource that we seek to
// distribute.  This can be a Tor bridge, Tor Brower download links, or
// snowflake proxies.
type Resource interface {
	String() string
	GetID() []byte
	IsPublic() bool
	// GetThreatModel() *ThreatModel
	// GetBlockingLocation() *Location
	// Return how the bridge wants to be distributed
	// GetDistributor() []Distributor
}

type ResourceSet interface {
	// Reload all resources from wherever they are.
	Load() *Resource
	Reload()
	Populate()
	Request() *Resource
}

type Distributor interface {
	Init() error
	Shutdown() error
	GetName() string
}

type Backend interface {
	GetResource() *Resource
}

// type IPCContext interface {
// 	RequestResources(*ResourceRequest, interface{}) error
// }
