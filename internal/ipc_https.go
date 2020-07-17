package internal

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"rdb/pkg"
)

type IPCContext struct {
	Config      *Config
	APIEndpoint string
}

func NewIpcContext(cfg *Config) *IPCContext {
	var c IPCContext
	c.Config = cfg

	proto := "https://"
	if c.Config.BackendKeyfile == "" {
		proto = "http://"
	}
	c.APIEndpoint = proto + c.Config.BackendAddress + "/" + c.Config.BackendEndpoint

	return &c
}

// RequestResources turns the given ResourceRequest into JSON, sends it to our
// HTTP API, decodes the response, and returns a slice of Resource objects.
func (c *IPCContext) RequestResources(req *pkg.ResourceRequest, i interface{}) error {

	log.Println("Making HTTPS IPC request.")

	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodGet, c.APIEndpoint, bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Printf("Trying to unmarshal %s", body)
	if err := json.Unmarshal(body, &i); err != nil {
		return err
	}
	return nil
}
