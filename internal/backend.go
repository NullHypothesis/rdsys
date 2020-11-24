package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc64"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

// BackendContext contains the state that our backend requires.
type BackendContext struct {
	Config    *Config
	Resources core.BackendResources
	rTestPool *ResourceTestPool
}

// startWebApi starts our Web server.
func (b *BackendContext) startWebApi(cfg *Config, srv *http.Server) {
	log.Printf("Starting Web API at %s.", cfg.Backend.WebApi.ApiAddress)

	mux := http.NewServeMux()
	mux.Handle(cfg.Backend.StatusEndpoint, http.HandlerFunc(b.statusHandler))
	mux.Handle(cfg.Backend.ResourceStreamEndpoint, http.HandlerFunc(b.resourcesHandler))
	mux.Handle(cfg.Backend.ResourcesEndpoint, http.HandlerFunc(b.resourcesHandler))
	mux.Handle(cfg.Backend.TargetsEndpoint, http.HandlerFunc(b.targetsHandler))
	srv.Handler = mux
	srv.Addr = cfg.Backend.WebApi.ApiAddress

	var err error
	if cfg.Backend.WebApi.CertFile != "" && cfg.Backend.WebApi.KeyFile != "" {
		err = srv.ListenAndServeTLS(cfg.Backend.WebApi.CertFile, cfg.Backend.WebApi.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	log.Printf("Web API shut down: %s", err)
}

// stopWebApi stops our Web server.
func (b *BackendContext) stopWebApi(srv *http.Server) {
	// Give our Web server five seconds to shut down.
	t := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), t)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error while shutting down Web API: %s", err)
	}
}

// InitBackend initialises our backend.
func (b *BackendContext) InitBackend(cfg *Config) {

	log.Println("Initialising backend.")
	b.Config = cfg
	rTypes := []string{}
	for rType, _ := range resources.ResourceMap {
		rTypes = append(rTypes, rType)
	}
	b.Resources = *core.NewBackendResources(rTypes, BuildStencil(cfg.Backend.DistProportions))

	b.rTestPool = NewResourceTestPool(cfg.Backend.BridgestrapEndpoint)
	defer b.rTestPool.Stop()
	for _, rType := range rTypes {
		b.Resources.Collection[rType].OnAddFunc = b.rTestPool.AddFunc()
	}

	quit := make(chan bool)

	var wg sync.WaitGroup
	ready := make(chan bool, 1)
	go func() {
		wg.Add(1)
		defer wg.Done()
		InitKraken(cfg, quit, ready, b.Resources)
	}()

	var srv http.Server
	go func() {
		wg.Add(1)
		defer wg.Done()
		b.startWebApi(cfg, &srv)
	}()

	// Wait until our data kraken parsed our bridge descriptors.
	<-ready
	log.Println("Kraken finished parsing bridge descriptors.")

	// We're done bootstrapping.  Now wait for a SIGTERM.
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint
	log.Println("Received SIGINT.")
	close(quit)
	b.stopWebApi(&srv)

	// Wait for goroutines to finish.
	wg.Wait()
	log.Println("All goroutines have finished.  Exiting.")
}

// extractResourceRequest extracts a ResourceRequest from the given HTTP
// request.  If an error occurs, the function writes the error to the given
// response writer and returns an error.
func extractResourceRequest(w http.ResponseWriter, r *http.Request) (*core.ResourceRequest, error) {

	var req *core.ResourceRequest

	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Printf("Failed to read HTTP body.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	if err := json.Unmarshal(b, &req); err != nil {
		log.Printf("Failed to unmarshal HTTP body %q.", b)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, err
	}

	return req, nil
}

// isAuthenticated authenticates the given HTTP request.  If this fails, it
// writes an error to the given ResponseWriter and returns false.
func (b *BackendContext) isAuthenticated(w http.ResponseWriter, r *http.Request) bool {

	// First, we take the bearer token from the 'Authorization' HTTP header.
	tokenLine := r.Header.Get("Authorization")
	if tokenLine == "" {
		log.Printf("Request carries no 'Authorization' HTTP header.")
		http.Error(w, "request carries no 'Authorization' HTTP header", http.StatusBadRequest)
		return false
	}
	if !strings.HasPrefix(tokenLine, "Bearer ") {
		log.Printf("Authorization header contains no bearer token.")
		http.Error(w, "authorization header contains no bearer token", http.StatusBadRequest)
		return false
	}
	fields := strings.Split(tokenLine, " ")
	givenToken := fields[1]

	// Do we have the given token on record?
	for _, savedToken := range b.Config.Backend.ApiTokens {
		if givenToken == savedToken {
			return true
		}
	}
	log.Printf("Invalid authentication token.")
	http.Error(w, "invalid authentication token", http.StatusUnauthorized)

	return false
}

func (b *BackendContext) getResourceStreamHandler(w http.ResponseWriter, r *http.Request) {

	if !b.isAuthenticated(w, r) {
		return
	}

	req, err := extractResourceRequest(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "http streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	diffs := make(chan *core.ResourceDiff)
	b.Resources.RegisterChan(req, diffs)
	defer b.Resources.UnregisterChan(req.RequestOrigin, diffs)
	defer close(diffs)

	sendDiff := func(diff *core.ResourceDiff) error {
		jsonBlurb, err := json.MarshalIndent(diff, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if _, err := fmt.Fprintf(w, string(jsonBlurb)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		fmt.Fprintf(w, "\r") // delimiter
		flusher.Flush()
		return nil
	}

	resourceMap := b.processResourceRequest(req)
	log.Printf("Sending distributor initial batch: %s", resourceMap)
	if err := sendDiff(&core.ResourceDiff{New: resourceMap}); err != nil {
		log.Printf("Error sending initial diff to distributor: %s.", err)
	}

	log.Printf("Entering streaming loop for %s.", r.RemoteAddr)
	for {
		select {
		// Is our HTTP connection done?
		case <-r.Context().Done():
			log.Printf("Exiting streaming loop for %s.", r.RemoteAddr)
			// Consume remaining hashring differences.
			for {
				select {
				case diff := <-diffs:
					log.Printf("Sending remaining hashring diff.")
					sendDiff(diff)
				default:
					return
				}
			}
		case diff := <-diffs:
			if err := sendDiff(diff); err != nil {
				log.Printf("Error sending diff to distributor: %s.", err)
				break
			}
		}
	}
}

func (b *BackendContext) statusHandler(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse parameters", http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "no 'id' parameter given", http.StatusBadRequest)
		return
	}
	id = strings.TrimSpace(id)
	id = strings.ToUpper(id)

	var result []string
	result = append(result, fmt.Sprintf("Bridge %s advertises:\n\n", id))

	// Iterate over each resource type that contains the given UID and add it
	// to the final result.
	foundResource := false
	table := crc64.MakeTable(resources.Crc64Polynomial)
	statuses := []string{"not yet tested", "functional", "dysfunctional"}
	for rType, _ := range resources.ResourceMap {
		sHashring, exists := b.Resources.Collection[rType]
		if !exists {
			continue
		}

		key := core.Hashkey(crc64.Checksum([]byte(rType+id), table))
		resource, err := sHashring.GetExact(key)
		if err != nil {
			// We may have been given a non-hashed fingerprint.  Let's try to
			// hash it, and see if we get a result.
			hId, err := resources.HashFingerprint(id)
			if err != nil {
				continue
			}
			key := core.Hashkey(crc64.Checksum([]byte(rType+hId), table))
			resource, err = sHashring.GetExact(key)
			if err != nil {
				continue
			}
		}
		foundResource = true

		rResult := fmt.Sprintf("* %s: %s\n", rType, statuses[resource.Test().State])
		if resource.Test().Error != "" {
			rResult += fmt.Sprintf("  Error: %s\n", resource.Test().Error)
		}
		if resource.Test().State != core.StateUntested {
			lastTested := resource.Test().LastTested
			tDiff := time.Now().UTC().Sub(lastTested)
			rResult += fmt.Sprintf("  Last tested: %s (%s ago)\n", lastTested, tDiff)
		}
		result = append(result, rResult+"\n")
	}
	if !foundResource {
		http.Error(w, "no resources for the given id", http.StatusNotFound)
	} else {
		fmt.Fprintf(w, strings.Join(result, ""))
	}
}

func (b *BackendContext) processResourceRequest(req *core.ResourceRequest) core.ResourceMap {

	resources := make(core.ResourceMap)
	for _, rType := range req.ResourceTypes {
		resources[rType] = b.Resources.Get(req.RequestOrigin, rType)
	}

	return resources
}

func (b *BackendContext) getResourcesHandler(w http.ResponseWriter, r *http.Request) {

	if !b.isAuthenticated(w, r) {
		return
	}

	req, err := extractResourceRequest(w, r)
	if err != nil {
		return
	}
	log.Printf("Distributor %q is asking for %q.", req.RequestOrigin, req.ResourceTypes)

	var resources []core.Resource
	for _, rType := range req.ResourceTypes {
		resources = append(resources, b.Resources.Get(req.RequestOrigin, rType)...)
	}
	log.Printf("Returning %d resources of type %s to distributor %q.",
		len(resources), req.ResourceTypes, req.RequestOrigin)

	jsonBlurb, err := json.Marshal(resources)
	if err != nil {
		http.Error(w, "error while turning resources into JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jsonBlurb))
}

// UnmarshalResources unmarshals a slice of raw JSON messages into the
// corresponding resources.
func UnmarshalResources(rawResources []json.RawMessage) ([]core.Resource, error) {

	rs := []core.Resource{}
	for _, rawResource := range rawResources {
		base := core.ResourceBase{}
		if err := json.Unmarshal(rawResource, &base); err != nil {
			return nil, err
		}

		if base.Type() == "" {
			return nil, errors.New("missing \"type\" field")
		}

		rFunc, ok := resources.ResourceMap[base.Type()]
		if !ok {
			return nil, fmt.Errorf("resource type %q not implemented", base.Type())
		}
		r := rFunc()

		if err := json.Unmarshal(rawResource, r); err != nil {
			return nil, errors.New("failed to unmarshal resource struct")
		}

		if !r.(core.Resource).IsValid() {
			return nil, fmt.Errorf("resource %q is not valid", base.Type())
		}
		rs = append(rs, r.(core.Resource))
	}

	return rs, nil
}

// postResourcesHandler handles POST requests that register a resource with our
// backend.
func (b *BackendContext) postResourcesHandler(w http.ResponseWriter, req *http.Request) {

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error reading %s's request body: %s", req.RemoteAddr, err)
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	rawResources := []json.RawMessage{}
	if err := json.Unmarshal(body, &rawResources); err != nil {
		log.Printf("Error unmarshalling %s's raw resources: %s", req.RemoteAddr, err)
		http.Error(w, "failed to unmarshal raw resources", http.StatusBadRequest)
		return
	}

	rs, err := UnmarshalResources(rawResources)
	if err != nil {
		log.Printf("Error unmarshalling %s's resources: %s", req.RemoteAddr, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, r := range rs {
		b.Resources.Add(r)
		log.Printf("Added %s's %q resource to collection.", req.RemoteAddr, r.Type())
	}
}

// resourcesHandler handles requests coming from distributors (if it's GET
// requests) and from proxies (if it's POST requests).
func (b *BackendContext) resourcesHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		if r.URL.Path == b.Config.Backend.ResourcesEndpoint {
			b.getResourcesHandler(w, r)
		} else if r.URL.Path == b.Config.Backend.ResourceStreamEndpoint {
			b.getResourceStreamHandler(w, r)
		}
	case http.MethodPost:
		if r.URL.Path == b.Config.Backend.ResourcesEndpoint {
			b.postResourcesHandler(w, r)
		}
	default:
		log.Printf("Received unsupported request method %q from %s.", r.Method, r.RemoteAddr)
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}

// targetsHandler handles requests coming from censorship measurement clients
// like OONI.
func (b *BackendContext) targetsHandler(w http.ResponseWriter, r *http.Request) {

	if !b.isAuthenticated(w, r) {
		return
	}
	http.Error(w, "not yet implemented", http.StatusInternalServerError)
}
