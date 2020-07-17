package distributors

import (
	"log"

	"rdb/internal"
	"rdb/pkg"
)

const (
	DistributorName = "https"
)

type HTTPSDistributor struct {
}

func (d *HTTPSDistributor) Init(cfg *internal.Config) error {
	log.Printf("Initialising %s distributor.", d.GetName())

	var bridges []pkg.Transport

	ipc := internal.NewIpcContext(cfg)
	req := pkg.ResourceRequest{DistributorName, "obfs4"}
	if err := ipc.RequestResources(&req, &bridges); err != nil {
		log.Printf("Error while requesting resources: %s", err)
	}
	for _, transport := range bridges {
		log.Println(transport.GetBridgeLine())
	}

	return nil
}

func (d *HTTPSDistributor) Shutdown() error {
	log.Printf("Shutting down %s distributor.", d.GetName())
	return nil
}

func (d *HTTPSDistributor) GetName() string {
	return DistributorName
}
