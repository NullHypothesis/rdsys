package resources

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/crc64"
	"log"
	"net"
	"reflect"
	"strings"
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

// BridgeBase implements variables and methods that are shared by vanilla and
// pluggable transport bridges.
type BridgeBase struct {
	Protocol    string `json:"protocol"`
	Address     IPAddr `json:"address"`
	Port        uint16 `json:"port"`
	Fingerprint string `json:"fingerprint"`
	Distributor string `json:"-"`
}

// Bridge represents a Tor bridge.
type Bridge struct {
	core.ResourceBase
	BridgeBase
	FirstSeen  time.Time    `json:"-"`
	LastSeen   time.Time    `json:"-"`
	Transports []*Transport `json:"-"`
}

// IsPublic always returns false because neither vanilla nor pluggable
// transport bridges are public.  (Granted, our default bridges are, but we
// don't distribute them using rdsys.)
func (b *BridgeBase) IsPublic() bool {
	return false
}

// BridgeUid determines a bridge's hash key by first hashing its fingerprint,
// and then calculating a CRC-64 over a concatenation of the bridge's type and
// its hashed fingerprint.
func (b *BridgeBase) BridgeUid(rType string) core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)

	hFingerprint, err := HashFingerprint(b.Fingerprint)
	if err != nil {
		log.Printf("Bug: Error while hashing fingerprint %s.", b.Fingerprint)
		hFingerprint = b.Fingerprint
	}

	return core.Hashkey(crc64.Checksum([]byte(rType+hFingerprint), table))
}

// NewBridges allocates and returns a new Bridges object.
func NewBridges() *Bridges {
	b := &Bridges{}
	b.Bridges = make(map[string]*Bridge)
	return b
}

// NewBridge allocates and returns a new Bridge object.
func NewBridge() *Bridge {
	b := &Bridge{ResourceBase: *core.NewResourceBase()}
	// A bridge (without pluggable transports) is always running vanilla Tor
	// over TCP.
	b.Protocol = ProtoTypeTCP
	b.SetType(ResourceTypeVanilla)
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
	return b.Type() != "" && b.Address.String() != "" && b.Port != 0
}

func (b *Bridge) GetBridgeLine() string {
	return strings.TrimSpace(fmt.Sprintf("%s:%d %s", PrintTorAddr(&b.Address), b.Port, b.Fingerprint))
}

func (b *Bridge) Oid() core.Hashkey {
	table := crc64.MakeTable(Crc64Polynomial)
	return core.Hashkey(crc64.Checksum([]byte(b.GetBridgeLine()), table))
}

func (b *Bridge) Uid() core.Hashkey {
	return b.BridgeUid(b.RType)
}

func (b *Bridge) String() string {
	return b.GetBridgeLine()
}

func (b *Bridge) Expiry() time.Duration {
	// Bridges should upload new descriptors at least every 18 hours:
	// https://gitweb.torproject.org/torspec.git/tree/dir-spec.txt?id=c2a584144330239d6aa032b0acfb8b5ba26719fb#n369
	return time.Duration(time.Hour * 18)
}

func GetTorBridgeTypes() []string {
	return []string{ResourceTypeVanilla, ResourceTypeObfs4}
}

// PrintTorAddr takes as input a *IPAddr object and if it contains an IPv6
// address, it wraps it in square brackets.  This is necessary because Tor
// expects IPv6 addresses enclosed by square brackets.
func PrintTorAddr(a *IPAddr) string {
	s := a.String()
	if v4 := a.IP.To4(); len(v4) == net.IPv4len {
		return s
	} else {
		return fmt.Sprintf("[%s]", s)
	}
}

// HashFingerprint takes as input a bridge's fingerprint and hashes it using
// SHA-1, as discussed by Tor Metrics:
// https://metrics.torproject.org/onionoo.html#parameters_lookup
func HashFingerprint(fingerprint string) (string, error) {

	fingerprint = strings.TrimSpace(fingerprint)

	rawFingerprint, err := hex.DecodeString(fingerprint)
	if err != nil {
		return "", err
	}

	rawHFingerprint := sha1.Sum(rawFingerprint)
	hFingerprint := hex.EncodeToString(rawHFingerprint[:])
	return strings.ToUpper(hFingerprint), nil
}
