// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/cockroachdb/cmux"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for a Server.
type Config struct {
	alamos.Instrumentation
	// ListenAddress is the address the server will listen on. The server's name will be
	// set to the host portion of the address.
	ListenAddress address.Address
	// Security is the security configuration.
	Security SecurityConfig
	// Branches is a list of branches to serve.
	Branches []Branch
	// Debug is a flag to enable debugging endpoints and utilities.
	Debug *bool
}

// Report implements the alamos.ReportProvider interface.
func (c Config) Report() alamos.Report {
	base := c.Security.Report()
	base["listen_address"] = c.ListenAddress
	base["branches"] = branchKeys(c.Branches)
	base["debug"] = *c.Debug
	return base
}

// SecurityConfig is the security configuration for the server.
type SecurityConfig struct {
	// Insecure is a flag to indicate whether the server should run in insecure mode.
	// If so, the server will not use TLS and will not verify client certificates.
	// All secure Branches with Routing.ServeIfInsecure set to true will be served.
	// If false,  the server will use TLS and will verify client certificates.
	// All secure Branches with Routing.ServeIfSecure set to true will be served.
	Insecure *bool
	// TLS is the TLS configuration for the server.
	TLS *tls.Config
}

// Report implements the alamos.ReportProvider interface.
func (s SecurityConfig) Report() alamos.Report {
	return alamos.Report{"insecure": *s.Insecure}
}

var (
	_ alamos.ReportProvider = Config{}
	_ alamos.ReportProvider = SecurityConfig{}
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default server configuration.
	DefaultConfig = Config{
		Debug: config.Bool(false),
		Security: SecurityConfig{
			Insecure: config.Bool(false),
		},
	}
)

// Override implements the config.Properties interface.
func (c Config) Override(other Config) Config {
	c.ListenAddress = override.String(c.ListenAddress, other.ListenAddress)
	c.Security.Insecure = override.Nil(c.Security.Insecure, other.Security.Insecure)
	c.Security.TLS = override.Nil(c.Security.TLS, other.Security.TLS)
	c.Branches = override.Slice(c.Branches, other.Branches)
	c.Debug = override.Nil(c.Debug, other.Debug)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements the config.Properties interface.
func (c Config) Validate() error {
	v := validate.New("server")
	validate.NotEmptyString(v, "listenAddress", c.ListenAddress)
	return v.Error()
}

// Server is the main server for a Synnax node. It processes all incoming RPCs and API
// requests. A Server can be configured to multiplex multiple Branches on the same port.
// It can also serve secure branches behind a TLS listener.
type Server struct {
	Config
	wg      signal.WaitGroup
	started chan struct{}
}

// New creates a new server using the specified configuration. The server must be started
// using the Serve method. If the configuration is invalid, an error is returned.
func New(configs ...Config) (*Server, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	return &Server{Config: cfg, started: make(chan struct{})}, err
}

// Started is a channel that can be listened to for when the server has been successfully started.
func (s *Server) Started() <-chan struct{} { return s.started }

// Serve starts the server and blocks until all branches have stopped. Only returns an
// error if the server exits abnormally (i.e. it wil ignore any errors emitted during
// standard shutdown procedure).
func (s *Server) Serve() (err error) {
	s.L.Info("starting server", zap.Int("port", s.ListenAddress.Port()))
	s.L.Debug("config", s.Report().ZapFields()...)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.Instrumentation))
	s.wg = sCtx
	defer cancel()
	lis, err := net.Listen("tcp", s.ListenAddress.PortString())
	if err != nil {
		return err
	}
	if *s.Security.Insecure {
		return s.serveInsecure(sCtx, lis)
	}
	return s.serveSecure(sCtx, lis)
}

// Stop stops the server gracefully, waiting for all branches to stop serving requests.
// If the server exits abnormally, the error can be discovered through the return value
// if the Serve method.
func (s *Server) Stop() {
	for _, b := range s.Branches {
		b.Stop()
	}
	_ = s.wg.Wait()
}

func (s *Server) serveSecure(sCtx signal.Context, lis net.Listener) error {
	var (
		root     = cmux.New(lis)
		insecure = cmux.New(root.Match(cmux.HTTP1Fast()))
		secure   = cmux.New(tls.NewListener(root.Match(cmux.Any()), s.Security.TLS))
	)

	s.startBranches(sCtx, secure /*insecureMux*/, false)
	s.startBranches(sCtx, insecure /*insecureMux*/, true)

	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(secure.Serve())
	}, signal.WithKey("secure"), signal.RecoverWithErrOnPanic())

	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(insecure.Serve())
	}, signal.WithKey("insecureMux"), signal.RecoverWithErrOnPanic())

	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(root.Serve())
	}, signal.WithKey("rootMux"), signal.RecoverWithErrOnPanic())

	close(s.started)
	return sCtx.Wait()
}

func (s *Server) serveInsecure(sCtx signal.Context, lis net.Listener) error {
	mux := cmux.New(lis)
	s.startBranches(sCtx, mux /*insecureMux*/, true)
	close(s.started)
	return filterCloserError(mux.Serve())
}

func (s *Server) startBranches(
	sCtx signal.Context,
	mux cmux.CMux,
	insecureMux bool,
) {
	branches := lo.Filter(s.Branches, func(b Branch, _ int) bool {
		return b.Routing().Policy.ShouldServe(*s.Security.Insecure, insecureMux)
	})
	if len(branches) == 0 {
		return
	}

	s.L.Debug(
		"starting branches",
		zap.Strings("branches", branchKeys(branches)),
		zap.Bool("insecureMux", insecureMux),
	)

	listeners := make([]net.Listener, len(branches))
	for i, b := range branches {
		listeners[i] = mux.Match(b.Routing().Matchers...)
	}
	bc := s.baseBranchContext()
	for i, b := range branches {
		b, i := b, i
		// HERE
		sCtx.Go(func(context.Context) error {
			bc.Lis = listeners[i]
			return filterCloserError(b.Serve(bc))
		}, signal.WithKey(b.Key()))
	}
}

func (s *Server) baseBranchContext() BranchContext {
	return BranchContext{
		Debug:      *s.Debug,
		Security:   s.Security,
		ServerName: s.ListenAddress.HostString(),
	}
}

func filterCloserError(err error) error {
	if errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func branchKeys(branches []Branch) []string {
	keys := make([]string, len(branches))
	for i, b := range branches {
		keys[i] = b.Key()
	}
	return keys
}
