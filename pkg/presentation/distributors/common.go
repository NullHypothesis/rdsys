package distributors

import (
	"log"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/https"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/salmon"
)

// Run takes as input the name of a distributor and rdsys's config file.  The
// function then looks up the distributor's constructor and runs it.
func Run(distName string, cfg *internal.Config) {

	var constructors = map[string]func(*internal.Config){
		salmon.DistName: salmon.InitFrontend,
		https.DistName:  https.InitFrontend,
	}

	runFunc, exists := constructors[distName]
	if !exists {
		log.Fatalf("Distributor %q not found.", distName)
	}
	log.Printf("Running distributor %q.", distName)
	runFunc(cfg)
}
