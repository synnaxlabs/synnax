package server

import (
	"context"
	"crypto/tls"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"net"
	"net/http"

	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type Config struct {
	ListenAddress address.Address
	Security      SecurityConfig
	Logger        *zap.Logger
	Branches      []Branch
}

type SecurityConfig struct {
	Insecure *bool
	TLS      *tls.Config
	CAName   string
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		Security: SecurityConfig{
			Insecure: config.BoolPointer(false),
		},
	}
)

func (c Config) Override(other Config) Config {
	c.ListenAddress = override.String(c.ListenAddress, other.ListenAddress)
	c.Security.Insecure = override.Nil(c.Security.Insecure, other.Security.Insecure)
	c.Security.TLS = override.Nil(c.Security.TLS, other.Security.TLS)
	c.Security.CAName = override.String(c.Security.CAName, other.Security.CAName)
	c.Logger = override.Nil(c.Logger, other.Logger)
	c.Branches = override.Slice(c.Branches, other.Branches)
	return c
}

func (c Config) Validate() error {
	v := validate.New("server")
	validate.NotEmptyString(v, "listenAddress", c.ListenAddress)
	validate.NotNil(v, "logger", c.Logger)
	validate.NotEmptyString(v, "security.caName", c.Security.CAName)
	return v.Error()
}

type Server struct {
	Config
	// closers are supplemental closers to be call on shutdown
	closers []func(ctx context.Context) error
}

func New(configs ...Config) (*Server, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Server{Config: cfg}, err
}

func (s *Server) Start(_ context.Context) (err error) {
	sCtx, cancel := signal.Background(signal.WithLogger(s.Logger))
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

func (s *Server) serveSecure(sCtx signal.Context, lis net.Listener) error {
	var (
		rootMux     = cmux.New(lis)
		insecureLis = rootMux.Match(cmux.HTTP1Fast())
		secureMux   = cmux.New(tls.NewListener(rootMux.Match(cmux.Any()), s.Security.TLS))
	)
	s.goRedirectInsecure(sCtx, insecureLis)
	s.startBranches(sCtx, secureMux)
	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(secureMux.Serve())
	}, signal.WithKey("secureMux"))
	return filterCloserError(rootMux.Serve())
}

func (s *Server) serveInsecure(sCtx signal.Context, lis net.Listener) error {
	rootMux := cmux.New(lis)
	s.startBranches(sCtx, rootMux)
	return filterCloserError(rootMux.Serve())
}

func (s *Server) startBranches(sCtx signal.Context, mux cmux.CMux) {
	listeners := make([]net.Listener, len(s.Branches))
	for i, b := range s.Branches {
		listeners[i] = mux.Match(b.Matchers()...)
	}
	bc := BranchConfig{Security: s.Security}
	for i, b := range s.Branches {
		b, i := b, i
		sCtx.Go(func(ctx context.Context) error {
			bc.Lis = listeners[i]
			return b.Serve(bc)
		}, signal.WithKey(b.Key()))
	}
}

func (s *Server) goRedirectInsecure(sCtx signal.Context, lis net.Listener) {
	srv := &http.Server{Handler: http.HandlerFunc(secureHTTPRedirect)}
	sCtx.Go(func(ctx context.Context) error {
		return filterCloserError(srv.Serve(lis))
	}, signal.WithKey("redirect"))
	s.closers = append(s.closers, srv.Shutdown)
}

func (s *Server) Stop() error {
	for _, b := range s.Branches {
		b.Stop()
	}
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, closer := range s.closers {
		c.ExecWithCtx(context.Background(), closer)
	}
	return c.Error()
}

func filterCloserError(err error) error {
	if errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func secureHTTPRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}
