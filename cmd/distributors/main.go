package main

import (
	"flag"
	"log"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	httpsUI "gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/https"
	salmonWeb "gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/salmon"
	stubWeb "gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors/stub"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors/https"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors/salmon"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/distributors/stub"
)

func main() {
	var configFilename, distName string
	flag.StringVar(&distName, "name", "", "Distributor name.")
	flag.StringVar(&configFilename, "config", "", "Configuration file.")
	flag.Parse()

	if distName == "" {
		log.Fatal("No distributor name provided.  The argument -name is mandatory.")
	}

	if configFilename == "" {
		log.Fatal("No configuration file provided.  The argument -config is mandatory.")
	}
	cfg, err := internal.LoadConfig(configFilename)
	if err != nil {
		log.Fatal(err)
	}

	var constructors = map[string]func(*internal.Config){
		salmon.DistName: salmonWeb.InitFrontend,
		https.DistName:  httpsUI.InitFrontend,
		stub.DistName:   stubWeb.InitFrontend,
	}
	runFunc, exists := constructors[distName]
	if !exists {
		log.Fatalf("Distributor %q not found.", distName)
	}

	log.Printf("Starting distributor %q.", distName)
	runFunc(cfg)
}
