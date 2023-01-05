// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package errors implements standard error types and formats to provide uniform,
// parseable exceptions to delta API consumers.
//
// All errors in this package are variations of the Typed struct; an error's type gives
// the API consumer information on how it should be parsed. All errors in this package
// can be encoded, and implement freighter interfaces for go-internal and cross language
// parsing.
package errors

import (
	"github.com/synnaxlabs/x/query"
	"github.com/cockroachdb/errors"
)

// Nil is a typed representation of a nil error.
var Nil = Typed{Type: TypeNil}

var Canceled = Typed{Type: TypeGeneral, Err: Message{Message: "request cancelled"}}

// Message is an error that contains a simple string message.
type Message struct {
	Message string `json:"message" msgpack:"message"`
}

// Error implements the error interface.
func (g Message) Error() string { return g.Message }

func newTypedMessage(t Type, msg string) Typed {
	return Typed{Type: t, Err: Message{Message: msg}}
}

// General is an error that doesn't fit into a specific Type. General errors should be
// used in the case where an error is expected to occur during normal use, such as a
// query returning multiple results when it should return exactly one. Unexpected errors
// should be used in the case where an error should not
// occur during normal use.
func General(err error) Typed { return newTypedMessage(TypeGeneral, err.Error()) }

// MaybeGeneral is a convenience function for returning a General error if the
// error is not nil.
func MaybeGeneral(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return General(err)
}

// Unexpected represents an error that should not occur during normal execution.
// For example, a seemingly healthy node is unreachable or a transaction fails
// to commit.
func Unexpected(err error) Typed { return newTypedMessage(TypeUnexpected, err.Error()) }

// MaybeUnexpected is a convenience function for returning a Unexpected error if
// the error is not nil.
func MaybeUnexpected(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Unexpected(err)
}

// Parse is an error that occurs when a user-supplied value cannot be parsed.
// For example, a user sends msgpack data when it should be JSON.
func Parse(err error) Typed { return newTypedMessage(TypeParse, err.Error()) }

// MaybeParse is a convenience function for returning a Parse error if the error
// is not nil.
func MaybeParse(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Parse(err)
}

// Auth is an error that occurs when a user is not authorized to perform an
// action or when authorization fails.
func Auth(err error) Typed { return newTypedMessage(TypeAuth, err.Error()) }

// MaybeAuth is a convenience function for returning a Auth error if the error
// is not nil.
func MaybeAuth(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Auth(err)
}

// Query is an error that occurs when a particular query fails.
func Query(err error) Typed {
	if errors.Is(err, query.NotFound) {
		return newTypedMessage(TypeQuery, err.Error())
	}
	if errors.Is(err, query.UniqueViolation) {
		return newTypedMessage(TypeQuery, err.Error())
	}
	return General(err)
}

// MaybeQuery is a convenience function for returning a Query error if the error
// is not nil.
func MaybeQuery(err error) Typed {
	if t, ok := maybe(err); ok {
		return t
	}
	return Query(err)
}

func maybe(err error) (Typed, bool) {
	if err == nil {
		return Nil, true
	}
	if t, ok := err.(Typed); ok {
		return t, true
	}
	return Typed{}, false
}

func Route(err error, path string) Typed {
	return Typed{
		Type: TypeRoute,
		Err:  routeError{Path: path, Message: Message{Message: err.Error()}},
	}
}

type routeError struct {
	Path string `json:"path" msgpack:"path"`
	Message
}
