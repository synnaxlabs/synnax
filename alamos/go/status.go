// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
