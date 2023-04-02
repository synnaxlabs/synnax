package alamos

import "go.opentelemetry.io/otel/codes"

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
	Ok    Status = iota
	Error        = 1
)
