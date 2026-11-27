// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oidc

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/invite"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the OIDC API service.
type ServiceConfig struct {
	// Auth reports whether the Core accepts the oidc method and ends the sessions of
	// removed links.
	//
	// [REQUIRED]
	Auth *auth.Service
	// OIDC stores provider links and checks proofs of provider accounts.
	//
	// [REQUIRED]
	OIDC *oidc.Service
	// Invite stores one-time invites.
	//
	// [REQUIRED]
	Invite *invite.Service
	// Session builds the response of the session that a redeemed invite opens.
	//
	// [REQUIRED]
	Session *session.Service
	// Password stores the password that an invited user sets.
	//
	// [REQUIRED]
	Password *password.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.OIDC = override.Nil(c.OIDC, other.OIDC)
	c.Invite = override.Nil(c.Invite, other.Invite)
	c.Session = override.Nil(c.Session, other.Session)
	c.Password = override.Nil(c.Password, other.Password)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.oidc")
	v.NotNil("auth", c.Auth)
	v.NotNil("oidc", c.OIDC)
	v.NotNil("invite", c.Invite)
	v.NotNil("session", c.Session)
	v.NotNil("password", c.Password)
	return v.Error()
}

// Service links, lists, and unlinks OpenID Connect provider accounts, and manages
// invites.
type Service struct {
	auth     *auth.Service
	oidc     *oidc.Service
	invite   *invite.Service
	session  *session.Service
	password *password.Service
}

// NewService opens a new OIDC API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		auth:     cfg.Auth,
		oidc:     cfg.OIDC,
		invite:   cfg.Invite,
		session:  cfg.Session,
		password: cfg.Password,
	}, nil
}

// LinkRequest links a provider account to the caller.
type LinkRequest struct {
	// Credentials prove the provider account.
	Credentials auth.OIDCCredentials `json:"credentials" msgpack:"credentials"`
}

// LinkResponse is a new provider link.
type LinkResponse struct {
	// Link is the stored link.
	Link oidc.Link `json:"link" msgpack:"link"`
}

// PrepareLink verifies the provider account that the credentials of req prove. It
// returns a validation error when the oidc method is off or the caller is not a user,
// and [auth.ErrInvalidCredentials] when the provider rejects the credentials.
func (s *Service) PrepareLink(
	ctx context.Context,
	req LinkRequest,
) (oidc.Account, error) {
	if err := s.auth.Accepts(auth.MethodOIDC); err != nil {
		return oidc.Account{}, err
	}
	if caller := apiauth.GetSubject(ctx); caller.Type != resource.TypeUser {
		return oidc.Account{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot link a provider account",
			caller.Type,
		)
	}
	return s.oidc.Verify(ctx, req.Credentials)
}

// Link links the account that PrepareLink verified to the caller. It returns a
// validation error when the account or the caller already has a link for the provider.
func (s *Service) Link(
	ctx context.Context,
	tx gorp.Tx,
	_ LinkRequest,
	account oidc.Account,
) (LinkResponse, error) {
	caller, err := user.KeyFromResourceID(apiauth.GetSubject(ctx))
	if err != nil {
		return LinkResponse{}, err
	}
	link, err := s.oidc.NewWriter(tx).Create(ctx, caller, account)
	if err != nil {
		return LinkResponse{}, err
	}
	return LinkResponse{Link: link}, nil
}

// RetrieveRequest selects provider links to list.
type RetrieveRequest struct {
	// Subjects limits the result to these users' links. Empty means the caller.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
}

// Holders implements bind.Held.
func (r RetrieveRequest) Holders() []resource.ID { return r.Subjects }

// RetrieveResponse lists provider links.
type RetrieveResponse struct {
	// Links are the links that matched.
	Links []oidc.Link `json:"links" msgpack:"links"`
}

// Retrieve lists the provider links of the given users in snap, or the caller's links
// when the request names no user.
func (s *Service) Retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	keys, err := userKeys(ctx, req.Subjects)
	if err != nil {
		return RetrieveResponse{}, err
	}
	links := make([]oidc.Link, 0)
	if err := s.oidc.NewRetrieve().
		Where(oidc.MatchUsers(keys...)).
		Entries(&links).
		Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Links: links}, nil
}

// UnlinkRequest names provider links to remove.
type UnlinkRequest struct {
	// Keys are the links to remove.
	Keys []oidc.LinkKey `json:"keys" msgpack:"keys"`
}

// Unlink removes the given provider links and ends the sessions they opened. Unknown
// keys are ignored.
func (s *Service) Unlink(
	ctx context.Context,
	tx gorp.Tx,
	req UnlinkRequest,
) (struct{}, error) {
	var links []oidc.Link
	if err := s.oidc.NewRetrieve().
		Where(oidc.MatchKeys(req.Keys...)).
		Entries(&links).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	stored := lo.Map(links, func(l oidc.Link, _ int) oidc.LinkKey { return l.Key })
	if err := s.oidc.NewWriter(tx).Delete(ctx, stored...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, stored...)
}

// userKeys returns the keys of the users in subjects, or the caller's key when subjects
// is empty.
func userKeys(ctx context.Context, subjects []resource.ID) ([]user.Key, error) {
	if len(subjects) == 0 {
		subjects = []resource.ID{apiauth.GetSubject(ctx)}
	}
	keys := make([]user.Key, 0, len(subjects))
	for _, subject := range subjects {
		if subject.Type != resource.TypeUser {
			continue
		}
		key, err := user.KeyFromResourceID(subject)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}
