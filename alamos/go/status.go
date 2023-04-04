package alamos

import "go.opentelemetry.io/otel/codes"

// Status represents the general status of an operation.
type Status uint8

var _otelStatusCodes = map[Status]codes.Code{
	Ok:    codes.Ok,
	Error: codes.Error,
}

func (s Status) otel() codes.Code {
	v, ok := _otelStatusCodes[s]
	if !ok {
		return codes.Unset
	}
	return v
}

const (
	// Ok represents a successful operation.
	Ok Status = iota
	// Error represents a failed operation.
	Error = 1
)
