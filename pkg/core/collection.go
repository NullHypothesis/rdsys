package core

import (
	"log"
	"sync"
)

const (
	ResourceIsNew = iota
	ResourceChanged
	ResourceIsGone
)

// BackendResources implements a collection of resources for our backend.  The
// backend uses this data structure to keep track of all of its resource types.
type BackendResources struct {
	sync.RWMutex
	// Collection maps a resource type (e.g. "obfs4") to its corresponding
	// split hashring.
	Collection map[string]*SplitHashring
	// EventRecipients maps a distributor name (e.g., "salmon") to an event
	// recipient struct that helps us keep track of notifying distributors when
	// their resources change.
	EventRecipients map[string]*EventRecipient
}

// EventRecipient represents the recipient of a resource event, i.e. a
// distributor; or rather, what we need to send updates to said distributor.
type EventRecipient struct {
	EventChans []chan *HashringDiff
	Request    *ResourceRequest
}

// NewBackendResources creates and returns a new resource collection.
func NewBackendResources(rNames []string, stencil *Stencil) *BackendResources {
	r := &BackendResources{}
	r.Collection = make(map[string]*SplitHashring)
	r.EventRecipients = make(map[string]*EventRecipient)

	for _, rName := range rNames {
		log.Printf("Creating split hashring for resource %q.", rName)
		r.Collection[rName] = NewSplitHashring()
		r.Collection[rName].Stencil = stencil
	}

	return r
}

// Add adds the given resource to the resource collection.  If the resource
// already exists but has changed (i.e. its unique ID remains the same but its
// object ID changed), we update the existing resource.
func (ctx *BackendResources) Add(r Resource) {

	hashring := ctx.Collection[r.Name()]
	if _, err := hashring.getIndex(r.Uid()); err != nil {
		if _, err := hashring.getIndex(r.Oid()); err == nil {
			// If the unique ID exists but the object ID doesn't, we're dealing
			// with an updated resource.
			ctx.propagateUpdate(r, ResourceChanged)
		}
	} else {
		// If the unique ID doesn't exist, we're dealing with a new resource.
		ctx.propagateUpdate(r, ResourceIsNew)
	}

	hashring.ForceAdd(r)
}

// Get returns a slice of resources of the requested type for the given
// distributor.
func (ctx *BackendResources) Get(distName string, rType string) []Resource {

	sHashring, exists := ctx.Collection[rType]
	if !exists {
		log.Printf("Requested resource type %q not present in our resource collection.", rType)
		return []Resource{}
	}

	resources, err := sHashring.GetForDist(distName)
	if err != nil {
		log.Printf("Failed to get resources for distributor %q: %s", distName, err)
	}
	return resources
}

// Prune removes expired resources.
func (ctx *BackendResources) Prune() {

	for _, hashring := range ctx.Collection {
		prunedResources := hashring.Prune()
		for _, resource := range prunedResources {
			ctx.propagateUpdate(resource, ResourceIsGone)
		}
	}
}

// propagateUpdate sends updates about new, changed, and gone resources to
// channels, allowing the backend to immediately inform a distributor of the
// update.
func (ctx *BackendResources) propagateUpdate(r Resource, event int) {
	ctx.Lock()
	defer ctx.Unlock()
	log.Printf("Propagating update to %d potential event recipients.",
		len(ctx.EventRecipients))

	// Prepare the hashring difference that we're about to send.
	diff := &HashringDiff{}
	rm := ResourceMap{r.Name(): []Resource{r}}
	switch event {
	case ResourceIsNew:
		diff.New = rm
	case ResourceChanged:
		diff.Changed = rm
	case ResourceIsGone:
		diff.Gone = rm
	}

	for distName, eventRecipient := range ctx.EventRecipients {

		// A distributor should only receive a diff if the resource in the diff
		// maps to the distributor.
		if !ctx.Collection[r.Name()].IsResourceInDist(r, distName) {
			log.Printf("Not propagating %s to %s.", r.Name(), distName)
			continue
		}
		if !ctx.EventRecipients[distName].Request.HasResourceType(r.Name()) {
			log.Printf("Skipping %s because %s didn't ask for this resource type.", r.Name(), distName)
			continue
		}

		log.Printf("%d channels for distributor recipient.", len(eventRecipient.EventChans))

		for _, c := range eventRecipient.EventChans {
			c <- diff
		}
	}
}

// RegisterChan registers a channel to be informed about resource updates.
func (ctx *BackendResources) RegisterChan(req *ResourceRequest, recipient chan *HashringDiff) {
	ctx.Lock()
	defer ctx.Unlock()

	distName := req.RequestOrigin
	log.Printf("Registered new channel for distributor %q to receive updates.", distName)
	_, exists := ctx.EventRecipients[distName]
	if !exists {
		er := &EventRecipient{Request: req, EventChans: []chan *HashringDiff{recipient}}
		ctx.EventRecipients[distName] = er
	} else {
		ctx.EventRecipients[distName].EventChans = append(ctx.EventRecipients[distName].EventChans, recipient)
	}
}

// UnregisterChan unregisters a channel to be informed about resource updates.
func (ctx *BackendResources) UnregisterChan(distName string, recipient chan *HashringDiff) {
	ctx.Lock()
	defer ctx.Unlock()

	chanSlice := ctx.EventRecipients[distName].EventChans
	newSlice := []chan *HashringDiff{}

	for i, c := range chanSlice {
		if c == recipient {
			log.Printf("Unregistering channel from recipients.")
			// Are we dealing with the last element in the slice?
			if i == len(chanSlice)-1 {
				newSlice = chanSlice[:i]
			} else {
				newSlice = append(chanSlice[:i], chanSlice[i+1:]...)
			}
			break
		}
	}
	ctx.EventRecipients[distName].EventChans = newSlice
}
