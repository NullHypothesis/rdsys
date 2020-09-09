package delivery

import (
	"gitlab.torproject.org/tpo/anti-censorship/rdsys/pkg/core"
)

type Mechanism interface {
	StartStream(*core.ResourceRequest)
	StopStream()
	MakeJsonRequest(interface{}, interface{}) error
}
