package resources

import (
	"fmt"
	"hash/crc64"
	"strings"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

// Transport represents a Tor bridge's pluggable transport.
type Transport struct {
	core.ResourceBase
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

func (t *Transport) IsValid() bool {
	return t.Type != "" && t.Address.String() != "" && t.Port != 0
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

func (t *Transport) Expiry() time.Duration {
	// Bridges should upload new descriptors at least every 18 hours:
	// https://gitweb.torproject.org/torspec.git/tree/dir-spec.txt?id=c2a584144330239d6aa032b0acfb8b5ba26719fb#n369
	return time.Duration(time.Hour * 18)
}

func (t *Transport) Oid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(t.String()), table))
}

func (t *Transport) Uid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(ResourceTypeObfs4+t.Fingerprint), table))
}
