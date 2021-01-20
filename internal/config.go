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
	ExtrainfoFile          string            `json:"extrainfo_file"`
	ApiTokens              map[string]string `json:"api_tokens"`
	ResourcesEndpoint      string            `json:"api_endpoint_resources"`
	ResourceStreamEndpoint string            `json:"api_endpoint_resource_stream"`
	TargetsEndpoint        string            `json:"api_endpoint_targets"`
	StatusEndpoint         string            `json:"web_endpoint_status"`
	MetricsEndpoint        string            `json:"web_endpoint_metrics"`
	BridgestrapEndpoint    string            `json:"bridgestrap_endpoint"`
	// DistProportions contains the proportion of resources that each
	// distributor should get.  E.g. if the HTTPS distributor is set to x and
	// the Salmon distributor is set to y, then HTTPS gets x/(x+y) of all
	// resources and Salmon gets y/(x+y).
	DistProportions    map[string]int `json:"distribution_proportions"`
	SupportedResources []string       `json:"supported_resources"`
	WebApi             WebApiConfig   `json:"web_api"`
}

type Distributors struct {
	Https  HttpsDistConfig  `json:"https"`
	Salmon SalmonDistConfig `json:"salmon"`
	Stub   StubDistConfig   `json:"stub"`
}

type StubDistConfig struct {
	Resources []string     `json:"resources"`
	WebApi    WebApiConfig `json:"web_api"`
}

type HttpsDistConfig struct {
	Resources []string     `json:"resources"`
	WebApi    WebApiConfig `json:"web_api"`
}

type SalmonDistConfig struct {
	Resources  []string     `json:"resources"`
	WebApi     WebApiConfig `json:"web_api"`
	WorkingDir string       `json:"working_dir"` // This is where Salmon stores its state.
}

type WebApiConfig struct {
	ApiAddress string `json:"api_address"`
	CertFile   string `json:"cert_file"`
	KeyFile    string `json:"key_file"`
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
