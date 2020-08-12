package pkg

// ResourceRequest represents a request for resources that a distributor sends
// to the backend.
type ResourceRequest struct {
	// Name of requesting distributor.
	// RequestOrigin string `json:"request_origin"`
	// ResourceType  string `json:"resource_type"`
}

// TestTargetRequest represents a request for resources to scan.
type TestTargetRequest struct {
	Id        string `json:"id"`
	ProbeType string `json:"type"`
	Location  string `json:"country_code"`
}
