package server

import (
	"context"
	"crypto/tls"
	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"net"
)

// Config is the configuration for a Server.
type Config struct {
	// ListenAddress is the address the server will listen on. The server's name will be
	// set to the host portion of the address.
	ListenAddress address.Address
	// Security is the security configuration.
	Security SecurityConfig
	// Branches is a list of branches to serve.
	Branches []Branch
	// Logger is the witness of it all.
	Logger *zap.Logger
	// Debug is a flag to enable debugging endpoints and utilities.
	Debug *bool
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

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default server configuration.
	DefaultConfig = Config{
		Debug: config.BoolPointer(false),
		Security: SecurityConfig{
			Insecure: config.BoolPointer(false),
		},
	}
)

// Override implements the config.Config interface.
func (c Config) Override(other Config) Config {
	c.ListenAddress = override.String(c.ListenAddress, other.ListenAddress)
	c.Security.Insecure = override.Nil(c.Security.Insecure, other.Security.Insecure)
	c.Security.TLS = override.Nil(c.Security.TLS, other.Security.TLS)
	c.Logger = override.Nil(c.Logger, other.Logger)
	c.Branches = override.Slice(c.Branches, other.Branches)
	c.Debug = override.Nil(c.Debug, other.Debug)
	return c
}

// Validate implements the config.Config interface.
func (c Config) Validate() error {
	v := validate.New("server")
	validate.NotEmptyString(v, "listenAddress", c.ListenAddress)
	validate.NotNil(v, "logger", c.Logger)
	return v.Error()
}

// Server is the main server for a Synnax node. It processes all incoming RPCs and API
// requests. A Server can be configured to multiplex multiple Branches on the same port.
// It can also serve secure branches behind a TLS listener.
type Server struct{ Config }

// New creates a new server using the specified configuration. The server must be started
// using the Serve method. If the configuration is invalid, an error is returned.
func New(configs ...Config) (*Server, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Server{Config: cfg}, err
}

// Serve starts the server and blocks until all branches have stopped. Only returns an
// error if the server exits abnormally (i.e. it wil ignore any errors emitted during
// standard shutdown procedure).
func (s *Server) Serve() (err error) {
	sCtx, cancel := signal.Background(signal.WithLogger(s.Logger), signal.WithContextKey("server"))
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

// Stop stops the server. Any errors encountered during shutdown are returned through
// the Serve method.
func (s *Server) Stop() {
	for _, b := range s.Branches {
		b.Stop()
	}
}

func (s *Server) serveSecure(sCtx signal.Context, lis net.Listener) error {
	var (
		root     = cmux.New(lis)
		insecure = cmux.New(root.Match(cmux.HTTP1Fast()))
		secure   = cmux.New(tls.NewListener(root.Match(cmux.Any()), s.Security.TLS))
	)

	s.startBranches(sCtx, secure, lo.Filter(s.Branches, func(b Branch, _ int) bool {
		return b.Routing().PreferSecure
	}))

	s.startBranches(sCtx, insecure, lo.Filter(s.Branches, func(b Branch, _ int) bool {
		info := b.Routing()
		return !info.PreferSecure && info.ServeIfSecure
	}))

	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(secure.Serve())
	}, signal.WithKey("secure"))

	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(insecure.Serve())
	}, signal.WithKey("insecureMux"))

	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(root.Serve())
	}, signal.WithKey("rootMux"))

	return sCtx.Wait()
}

func (s *Server) serveInsecure(sCtx signal.Context, lis net.Listener) error {
	mux := cmux.New(lis)
	s.startBranches(sCtx, mux, lo.Filter(s.Branches, func(b Branch, _ int) bool {
		return b.Routing().ServeIfInsecure
	}))
	return filterCloserError(mux.Serve())
}

func (s *Server) startBranches(
	sCtx signal.Context,
	mux cmux.CMux,
	branches []Branch,
) {
	listeners := make([]net.Listener, len(branches))
	for i, b := range branches {
		listeners[i] = mux.Match(b.Routing().Matchers...)
	}
	bc := BranchContext{Security: s.Security, Debug: *s.Debug}
	for i, b := range branches {
		b, i := b, i
		bc.Lis = listeners[i]
		sCtx.Go(func(context.Context) error {
			return filterCloserError(b.Serve(bc))
		}, signal.WithKey(b.Key()))
	}
}

func (s *Server) baseBranchContext() BranchContext {
	return BranchContext{
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
