// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package oidc authenticates users through OpenID Connect providers. A user holds one
// [Link] per provider, and the package implements [auth.Authenticator] for
// [auth.MethodOIDC].
package oidc

import (
	"context"
	"maps"
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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Provider is an OpenID Connect provider that the Core accepts logins from.
type Provider struct {
	// Name identifies the provider in proofs and links.
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
	// Role assigns NewUserRole to a user created at first login.
	//
	// [REQUIRED]
	Role *role.Service
	// NewUserRole is the role a user created at first login receives.
	//
	// [REQUIRED]
	NewUserRole role.Key
	// Providers are the providers the Core accepts logins from.
	//
	// [OPTIONAL] - Defaults to none.
	Providers []Provider
	// DiscoveryTimeout bounds each read of a provider's discovery document and how
	// long [Service.Logins] waits for one.
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
	c.Role = override.Nil(c.Role, other.Role)
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
	v.NotNil("role", c.Role)
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
	table     *gorp.Table[Key, Link]
	indexes   indexes
	providers map[string]*provider
}

var _ auth.Authenticator = (*Service)(nil)

// provider reads its discovery document on first use, so a provider that is down when
// the Core starts does not stop the Core.
type provider struct {
	Provider
	timeout time.Duration
	// lock holds one discovery at a time; a waiter gives up when its context ends
	lock       chan struct{}
	discovered *gooidc.Provider
}

func (p *provider) discover(ctx context.Context) (*gooidc.Provider, error) {
	select {
	case p.lock <- struct{}{}:
	case <-ctx.Done():
		return nil, errors.Wrapf(ctx.Err(), "discover provider %q", p.Name)
	}
	defer func() { <-p.lock }()
	if p.discovered != nil {
		return p.discovered, nil
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	discovered, err := gooidc.NewProvider(ctx, p.Issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "discover provider %q", p.Name)
	}
	p.discovered = discovered
	return discovered, nil
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
	}
	for _, p := range cfg.Providers {
		s.providers[p.Name] = &provider{
			Provider: p,
			timeout:  cfg.DiscoveryTimeout,
			lock:     make(chan struct{}, 1),
		}
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Link]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases its resources.
func (s *Service) Close() error { return s.table.Close() }

// NewRetrieve opens a query for stored links.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), baseTX: s.cfg.DB, indexes: s.indexes}
}

// NewWriter opens a [Writer] that runs in tx.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{svc: s, tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

// Login is where a client sends a person to log in through a provider.
type Login struct {
	// Provider is the name of the provider.
	Provider string
	// AuthorizationEndpoint is the provider's login address.
	AuthorizationEndpoint string
	// ClientID is the Core's client ID at the provider.
	ClientID string
}

// Logins returns the login address of each provider, sorted by name. It reads the
// providers' discovery documents at the same time, and leaves out and logs a provider
// whose document it cannot read within the discovery timeout, so one provider being
// down does not hide or delay the others.
func (s *Service) Logins(ctx context.Context) []Login {
	names := slices.Sorted(maps.Keys(s.providers))
	logins := make([]*Login, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Go(func() {
			p := s.providers[name]
			// bounds the wait for a discovery another login already started
			dCtx, cancel := context.WithTimeout(ctx, s.cfg.DiscoveryTimeout)
			defer cancel()
			discovered, err := p.discover(dCtx)
			if err != nil {
				s.cfg.L.Warn("OIDC provider unavailable", zap.Error(err))
				return
			}
			logins[i] = &Login{
				Provider:              name,
				AuthorizationEndpoint: discovered.Endpoint().AuthURL,
				ClientID:              p.ClientID,
			}
		})
	}
	wg.Wait()
	return lo.FilterMap(logins, func(l *Login, _ int) (Login, bool) {
		if l == nil {
			return Login{}, false
		}
		return *l, true
	})
}

// Account is a provider account that proved itself at the provider.
type Account struct {
	// Provider is the name of the provider.
	Provider string
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
	oauth := oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     discovered.Endpoint(),
		RedirectURL:  creds.RedirectURI,
	}
	tok, err := oauth.Exchange(ctx, creds.Code, oauth2.VerifierOption(creds.Verifier))
	if err != nil {
		var rejected *oauth2.RetrieveError
		if errors.As(err, &rejected) {
			return Account{}, auth.ErrInvalidCredentials
		}
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
	return Account{Provider: p.Name, ID: idToken.Subject, Name: claims.Name}, nil
}

// Authenticate implements [auth.Authenticator]. The first login of an unlinked account
// creates a user with the new user role and links the account to it, unless the
// provider disables user creation, in which case it returns
// [auth.ErrInvalidCredentials].
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
			Where(MatchProviders(account.Provider)).
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
	return auth.Identity{Subject: link.Subject, Credential: link.Key}, nil
}

func (s *Service) createUser(
	ctx context.Context,
	tx gorp.Tx,
	account Account,
) (Link, error) {
	u, err := s.cfg.User.NewWriter(tx).Create(ctx, user.User{Name: account.Name})
	if err != nil {
		return Link{}, err
	}
	if err = s.cfg.Role.NewWriter(tx, false).AssignRole(
		ctx,
		u.OntologyID(),
		s.cfg.NewUserRole,
	); err != nil {
		return Link{}, err
	}
	return s.NewWriter(tx).Create(ctx, u.OntologyID(), account)
}

// DeleteFor implements [auth.Authenticator].
func (s *Service) DeleteFor(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...ontology.ID,
) error {
	return s.NewWriter(tx).deleteFor(ctx, subjects...)
}
