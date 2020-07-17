package main

import (
	"flag"
	"log"

	"rdb/internal"
	"rdb/internal/distributors"
)

func main() {
	// TODO: Maybe outsource flag parsing to shared code.
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

	var h distributors.HTTPSDistributor
	h.Init(cfg)
}
