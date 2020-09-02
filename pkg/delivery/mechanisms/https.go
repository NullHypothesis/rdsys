package mechanisms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

type HttpsIpcContext struct {
	ApiEndpoint string
	ApiMethod   string
}

func (c *HttpsIpcContext) MakeRequest(req interface{}, ret interface{}) error {

	log.Println("Making HTTPS IPC request.")
	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(c.ApiMethod, c.ApiEndpoint, bytes.NewBuffer(encoded))
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
	if err := json.Unmarshal(body, &ret); err != nil {
		return err
	}
	return nil
}

// RequestResources turns the given ResourceRequest into JSON, sends it to our
// HTTP API, decodes the response, and returns a slice of Resource objects.
func (c *HttpsIpcContext) RequestResources(req *core.ResourceRequest, i interface{}) error {

	log.Println("Making HTTPS IPC request.")

	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodGet, c.ApiEndpoint, bytes.NewBuffer(encoded))
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got HTTP status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Printf("Trying to unmarshal %s", body)
	if err := json.Unmarshal(body, i); err != nil {
		return err
	}
	return nil
}
