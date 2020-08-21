package resources

import (
	"fmt"
	"hash/crc64"
	"strings"

	"gitlab.torproject.org/tpo/anti-censorship/ouroboros/pkg/core"
)

// Transport represents a Tor bridge's pluggable transport.
type Transport struct {
	core.ResourceBase
	Type        string            `json:"type"`
	Protocol    string            `json:"protocol"`
	Address     IPAddr            `json:"address"`
	Port        uint16            `json:"port"`
	Fingerprint string            `json:"fingerprint"`
	Parameters  map[string]string `json:"params,omitempty"`
	Bridge      *Bridge           `json:"-"`
	BlockedIn   []*core.Location  `json:"-"`
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

func (t *Transport) Name() string {
	return t.Type
}

func (t *Transport) IsDepleted() bool {
	return false
}

func (t *Transport) IsPublic() bool {
	return false
}

func (t *Transport) Oid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(t.String()), table))
}

func (t *Transport) Uid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(t.Fingerprint), table))
}
