// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"github.com/synnaxlabs/alamos"
	"net"

	"github.com/cockroachdb/cmux"
)

// BranchContext is the context for operating a Branch.
type BranchContext struct {
	alamos.Instrumentation
	// List is the listener the branch should listen for incoming requests on.
	Lis net.Listener
	// ServerName is the name of the Server this branch is running on.
	ServerName string
	// Security contains the security configuration for the Server.
	Security SecurityConfig
	// Debug is a flag to enable debugging endpoints and utilities.
	Debug bool
}

// RoutingPolicy determines how a Branch should be served depending on the security
// configuration of the Server.
type RoutingPolicy int

const (
	// ServeOnlyIfInsecure serves the Branch only if the server is running in insecure
	// mode.
	ServeOnlyIfInsecure RoutingPolicy = iota + 1
	// ServeOnlyIfSecure serves the Branch only if the server is running in secure mode.
	ServeOnlyIfSecure
	// ServeOnInsecureIfSecure serves the Branch without TLS if the server is running
	// in secure mode.
	ServeOnInsecureIfSecure
	// ServeAlwaysPreferSecure serves the Branch with TLS if the server is running in
	// secure mode and without TLS if the server is running in insecure mode.
	ServeAlwaysPreferSecure
	// ServeAlwaysPreferInsecure serves the Branch without TLS regardless of the server
	// mode.
	ServeAlwaysPreferInsecure
)

// ShouldServe returns true if the Branch should be served under the given listening
// conditions.
func (r RoutingPolicy) ShouldServe(insecure, insecureMux bool) bool {
	if !insecure && !insecureMux {
		return r == ServeAlwaysPreferSecure || r == ServeOnlyIfSecure
	} else if !insecure && insecureMux {
		return r == ServeOnInsecureIfSecure || r == ServeAlwaysPreferInsecure
	} else if insecure && insecureMux {
		return r == ServeAlwaysPreferInsecure || r == ServeOnlyIfInsecure || r == ServeAlwaysPreferSecure
	}
	panic("[server]  - invalid routing policy")
}

// BranchRouting is the information provided by a Branch to the Server so that it can
// appropriately route requests to it.
type BranchRouting struct {
	// Matchers returns a list of cmux matchers that will be used to determine
	// which requests should be handled by this branch.
	Matchers []cmux.Matcher
	// Policy determines how this branch should be served depending on the current
	// security configuration of the server.
	Policy RoutingPolicy
}

// Branch represents a sub-server of the main server, which process requests that
// match a specific pattern.
type Branch interface {
	// Key is a human-readable key that identifies this branch.
	Key() string
	// Routing returns the BranchRouting for this Branch.
	Routing() BranchRouting
	// Serve starts the branch using the provided ctx and should block until the branch
	// exits abnormally or is stopped by calling Stop.
	Serve(ctx BranchContext) error
	// Stop stops the branch gracefully.
	// (TODO: Evaluate whether we should pass a context here to allow for a timeout.)
	Stop()
}
