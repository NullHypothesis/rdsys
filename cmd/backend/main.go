package main

import (
	"flag"
	"io"
	"log"
	"os"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
)

func main() {
	// TODO: Can we outsource flag parsing and share code across command line
	// tools?
	var configFilename, logFilename string
	flag.StringVar(&configFilename, "config", "", "Configuration file.")
	flag.StringVar(&logFilename, "log", "", "File to write logs to.")
	flag.Parse()

	var logOutput io.Writer = os.Stderr
	if logFilename != "" {
		logFd, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
		logOutput = logFd
		log.SetOutput(logOutput)
		defer logFd.Close()
	}

	if configFilename == "" {
		log.Fatal("No configuration file provided.  The argument -config is mandatory.")
	}
	cfg, err := internal.LoadConfig(configFilename)
	if err != nil {
		log.Fatal(err)
	}
	b := internal.BackendContext{}
	b.InitBackend(cfg)
}
