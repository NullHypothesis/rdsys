package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// Config represents our central configuration file.
type Config struct {
	ExtrainfoFile   string `json:"extrainfo_file"`
	BackendAddress  string `json:"backend_address"`
	BackendEndpoint string `json:"backend_endpoint"`
	BackendCertfile string `json:"backend_certfile"`
	BackendKeyfile  string `json:"backend_keyfile"`
}

// LoadConfig loads the given JSON configuration file and returns the resulting
// Config configuration object.
func LoadConfig(filename string) (*Config, error) {

	log.Printf("Attempting to load configuration file at %s.", filename)

	info, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	if info.Mode() != 0600 {
		return nil, fmt.Errorf("file %s contains secrets and therefore must have 0600 permissions", filename)
	}

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err = json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
