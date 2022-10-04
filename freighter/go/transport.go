package freighter

import (
	"errors"
	"github.com/synnaxlabs/x/alamos"
)

var (
	// Unreachable is returned when a target cannot be reached.
	Unreachable = errors.New("[freighter] - target unreachable")
)

// Payload represents a piece of data that can be sent over the freighter.
type Payload = any

type Transport interface {
	alamos.Reporter
	Use(...Middleware)
}

type Reporter struct {
	Protocol  string
	Encodings []string
}

func (t Reporter) Report() alamos.Report {
	rep := make(alamos.Report)
	rep["protocol"] = t.Protocol
	rep["encodings"] = t.Encodings
	return rep
}
