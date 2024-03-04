// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ferrors

type Type string

const (
	// TypeEmpty represents an error that hasn't been properly parsed or detected.
	// TypeEmpty errors typically represent a programming error.
	TypeEmpty = Type("")
	// TypeNil represents a nil error i.e. one that has not occurred.
	TypeNil = Type("nil")
	// TypeUnknown represents an error that was not registered with the ferrors package.
	TypeUnknown = Type("unknown")
	// TypeRoach represents an error type that was encoded using cockroachdb's errors package.
	// This is the default error type for errors that are not registered with the ferrors package,
	// and is used mostly for go-to-go communication.
	TypeRoach = Type("roach")
)

type Error interface {
	error
	// FreighterType returns the type of the error. Freighter uses this to determine the
	// correct decode to use on the other end of the transport.
	FreighterType() Type
}

// Typed is the easiest way to create an error type that satisfies the Error interface.
// It takes the error and attaches the provided type to it. Then you can define
// encode and decode functions for the type and register them with the ferrors package.
func Typed(err error, t Type) Error {
	if err == nil {
		return Nil
	}
	// Check if it already satisfies the Details interface.
	if tErr, ok := err.(Error); ok {
		return tErr
	}
	return &typed{error: err, t: t}
}

type typed struct {
	error `json:"error"`
	t     Type
}

var _ Error = (*typed)(nil)

// FreighterType implements Error.
func (t typed) FreighterType() Type { return t.t }

var Nil = &typed{error: nil, t: TypeNil}
