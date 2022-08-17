package freighter

import "errors"

var (
	// Unreachable is returned when a target cannot be reached.
	Unreachable = errors.New("[freighter] - target unreachable")
)

// Payload represents a piece of data that can be sent over the freighter.
type Payload = any

type Transport interface {
	Digest() Digest
}

type Digest struct {
	Protocol  string
	Encodings []string
}

func (d Digest) LogArgs() []interface{} {
	return []interface{}{
		"protocol",
		d.Protocol,
		"encodings",
		d.Encodings,
	}
}
