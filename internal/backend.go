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
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/delivery"
)

// BackendContext contains the state that our backend requires.
type BackendContext struct {
	Config      *Config
	Resources   ResourceCollection
	bridgestrap delivery.Mechanism
}

type ResourceHook func(core.Resource) error

// ResourceCollection maps a resource type (e.g., "obfs4") to a hashring
// stencil.
// TODO: This data structure may belong to a different layer of abstraction.
type ResourceCollection struct {
	StencilTemplate *core.Stencil
	Collection      map[string]*core.SplitHashring
}

func (r ResourceCollection) Add(name string, resource core.Resource) {

	rName := resource.Name()
	_, exists := r.Collection[rName]
	if !exists {
		log.Printf("Creating new split hashring for resource %q.", rName)
		r.Collection[rName] = core.NewSplitHashring()
		r.Collection[rName].Stencil = *r.StencilTemplate
		r.Collection[rName].OnAddFunc = func(r core.Resource) {
		}
	}
	r.Collection[rName].Add(resource)
}

func (r ResourceCollection) Get(distName string, rType string) []core.Resource {

	sHashring, exists := r.Collection[rType]
	if !exists {
		log.Printf("Requested resource type %q not present in our resource collection.", rType)
		return []core.Resource{}
	}

	resources, err := sHashring.GetForDist(distName)
	if err != nil {
		log.Printf("Failed to get resources for distributor %q: %s", distName, err)
	}
	return resources
}

// startWebApi starts our Web server.
func (b *BackendContext) startWebApi(cfg *Config, srv *http.Server) {
	log.Printf("Starting Web API at %s.", cfg.Backend.ApiAddress)

	mux := http.NewServeMux()
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
	log.Printf("Web server shut down: %s", err)
}

// stopWebApi stops our Web server.
func (b *BackendContext) stopWebApi(srv *http.Server) {
	// Give our Web server five seconds to shut down.
	t := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), t)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error while stopping Web API: %s", err)
	}
}

// InitBackend initialises our backend.
func (b *BackendContext) InitBackend(cfg *Config) {

	log.Println("Initialising backend.")
	b.Config = cfg
	b.Resources.Collection = make(map[string]*core.SplitHashring)
	b.Resources.StencilTemplate = BuildStencil(cfg.Backend.DistProportions)
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

func (b *BackendContext) getResourcesHandler(w http.ResponseWriter, r *http.Request) {
	// Here's how we can test the API using the command line:
	// curl -X GET localhost:7100 -d '{"request_origin":"https","resource_types":["obfs4"]}'
	req, err := extractResourceRequest(w, r)
	if err != nil {
		return
	}
	log.Printf("Distributor %q is asking for %q.", req.RequestOrigin, req.ResourceTypes)

	// TODO: This needs to be re-architected to support a long-poll request
	// model:
	// https://lucasroesler.com/2018/07/golang-long-polling-a-tale-of-server-timeouts/
	var resources []core.Resource
	for _, rType := range req.ResourceTypes {
		resources = append(resources, b.Resources.Get(req.RequestOrigin, rType)...)
	}
	log.Printf("Returning %d resources to distributor %q.", len(resources), req.RequestOrigin)

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
	http.Error(w, "not yet implemented", http.StatusInternalServerError)
}

// resourcesHandler handles requests coming from distributors (if it's GET
// requests) and from proxies (if it's POST requests).
func (b *BackendContext) resourcesHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		b.getResourcesHandler(w, r)
	} else if r.Method == http.MethodPost {
		b.postResourcesHandler(w, r)
	} else {
		log.Printf("Received unsupported request method %q from %s.", r.Method, r.RemoteAddr)
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
	}
}

// targetsHandler handles requests coming from censorship measurement clients
// like OONI.
func (b *BackendContext) targetsHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not yet implemented", http.StatusInternalServerError)
}
