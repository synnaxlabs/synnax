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
	"net/http"
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
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
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
	// NewUserRole is the role a user created at first login receives.
	//
	// [REQUIRED]
	NewUserRole role.Key
	// Providers are the providers the Core accepts logins from.
	//
	// [OPTIONAL] - Defaults to none.
	Providers []Provider
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.User = override.Nil(c.User, other.User)
	c.NewUserRole = override.UUID(c.NewUserRole, other.NewUserRole)
	c.Providers = override.Slice(c.Providers, other.Providers)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("oidc")
	v.NotNil("db", c.DB)
	v.NotNil("user", c.User)
	v.Ternary("new_user_role", c.NewUserRole == uuid.Nil(), "must be set")
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
}

var _ auth.Authenticator = (*Service)(nil)

// requestTimeout bounds one request to a provider.
const requestTimeout = 10 * time.Second

// provider reads its discovery document on first use, so a provider that is down when
// the Core starts does not stop the Core. Concurrent logins share one read, and a
// failed read is retried by the next login.
type provider struct {
	Provider
	reads      singleflight.Group
	mu         sync.Mutex
	discovered *gooidc.Provider
}

func (p *provider) cached() *gooidc.Provider {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.discovered
}

func (p *provider) discover(ctx context.Context) (*gooidc.Provider, error) {
	if d := p.cached(); d != nil {
		return d, nil
	}
	read := p.reads.DoChan("", func() (any, error) {
		readCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			requestTimeout,
		)
		defer cancel()
		readCtx = gooidc.ClientContext(
			readCtx,
			&http.Client{Timeout: requestTimeout},
		)
		d, err := gooidc.NewProvider(readCtx, p.Issuer)
		if err != nil {
			return nil, errors.Wrapf(err, "discover provider %q", p.Name)
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		p.discovered = d
		return nil, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-read:
		if res.Err != nil {
			return nil, res.Err
		}
		return p.cached(), nil
	}
}

// OpenService opens a [Service] with the given configurations.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:       cfg,
		indexes:   newIndexes(),
		providers: make(map[string]*provider, len(cfg.Providers)),
	}
	for _, p := range cfg.Providers {
		s.providers[p.Name] = &provider{Provider: p}
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[LinkKey, Link]{
		DB:              cfg.DB,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Close closes the service and releases its resources.
func (s *Service) Close() error { return s.table.Close() }

func (s *Service) newRetrieve() Retrieve {
	return Retrieve{gorp: s.table.NewRetrieve(), indexes: s.indexes}
}

func (s *Service) newWriter(tx gorp.Tx) writer {
	return writer{svc: s, tx: tx}
}

// account is a provider account that proved itself at the provider.
type account struct {
	// Provider is the name of the provider.
	Provider string
	// Issuer is the issuer URL of the provider.
	Issuer string
	// ID is the provider's permanent ID for the account, its `sub` claim.
	ID string
	// Name is the person's display name, its `name` claim. It can be empty.
	Name string
}

// verify exchanges the authorization code in creds with its provider and returns the
// account that the provider's ID token names. It returns a validation error for an
// unknown provider and [auth.ErrInvalidCredentials] when the provider rejects the code
// or issues an ID token that fails verification. Every provider token is discarded.
func (s *Service) verify(
	ctx context.Context,
	creds auth.OIDCCredentials,
) (account, error) {
	if err := creds.Validate(); err != nil {
		return account{}, err
	}
	p, ok := s.providers[creds.Provider]
	if !ok {
		return account{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown provider %q",
			creds.Provider,
		)
	}
	discovered, err := p.discover(ctx)
	if err != nil {
		return account{}, err
	}
	oauth := oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     discovered.Endpoint(),
		RedirectURL:  creds.RedirectURI,
	}
	exchangeCtx := context.WithValue(
		ctx,
		oauth2.HTTPClient,
		&http.Client{Timeout: requestTimeout},
	)
	tok, err := oauth.Exchange(
		exchangeCtx,
		creds.Code,
		oauth2.VerifierOption(creds.Verifier),
	)
	if err != nil {
		var rejected *oauth2.RetrieveError
		if errors.As(err, &rejected) {
			return account{}, auth.ErrInvalidCredentials
		}
		return account{}, err
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return account{}, auth.ErrInvalidCredentials
	}
	idToken, err := discovered.Verifier(&gooidc.Config{ClientID: p.ClientID}).
		Verify(ctx, raw)
	if err != nil {
		return account{}, auth.ErrInvalidCredentials
	}
	var claims struct {
		Name string `json:"name"`
	}
	if err = idToken.Claims(&claims); err != nil {
		return account{}, auth.ErrInvalidCredentials
	}
	return account{
		Provider: p.Name,
		Issuer:   p.Issuer,
		ID:       idToken.Subject,
		Name:     claims.Name,
	}, nil
}

// Authenticate implements [auth.Authenticator]. The first login of an unlinked account
// creates a user with the new user role and links the account to it, unless the
// provider disables user creation, in which case it returns
// [auth.ErrInvalidCredentials]. An account whose link holds another issuer than the
// provider's also returns [auth.ErrInvalidCredentials].
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
	acc, err := s.verify(ctx, creds.OIDCCredentials)
	if err != nil {
		return auth.Identity{}, err
	}
	var link Link
	err = s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.newRetrieve().
			Where(MatchProviders(acc.Provider)).
			Where(MatchAccounts(acc.ID)).
			Entry(&link).
			Exec(ctx, tx)
		if err == nil && link.Issuer != acc.Issuer {
			return auth.ErrInvalidCredentials
		}
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		if s.providers[acc.Provider].UserCreationDisabled {
			return auth.ErrInvalidCredentials
		}
		link, err = s.createUser(ctx, tx, acc)
		return err
	})
	if err != nil {
		return auth.Identity{}, err
	}
	return auth.Identity{Subject: link.Subject, Credential: link.Key}, nil
}

// Version implements [auth.Authenticator]. A link keeps one version for life. A link
// whose provider or issuer is not in the config is not found, so its sessions end.
func (s *Service) Version(
	ctx context.Context,
	snap gorp.Snapshot,
	credential uuid.UUID,
) (telem.TimeStamp, error) {
	var link Link
	if err := s.newRetrieve().
		Where(MatchKeys(credential)).
		Entry(&link).
		Exec(ctx, snap); err != nil {
		return 0, err
	}
	if p, ok := s.providers[link.Provider]; !ok || p.Issuer != link.Issuer {
		return 0, errors.Wrapf(
			query.ErrNotFound,
			"provider %q with issuer %q is not in the config",
			link.Provider,
			link.Issuer,
		)
	}
	return 0, nil
}

func (s *Service) createUser(
	ctx context.Context,
	tx gorp.Tx,
	acc account,
) (Link, error) {
	// a provider need not send a name claim
	name := lo.Ternary(acc.Name != "", acc.Name, "Unnamed user")
	u, err := s.cfg.User.NewWriter(tx).Create(
		ctx,
		user.User{Name: name, Role: s.cfg.NewUserRole},
	)
	if err != nil {
		return Link{}, err
	}
	return s.newWriter(tx).create(ctx, u.ResourceID(), acc)
}
