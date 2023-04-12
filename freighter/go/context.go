// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import (
	"context"
	"crypto/tls"
	"github.com/synnaxlabs/x/address"
)

type Location uint8

//go:generate stringer -type=Location
const (
	// ClientSide indicates whether the middleware is located on the client side of the request.
	ClientSide Location = iota + 1
	// ServerSide indicates whether the middleware is located on the server side of the request.
	ServerSide
)

type Type uint8

//go:generate stringer -type=Type
const (
	// Unary is set on middleware that is executed for a unary request.
	Unary Type = iota + 1
	// Stream is set on middleware that is executed for a streaming request.
	Stream
)

// Context represents the metadata for a request that is passed to Middleware.
type Context struct {
	context.Context
	// Location indicates the location of the middleware (client or server).
	Location Location
	// Type indicates the type of the middleware (unary or stream).
	Type Type
	// Protocol is the protocol that the request is being sent over.
	Protocol string
	// Target is the address the request is being sent to.
	Target address.Address
	// Sec is the security information for the requests/response connection.
	Sec SecurityInfo
	// Params is a set of arbitrary parameters that can be set by client side middleware,
	// and read by server side middleware.
	Params
}

// SecurityInfo represents the security information for a request.
type SecurityInfo struct {
	// TLS contains the TLS information for the request. If Used is false, the connection
	// is not protected by TLS, and the ConnectionState is invalid.
	TLS struct {
		// Used is set to true if TLS is being used.
		Used bool
		// ConnectionState is the TLS connection state.
		tls.ConnectionState
	}
}

// Params is a set of arbitrary parameters that can be set by client side middleware, and
// read by server side middleware.
type Params map[string]interface{}

// GetRequired returns the value of the given key, or panics if the key is not set. It
// should only be used in contexts where the absence of a key represents a programming
// error.
func (p Params) GetRequired(k string) interface{} {
	v, ok := p[k]
	if !ok {
		panic("missing required param: " + k)
	}
	return v
}

// GetDefault returns the value of the given key, or the given default value if the key
// is not set.
func (p Params) GetDefault(k string, def interface{}) interface{} {
	v, ok := p[k]
	if !ok {
		return def
	}
	return v
}

// Get returns the value of the given key, or nil if the key is not set.
func (p Params) Get(k string) (interface{}, bool) {
	v, ok := p[k]
	return v, ok
}

// Set sets the value of the given key.
func (p Params) Set(k string, v interface{}) { p[k] = v }

// SetIfAbsent sets the value of the given key if it is not already set.
func (p Params) SetIfAbsent(k string, v interface{}) {
	if _, ok := p[k]; !ok {
		p[k] = v
	}
}
