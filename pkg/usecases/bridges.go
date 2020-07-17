package usecases

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	domain "rdb/pkg/domain"
)

const (
	BridgeTypeVanilla      = "vanilla"
	BridgeTypeObfs4        = "obfs4"
	BridgeTypeScrambleSuit = "scramblesuit"

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
	Type        string             `json:"type"`
	Protocol    string             `json:"protocol"`
	Address     IPAddr             `json:"address"`
	Port        uint16             `json:"port"`
	Fingerprint string             `json:"fingerprint"`
	Distributor string             `json:"-"`
	FirstSeen   time.Time          `json:"-"`
	LastSeen    time.Time          `json:"-"`
	BlockedIn   []*domain.Location `json:"-"`
	Transports  []*Transport       `json:"-"`
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
	b.Type = BridgeTypeVanilla
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

func (b *Bridge) GetBridgeLine() string {
	return fmt.Sprintf("%s:%d %s", b.Address.String(), b.Port, b.Fingerprint)
}
