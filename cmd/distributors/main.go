package main

import (
	"flag"
	"log"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/presentation/distributors"
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

	distributors.Run(distName, cfg)
}
