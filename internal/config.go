package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

// Config represents our central configuration file.
type Config struct {
	Backend      BackendConfig `json:"backend"`
	Distributors Distributors  `json:"distributors"`
}

type BackendConfig struct {
	ExtrainfoFile     string         `json:"extrainfo_file"`
	ApiTokens         []string       `json:"api_tokens"`
	ApiAddress        string         `json:"api_address"`
	ResourcesEndpoint string         `json:"api_endpoint_resources"`
	TargetsEndpoint   string         `json:"api_endpoint_targets"`
	Certfile          string         `json:"certfile"`
	Keyfile           string         `json:"keyfile"`
	DistProportions   map[string]int `json:"distribution_proportions"`
}

type Distributors struct {
	Https  HttpsDistConfig  `json:"https"`
	Salmon SalmonDistConfig `json:"salmon"`
}

type HttpsDistConfig struct {
	ApiAddress string `json:"api_address"`
}

type SalmonDistConfig struct {
	WorkingDirectory string `json:"working_directory"`
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

// TODO: This function may belong somewhere else.
// BuildIntervalChain turns the distributor proportions into an interval chain,
// which helps us determine what distributor a given resource should map to.
func BuildStencil(proportions map[string]int) *core.Stencil {

	var keys []string
	for key, _ := range proportions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	stencil := &core.Stencil{}
	i := 0
	for _, k := range keys {
		stencil.AddInterval(&core.Interval{i, i + proportions[k] - 1, k})
		i += proportions[k]
	}
	return stencil
}
