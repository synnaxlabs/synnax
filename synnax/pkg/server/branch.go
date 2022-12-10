package server

import (
	"go.uber.org/zap"
	"net"

	"github.com/cockroachdb/cmux"
)

// BranchContext is the context for operating a Branch.
type BranchContext struct {
	// List is the listener the branch should listen for incoming requests on.
	Lis net.Listener
	// ServerName is the name of the Server this branch is running on.
	ServerName string
	// Security contains the security configuration for the Server.
	Security SecurityConfig
	// Debug is a flag to enable debugging endpoints and utilities.
	Debug bool
	// Logger is the witness of it all.
	Logger *zap.Logger
}

// BranchRouting is the information provided by a Branch to the Server so that it can
// appropriately route requests to it.
type BranchRouting struct {
	// Matchers returns a list of cmux matchers that will be used to determine
	// which requests should be handled by this branch.
	Matchers []cmux.Matcher
	// PreferSecure returns true if this branch should be run behind a TLS multiplexer when
	// running in secure mode.
	PreferSecure bool
	// ServeIfSecure returns true if this branch should be run even if the server is
	// running in secure mode.
	ServeIfSecure bool
	// ServeIfInsecure returns true if this branch should be run even if the server is
	// running in insecure mode.
	ServeIfInsecure bool
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
