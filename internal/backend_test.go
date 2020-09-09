package internal

import (
	"net/http"
	"testing"
)

func TestAuthentication(t *testing.T) {

	b := BackendContext{}
	tokens := make(map[string]string)
	tokens["https"] = "8M4WSTrhwatWYGDWJw1OtS2cDXYfJtAetCcaFP94lYo="
	b.Config = &Config{BackendConfig{ApiTokens: tokens}, Distributors{}}

	w := http.ResponseWriter{}
	r := &http.Request{}
	if b.IsAuthenticated(w, r) {
		t.Error("broken request passed authentication")
	}
}
