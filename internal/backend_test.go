package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

func TestAuthentication(t *testing.T) {

	b := BackendContext{}
	tokens := make(map[string]string)
	tokens["https"] = "8M4WSTrhwatWYGDWJw1OtS2cDXYfJtAetCcaFP94lYo="
	b.Config = &Config{BackendConfig{ApiTokens: tokens}, Distributors{}}

	rr := httptest.NewRecorder()
	r := &http.Request{}
	if b.isAuthenticated(rr, r) {
		t.Error("broken request passed authentication")
	}
}

func TestUnmarshalResources(t *testing.T) {

	rs, err := UnmarshalResources([]json.RawMessage{[]byte("")})
	if err == nil {
		t.Fatalf("raw json with missing type field was accepted")
	}

	rs, err = UnmarshalResources([]json.RawMessage{[]byte("{\"type\": \"foo\"}")})
	if err == nil {
		t.Fatalf("non-existing resource type was accepted")
	}

	rs, err = UnmarshalResources([]json.RawMessage{[]byte("{\"type\": \"obfs4\"}")})
	if err == nil {
		t.Fatalf("incomplete resource type was accepted")
	}

	obfs4Submission := []byte("{\"type\": \"obfs4\", \"address\": \"1.2.3.4\", \"port\": 1234}")
	rs, err = UnmarshalResources([]json.RawMessage{obfs4Submission, obfs4Submission})
	if err != nil {
		t.Fatalf("valid resource was rejected")
	}
	if len(rs) != 2 {
		t.Errorf("incorrect number of resources extracted")
	}
}

func TestPostResourcesHandler(t *testing.T) {

	b := BackendContext{}
	b.Config = &Config{}
	b.Config.Backend.ApiTokens = make(map[string]string)
	b.Config.Backend.ApiTokens["foo"] = "bar"

	b.Resources = *core.NewBackendResources([]string{"obfs4"}, nil)

	rr := httptest.NewRecorder()
	body := strings.NewReader("[{\"type\": \"obfs4\", \"address\": \"1.2.3.4\", \"port\": 1234}]")
	req, err := http.NewRequest("POST", "/resources", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer bar")

	b.postResourcesHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected HTTP return code 200 but got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	body = strings.NewReader("")
	req, err = http.NewRequest("POST", "/resources", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer bar")

	b.postResourcesHandler(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected HTTP return code 400 but got %d", rr.Code)
	}
}
