// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package oidc authenticates users through OpenID Connect providers. A user holds one
// [Link] per issuer, and the package implements [auth.Authenticator] for
// [auth.MethodOIDC].
package oidc

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"
	"uuid"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Provider is an OpenID Connect provider that the Core accepts logins from.
type Provider struct {
	// Name identifies the provider in proofs.
	Name string
	// Issuer is the provider's issuer URL. The Core reads the provider's endpoints from
	// its discovery document.
	Issuer string
	// ClientID is the Core's client ID at the provider.
	ClientID string
	// ClientSecret authenticates the Core at the provider's token endpoint. It is empty
	// for a public client.
	ClientSecret string
	// UserCreationDisabled stops the first login of an unlinked account from creating
	// a user. Such an account then logs in only after its user links it.
	UserCreationDisabled bool
}

// ServiceConfig is the configuration for opening a [Service].
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, metrics, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// DB stores links.
	//
	// [REQUIRED]
	DB *gorp.DB
	// User creates the user of an account's first login.
	//
	// [REQUIRED]
	User *user.Service
	// NewUserRole is the role a user created at first login receives.
	//
	// [REQUIRED]
	NewUserRole role.Key
	// Providers are the providers the Core accepts logins from.
	//
	// [OPTIONAL] - Defaults to none.
	Providers []Provider
	// DiscoveryTimeout bounds each read of a provider's discovery document.
	//
	// [OPTIONAL] - Defaults to 5 seconds.
	DiscoveryTimeout time.Duration
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.User = override.Nil(c.User, other.User)
	c.NewUserRole = override.UUID(c.NewUserRole, other.NewUserRole)
	c.Providers = override.Slice(c.Providers, other.Providers)
	c.DiscoveryTimeout = override.Numeric(c.DiscoveryTimeout, other.DiscoveryTimeout)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("oidc")
	v.NotNil("db", c.DB)
	v.NotNil("user", c.User)
	v.Ternary("new_user_role", c.NewUserRole == uuid.Nil(), "must be set")
	v.Positive("discovery_timeout", c.DiscoveryTimeout)
	names := make(set.Set[string], len(c.Providers))
	for _, p := range c.Providers {
		v.Ternary("providers", p.Name == "", "name must be set")
		v.Ternaryf("providers", names.Contains(p.Name), "duplicate name %q", p.Name)
		v.Ternaryf("providers", p.Issuer == "", "%q needs an issuer", p.Name)
		v.Ternaryf("providers", p.ClientID == "", "%q needs a client ID", p.Name)
		names.Add(p.Name)
	}
	return v.Error()
}

// Service stores links and checks proofs of provider accounts.
type Service struct {
	cfg       ServiceConfig
	table     *gorp.Table[LinkKey, Link]
	indexes   indexes
	providers map[string]*provider
	issuers   set.Set[string]
	shutdown  context.CancelFunc
	reads     sync.WaitGroup
	// transport carries the discovery reads, so Close can drop their connections.
	transport *http.Transport
}

var _ auth.Authenticator = (*Service)(nil)

// requestTimeout bounds one request to a provider.
const requestTimeout = 10 * time.Second

// retryAfter is how long a provider whose discovery failed answers with that failure
// before the next read.
const retryAfter = 30 * time.Second

// provider reads its discovery document in the background, so a provider that is down
// when the Core starts does not stop the Core. Concurrent reads share one request, and
// a failed read is kept for retryAfter.
type provider struct {
	Provider
	timeout   time.Duration
	transport http.RoundTripper
	// ctx ends the reads when the service closes.
	ctx context.Context
	L   *alamos.Logger
	wg  *sync.WaitGroup
	mu  struct {
		sync.Mutex
		discovered *gooidc.Provider
		err        error
		failed     time.Time
		// reading closes when the read in flight ends. It is nil when none is.
		reading chan struct{}
	}
}

// state returns the discovered provider, or the error of a read that failed less than
// retryAfter ago. Both are nil when the provider needs a new read.
func (p *provider) state() (*gooidc.Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.mu.err != nil && time.Since(p.mu.failed) >= retryAfter {
		return nil, nil
	}
	return p.mu.discovered, p.mu.err
}

// read starts a read of the discovery document, or joins the one in flight. The
// returned channel closes when the read ends.
func (p *provider) read() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.mu.reading != nil {
		return p.mu.reading
	}
	reading := make(chan struct{})
	p.mu.reading = reading
	p.wg.Go(func() {
		ctx, cancel := context.WithTimeout(p.ctx, p.timeout)
		defer cancel()
		ctx = gooidc.ClientContext(
			ctx,
			&http.Client{Timeout: p.timeout, Transport: p.transport},
		)
		d, err := gooidc.NewProvider(ctx, p.Issuer)
		p.mu.Lock()
		defer p.mu.Unlock()
		defer close(reading)
		p.mu.reading = nil
		if err != nil {
			p.mu.err = errors.Wrapf(err, "discover provider %q", p.Name)
			p.mu.failed = time.Now()
			p.L.Warn("OIDC provider unavailable", zap.Error(p.mu.err))
			return
		}
		p.mu.discovered, p.mu.err = d, nil
	})
	return reading
}

// discover returns the discovered provider, waiting for a read when it has none. It
// returns the error of a read that failed less than retryAfter ago without a new read.
func (p *provider) discover(ctx context.Context) (*gooidc.Provider, error) {
	if d, err := p.state(); d != nil || err != nil {
		return d, err
	}
	select {
	case <-ctx.Done():
		return nil, errors.Wrapf(ctx.Err(), "discover provider %q", p.Name)
	case <-p.read():
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.mu.discovered, p.mu.err
	}
}

// OpenService opens a [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{DiscoveryTimeout: 5 * time.Second}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:       cfg,
		indexes:   newIndexes(),
		providers: make(map[string]*provider, len(cfg.Providers)),
		issuers:   make(set.Set[string], len(cfg.Providers)),
		transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[LinkKey, Link]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	readCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.shutdown = cancel
	for _, p := range cfg.Providers {
		s.issuers.Add(p.Issuer)
		pr := &provider{
			Provider:  p,
			timeout:   cfg.DiscoveryTimeout,
			transport: s.transport,
			ctx:       readCtx,
			L:         cfg.L,
			wg:        &s.reads,
		}
		s.providers[p.Name] = pr
		pr.read()
	}
	return s, nil
}

// Close stops the discovery reads in flight and closes the service.
func (s *Service) Close() error {
	s.shutdown()
	s.reads.Wait()
	s.transport.CloseIdleConnections()
	return s.table.Close()
}

// NewRetrieve opens a query for stored links.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: tx}
}

// Login is where a client sends a person to log in through a provider.
type Login struct {
	// Provider is the name of the provider.
	Provider string
	// Issuer is the provider's issuer URL, which the links of its accounts hold.
	Issuer string
	// AuthorizationEndpoint is the provider's login address.
	AuthorizationEndpoint string
	// ClientID is the Core's client ID at the provider.
	ClientID string
}

// Logins returns the login address of each provider whose discovery document the
// Core has read, sorted by name. It never waits on a provider: it leaves out each
// provider whose document is not read yet or failed to read, and starts a new read for
// it unless one failed less than a retry interval ago.
func (s *Service) Logins() []Login {
	logins := make([]Login, 0, len(s.providers))
	for _, name := range slices.Sorted(maps.Keys(s.providers)) {
		p := s.providers[name]
		d, err := p.state()
		if d == nil {
			if err == nil {
				p.read()
			}
			continue
		}
		logins = append(logins, Login{
			Provider:              name,
			Issuer:                p.Issuer,
			AuthorizationEndpoint: d.Endpoint().AuthURL,
			ClientID:              p.ClientID,
		})
	}
	return logins
}

// Account is a provider account that proved itself at the provider.
type Account struct {
	// Provider is the name of the provider.
	Provider string
	// Issuer is the issuer URL of the provider.
	Issuer string
	// ID is the provider's permanent ID for the account, its `sub` claim.
	ID string
	// Name is the person's display name, its `name` claim. It can be empty.
	Name string
}

// Verify exchanges the authorization code in creds with its provider and returns the
// account that the provider's ID token names. It returns a validation error for an
// unknown provider and [auth.ErrInvalidCredentials] when the provider rejects the code
// or issues an ID token that fails verification. Every provider token is discarded.
func (s *Service) Verify(
	ctx context.Context,
	creds auth.OIDCCredentials,
) (Account, error) {
	if err := creds.Validate(); err != nil {
		return Account{}, err
	}
	p, ok := s.providers[creds.Provider]
	if !ok {
		return Account{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown provider %q",
			creds.Provider,
		)
	}
	discovered, err := p.discover(ctx)
	if err != nil {
		return Account{}, err
	}
	tok, err := exchange(ctx, p, discovered, creds)
	if err != nil {
		return Account{}, err
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return Account{}, auth.ErrInvalidCredentials
	}
	idToken, err := discovered.Verifier(&gooidc.Config{ClientID: p.ClientID}).
		Verify(ctx, raw)
	if err != nil {
		return Account{}, auth.ErrInvalidCredentials
	}
	var claims struct {
		Name string `json:"name"`
	}
	if err = idToken.Claims(&claims); err != nil {
		return Account{}, auth.ErrInvalidCredentials
	}
	return Account{
		Provider: p.Name,
		Issuer:   p.Issuer,
		ID:       idToken.Subject,
		Name:     claims.Name,
	}, nil
}

// exchange trades the authorization code in creds for the provider's tokens. It returns
// [auth.ErrInvalidCredentials] when the provider rejects the code.
func exchange(
	ctx context.Context,
	p *provider,
	discovered *gooidc.Provider,
	creds auth.OIDCCredentials,
) (*oauth2.Token, error) {
	oauth := oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     discovered.Endpoint(),
		RedirectURL:  creds.RedirectURI,
	}
	ctx = context.WithValue(
		ctx,
		oauth2.HTTPClient,
		&http.Client{Timeout: requestTimeout},
	)
	tok, err := oauth.Exchange(ctx, creds.Code, oauth2.VerifierOption(creds.Verifier))
	var rejected *oauth2.RetrieveError
	if errors.As(err, &rejected) {
		return nil, auth.ErrInvalidCredentials
	}
	return tok, err
}

// Authenticate implements [auth.Authenticator]. The first login of an unlinked account
// creates a user with the new user role and links the account to it, unless the
// provider disables user creation, in which case it returns
// [auth.ErrInvalidCredentials]. A link is found by the issuer and the account, so
// renaming a provider in the config keeps its links.
func (s *Service) Authenticate(
	ctx context.Context,
	proof auth.Proof,
) (auth.Identity, error) {
	creds, ok := proof.Variant.(auth.OIDCProof)
	if !ok {
		return auth.Identity{}, errors.Wrapf(
			validate.ErrValidation,
			"OIDC authenticator cannot check a %T proof",
			proof.Variant,
		)
	}
	account, err := s.Verify(ctx, creds.OIDCCredentials)
	if err != nil {
		return auth.Identity{}, err
	}
	var link Link
	err = s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.NewRetrieve().
			Where(MatchIssuers(account.Issuer)).
			Where(MatchAccounts(account.ID)).
			Entry(&link).
			Exec(ctx, tx)
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		if s.providers[account.Provider].UserCreationDisabled {
			return auth.ErrInvalidCredentials
		}
		link, err = s.createUser(ctx, tx, account)
		return err
	})
	if err != nil {
		return auth.Identity{}, err
	}
	return auth.Identity{
		Subject:    user.ResourceID(link.User),
		Credential: link.Key,
	}, nil
}

// Version implements [auth.Authenticator]. A link keeps one version for life. A link
// whose issuer no provider in the config has is not found, so its sessions end.
func (s *Service) Version(
	ctx context.Context,
	snap gorp.Snapshot,
	credential uuid.UUID,
) (telem.TimeStamp, error) {
	var link Link
	if err := s.NewRetrieve().
		Where(MatchKeys(credential)).
		Entry(&link).
		Exec(ctx, snap); err != nil {
		return 0, err
	}
	if !s.issuers.Contains(link.Issuer) {
		return 0, errors.Wrapf(
			query.ErrNotFound,
			"no provider with issuer %q is in the config",
			link.Issuer,
		)
	}
	return 0, nil
}

func (s *Service) createUser(
	ctx context.Context,
	tx gorp.Tx,
	account Account,
) (Link, error) {
	// a provider need not send a name claim
	name := lo.Ternary(account.Name != "", account.Name, "Unnamed user")
	u, err := s.cfg.User.NewWriter(tx).Create(
		ctx,
		user.User{Name: name, Role: s.cfg.NewUserRole},
	)
	if err != nil {
		return Link{}, err
	}
	return s.NewWriter(tx).Create(ctx, u.Key, account)
}
