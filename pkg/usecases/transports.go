package usecases

import (
	"fmt"
	"strings"

	domain "rdb/pkg/domain"
)

// Transport represents a Tor bridge's pluggable transport.
type Transport struct {
	Resource    domain.ResourceBase
	Type        string             `json:"type"`
	Protocol    string             `json:"protocol"`
	Address     IPAddr             `json:"address"`
	Port        uint16             `json:"port"`
	Fingerprint string             `json:"fingerprint"`
	Parameters  map[string]string  `json:"params,omitempty"`
	Bridge      *Bridge            `json:"-"`
	BlockedIn   []*domain.Location `json:"-"`
}

// NewTransport returns a new Transport object.
func NewTransport() *Transport {
	t := &Transport{}
	// As of 2020-05-19, all of our pluggable transports are based on TCP, so
	// we might as well make it the default for now.
	t.Protocol = ProtoTypeTCP
	t.Parameters = make(map[string]string)
	return t
}

func (t *Transport) String() string {

	var args []string
	for key, value := range t.Parameters {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}

	return fmt.Sprintf("%s %s:%d %s %s", t.Type, t.Address.String(), t.Port, t.Fingerprint, strings.Join(args, " "))
}

func (t *Transport) IsDepleted() bool {
	return false
}

func (t *Transport) IsPublic() bool {
	return false
}
