package distributors

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/internal"
	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/delivery/mechanisms"
)

const (
	HttpsDistName        = "https"
	BridgeReloadInterval = time.Minute * 10
)

type HttpsDistributor struct {
	srv     http.Server
	bridges []core.Resource
	ring    core.Hashring
	ipc     core.IpcMechanism
	cfg     *internal.Config
}

// requestResources periodically requests updated bridges from our backend.
func (d *HttpsDistributor) periodicTasks() {

	ticker := time.NewTicker(BridgeReloadInterval)
	defer ticker.Stop()
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)

	for {
		select {
		case <-ticker.C:
			d.requestResources()
			for _, transport := range d.bridges {
				log.Println(transport.String())
			}
		case <-sigint:
			log.Println("Caught SIGINT.")
			if err := d.Shutdown(); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (d *HttpsDistributor) requestResources() {
	req := core.ResourceRequest{HttpsDistName, []string{"obfs4"}}

	if err := d.ipc.RequestResources(&req, &d.bridges); err != nil {
		log.Printf("Error while requesting resources: %s", err)
	}
}

// Init starts our Web server.
func (d *HttpsDistributor) Init(cfg *internal.Config) error {
	log.Printf("Initialising %s distributor.", HttpsDistName)

	d.cfg = cfg
	// d.ipc = delivery.NewHttpsIpcContext(cfg)
	httpsIpc := &mechanisms.HttpsIpcContext{}
	//httpsIpc.ApiEndpoint = cfg.Distributors.Https.ApiAddress
	httpsIpc.ApiEndpoint = "http://" + cfg.Backend.ApiAddress + cfg.Backend.ResourcesEndpoint
	httpsIpc.ApiMethod = http.MethodGet
	d.ipc = httpsIpc
	d.requestResources()
	go d.periodicTasks()

	// Dummy bridge distribution function.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, d.bridges[0].String())
	})

	log.Println("Starting Web server.")
	d.srv.Addr = cfg.Distributors.Https.ApiAddress
	if err := d.srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

	return nil
}

// Shutdown shuts down our Web server.
func (d *HttpsDistributor) Shutdown() error {
	log.Printf("Shutting down %s distributor.", HttpsDistName)

	// Give our Web server five seconds to shut down.
	t := time.Now().Add(5 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), t)
	defer cancel()

	if err := d.srv.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
