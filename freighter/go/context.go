// Copyright 2025 Synnax Labs, Inc.
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

// Role indicates whether the middleware is located on the client- or server-side of the
// request.
type Role uint8

//go:generate stringer -type=Role
const (
	// Client indicates that the middleware is located on the client-side of the
	// request.
	Client Role = iota + 1
	// Server indicates that the middleware is located on the server-side of the
	// request.
	Server
)

// Variant indicates the variant of transport (unary or streaming) that the middleware
// is being executed for.
type Variant uint8

//go:generate stringer -type=Variant
const (
	// Unary is set on middleware that is executed for a unary request.
	Unary Variant = iota + 1
	// Stream is set on middleware that is executed for a streaming request.
	Stream
)

// Context represents the metadata for a request that is passed to Middleware.
type Context struct {
	context.Context
	Role
	Variant
	// Protocol is the protocol that the request is being sent over.
	Protocol string
	// Target is the address the request is being sent to.
	Target address.Address
	SecurityInfo
	Params
}

// SecurityInfo represents the security information for a request.
type SecurityInfo struct {
	// TLS contains the TLS information for the request. If Used is false, the
	// connection is not protected by TLS, and the ConnectionState is invalid.
	TLS struct {
		// Used is set to true if TLS is being used.
		Used bool
		// ConnectionState is the TLS connection state.
		tls.ConnectionState
	}
}

// Params is a set of arbitrary parameters that can be set by client-side middleware,
// and read by server-side middleware.
type Params map[string]any

// GetDefault returns the value of the given key, or the given default value if the key
// is not set.
func (p Params) GetDefault(k string, def any) any {
	v, ok := p[k]
	if !ok {
		return def
	}
	return v
}

// Get returns the value of the given key, or nil if the key is not set.
func (p Params) Get(k string) (any, bool) {
	v, ok := p[k]
	return v, ok
}

// Set sets the value of the given key.
func (p Params) Set(k string, v any) { p[k] = v }
