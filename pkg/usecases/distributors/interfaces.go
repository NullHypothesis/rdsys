package distributors

import (
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/internal"
)

// Distributor represents a distribution mechanism, e.g. Salmon or HTTPS.
type Distributor interface {
	Init(*internal.Config)
	Shutdown()
}
