package mechanisms

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/usecases/resources"
)

const (
	InterMessageDelimiter  = '\r'
	DefaultTimeBeforeRetry = time.Second * 1
	MaxTimeBeforeRetry     = time.Hour
)

// HttpsIpcContext implements the delivery.Mechanism interface.
type HttpsIpcContext struct {
	apiEndpoint     string
	messages        chan *core.HashringDiff
	done            chan bool
	wg              sync.WaitGroup
	timeBeforeRetry time.Duration
}

func NewHttpsIpc(apiEndpoint string) *HttpsIpcContext {

	return &HttpsIpcContext{apiEndpoint: apiEndpoint}
}

// StartStream initates the start of the HTTP resource stream.
func (ctx *HttpsIpcContext) StartStream(req *core.ResourceRequest) {
	ctx.messages = req.Receiver
	ctx.done = make(chan bool)
	ctx.wg.Add(1)
	ctx.timeBeforeRetry = DefaultTimeBeforeRetry
	go ctx.handleStream(req)
}

// StopStream signals the HTTP resource stream to stop and waits until it's
// done.
func (ctx *HttpsIpcContext) StopStream() {
	close(ctx.done)
	ctx.wg.Wait()
}

// MakeJsonRequest marshalls the given request into JSON, sends it to the
// destination that's set in the given context, and writes the resulting
// response to the given return interface.  If an error occurs, the function
// returns an error.
func (ctx *HttpsIpcContext) MakeJsonRequest(req interface{}, ret interface{}) error {

	resp, err := ctx.sendRequest(req, "")
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

	if err := json.Unmarshal(body, &ret); err != nil {
		return err
	}

	return nil
}

// expBackoff returns an exponentially increasing time duration with each
// subsequent call; starting at DefaultTimeBeforeRetry and maxing out at
// MaxTimeBeforeRetry.
func (ctx *HttpsIpcContext) expBackoff() time.Duration {

	ret := ctx.timeBeforeRetry
	ctx.timeBeforeRetry *= 2
	if ctx.timeBeforeRetry > MaxTimeBeforeRetry {
		ctx.timeBeforeRetry = MaxTimeBeforeRetry
	}
	return ret
}

// handleStream initiates our resource stream and relays information from the
// backend to the caller.  If our connection to the backend unexpectedly
// terminates, the function tries to establish a new connection, which is
// transparent to the caller.
func (ctx *HttpsIpcContext) handleStream(req *core.ResourceRequest) {

	defer ctx.wg.Done()
	retChan := make(chan error)
	defer close(retChan)
	incoming := make(chan []byte)
	defer close(incoming)

	// setupConn tries to create a persistent HTTP connection to our backend.
	// If that fails, the function continues to try again, indefinitely.  Once
	// a connection was established, the function relays all data from the
	// backend to the channel 'incoming'.  If the backend closes the connection
	// on us, the function writes the error to the channel 'retChan' and
	// returns.
	setupConn := func() {
		var err error
		var resp *http.Response
		for success := false; !success; success = (err == nil) {
			log.Printf("Making HTTP request to initiate resource stream.")
			resp, err = ctx.sendRequest(req, req.BearerToken)
			if err != nil {
				log.Printf("Error making HTTP request: %s", err.Error())
				log.Printf("Trying again in %s.", ctx.timeBeforeRetry)
				time.Sleep(ctx.expBackoff())
			}
		}
		defer resp.Body.Close()
		ctx.timeBeforeRetry = DefaultTimeBeforeRetry

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes(InterMessageDelimiter)
			if err != nil {
				retChan <- err
				return
			}
			incoming <- bytes.TrimSpace(line)
		}
	}

	go setupConn()
	for {
		select {
		// We got a new JSON chunk from our backend.
		case chunk := <-incoming:
			helper := resources.TmpHashringDiff{}
			if err := json.Unmarshal(chunk, &helper); err != nil {
				log.Printf("Error unmarshalling preliminary JSON from backend: %s", err)
				break
			}
			diff, err := resources.UnmarshalTmpHashringDiff(&helper)
			if err != nil {
				log.Printf("Error unmarshalling remaining JSON from backend: %s", err)
				break
			}
			ctx.messages <- diff
		// We lost our connection to the backend.  Let's try again.
		case err := <-retChan:
			log.Printf("Lost connection to backend (%s).  Retrying.", err.Error())
			go setupConn()
		// We're told to terminate.
		case <-ctx.done:
			log.Printf("Stopping HTTP resource stream.")
			return
		}
	}
}

// sendRequest marshalls the given request into JSON and sends it to the API
// endpoint that's part of the given context.  If not "", the function sets the
// given bearer token in the HTTP request.
func (ctx *HttpsIpcContext) sendRequest(req interface{}, bearerToken string) (*http.Response, error) {

	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Making new %s request to: %s", http.MethodGet, ctx.apiEndpoint)
	httpReq, err := http.NewRequest(http.MethodGet, ctx.apiEndpoint, bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
