package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/delivery/mechanisms"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

// BackendContext contains the state that our backend requires.
type BackendContext struct {
	Config      *Config
	Resources   core.BackendResources
	bridgestrap delivery.Mechanism
}

// startWebApi starts our Web server.
func (b *BackendContext) startWebApi(cfg *Config, srv *http.Server) {
	log.Printf("Starting Web API at %s.", cfg.Backend.ApiAddress)

	mux := http.NewServeMux()
	mux.Handle(cfg.Backend.ResourceStreamEndpoint, http.HandlerFunc(b.resourcesHandler))
	mux.Handle(cfg.Backend.ResourcesEndpoint, http.HandlerFunc(b.resourcesHandler))
	mux.Handle(cfg.Backend.TargetsEndpoint, http.HandlerFunc(b.targetsHandler))
	srv.Handler = mux
	srv.Addr = cfg.Backend.ApiAddress

	var err error
	if cfg.Backend.Certfile != "" && cfg.Backend.Keyfile != "" {
		err = srv.ListenAndServeTLS(cfg.Backend.Certfile, cfg.Backend.Keyfile)
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

	bridgestrapCtx := mechanisms.NewHttpsIpc(cfg.Backend.BridgestrapEndpoint)

	for _, rType := range rTypes {
		b.Resources.Collection[rType].OnAddFunc = queryBridgestrap(bridgestrapCtx)
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

	diffs := make(chan *core.HashringDiff)
	b.Resources.RegisterChan(req, diffs)
	defer b.Resources.UnregisterChan(req.RequestOrigin, diffs)
	defer close(diffs)

	sendDiff := func(diff *core.HashringDiff) error {
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

	log.Printf("Sending client initial batch of resources.")
	resourceMap := b.processResourceRequest(req)
	if err := sendDiff(&core.HashringDiff{New: resourceMap}); err != nil {
		log.Printf("Error sending initial diff to client: %s.", err)
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
				log.Printf("Error sending diff to client: %s.", err)
				break
			}
		}
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

func (b *BackendContext) postResourcesHandler(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading HTTP body of %s: %s", r.RemoteAddr, err)
		http.Error(w, "failed to read HTTP body", http.StatusBadRequest)
		return
	}

	// Start by unmarshalling the resource base which is shared by all
	// resources.  We only care about the "type" field because it helps us
	// decide how we should unmarshal the remaining JSON.
	base := &core.ResourceBase{}
	if err := json.Unmarshal(body, base); err != nil {
		log.Printf("Error unmarshalling %s's resource type: %s", r.RemoteAddr, err)
		http.Error(w, "failed to unmarshal resource type", http.StatusBadRequest)
		return
	}

	resourceFunc, ok := resources.ResourceMap[base.Type]
	if !ok {
		log.Printf("Error obtaining struct for resource type %q for %s.", base.Type, r.RemoteAddr)
		http.Error(w, "given resource type not implemented", http.StatusNotImplemented)
		return
	}
	resource := resourceFunc()

	if err := json.Unmarshal(body, resource); err != nil {
		log.Printf("Error unmarshalling %s's resource struct: %s", r.RemoteAddr, err)
		http.Error(w, "failed to unmarshal resource struct", http.StatusBadRequest)
		return
	}

	b.Resources.Add(resource.(core.Resource))
	log.Printf("Added %s's %q resource to collection.", r.RemoteAddr, base.Type)
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
