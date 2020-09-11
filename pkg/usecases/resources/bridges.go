package resources

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"

	"hash/crc64"
	"sync"
	"time"

	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

const (
	ProtoTypeTCP = "tcp"
	ProtoTypeUDP = "udp"

	DistributorMoat        = "moat"
	DistributorHttps       = "https"
	DistributorEmail       = "email"
	DistributorUnallocated = "unallocated"

	BridgeReloadInterval = time.Hour
)

// IPAddr embeds net.IPAddr.  The only difference to net.IPAddr is that we
// implement a MarshalJSON method that allows for convenient marshalling of IP
// addresses.
type IPAddr struct {
	net.IPAddr
}

func (a IPAddr) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *IPAddr) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &a.IPAddr.IP)
}

// Bridges represents a set of Bridge objects.
type Bridges struct {
	m       sync.Mutex
	Bridges map[string]*Bridge
}

// Bridge represents a Tor bridge.
type Bridge struct {
	core.ResourceBase
	Protocol    string           `json:"protocol"`
	Address     IPAddr           `json:"address"`
	Port        uint16           `json:"port"`
	Fingerprint string           `json:"fingerprint"`
	Distributor string           `json:"-"`
	FirstSeen   time.Time        `json:"-"`
	LastSeen    time.Time        `json:"-"`
	BlockedIn   []*core.Location `json:"-"`
	Transports  []*Transport     `json:"-"`
}

func (b Bridge) IsPublic() bool {
	return false
}

// NewBridges allocates and returns a new Bridges object.
func NewBridges() *Bridges {
	b := &Bridges{}
	b.Bridges = make(map[string]*Bridge)
	return b
}

// NewBridge allocates and returns a new Bridge object.
func NewBridge() *Bridge {
	b := &Bridge{}
	// A bridge (without pluggable transports) is always running vanilla Tor
	// over TCP.
	b.Protocol = ProtoTypeTCP
	b.Type = ResourceTypeVanilla
	return b
}

// AddTransport adds the given transport to the bridge.
func (b *Bridge) AddTransport(t1 *Transport) {
	for _, t2 := range b.Transports {
		if reflect.DeepEqual(t1, t2) {
			// We already have this transport on record.
			return
		}
	}
	b.Transports = append(b.Transports, t1)
}

func (b *Bridge) IsValid() bool {
	return b.Type != "" && b.Address.String() != "" && b.Port != 0
}

func (b *Bridge) GetBridgeLine() string {
	return fmt.Sprintf("%s:%d %s", b.Address.String(), b.Port, b.Fingerprint)
}

func (b *Bridge) Oid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(b.GetBridgeLine()), table))
}

func (b *Bridge) Uid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(b.Fingerprint), table))
}

func (b *Bridge) IsDepleted() bool {
	return false
}

func (b *Bridge) String() string {
	return b.GetBridgeLine()
}

func (b *Bridge) Name() string {
	return b.Type
}

func (b *Bridge) Expiry() time.Duration {
	// Bridges should upload new descriptors at least every 18 hours:
	// https://gitweb.torproject.org/torspec.git/tree/dir-spec.txt?id=c2a584144330239d6aa032b0acfb8b5ba26719fb#n369
	return time.Duration(time.Hour * 18)
}

func GetTorBridgeTypes() []string {
	return []string{ResourceTypeVanilla, ResourceTypeObfs4}
}
