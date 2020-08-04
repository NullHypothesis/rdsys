package main

import (
	"flag"
	"log"

	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/internal"
	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/usecases/distributors"
)

func main() {
	// TODO: Can we outsource flag parsing and share code across command line
	// tools?
	var configFilename string
	flag.StringVar(&configFilename, "config", "", "Configuration file.")
	flag.Parse()

	if configFilename == "" {
		log.Fatal("No configuration file provided.  The argument -config is mandatory.")
	}
	cfg, err := internal.LoadConfig(configFilename)
	if err != nil {
		log.Fatal(err)
	}

	var h distributors.HttpsDistributor
	h.Init(cfg)
}
