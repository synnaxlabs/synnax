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
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the OIDC API service.
type ServiceConfig struct {
	// Auth checks that the Core accepts OIDC and ends the sessions links opened.
	//
	// [REQUIRED]
	Auth *svcauth.Service
	// OIDC stores provider links and checks proofs of provider accounts.
	//
	// [REQUIRED]
	OIDC *oidc.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the OIDC API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Auth = override.Nil(c.Auth, other.Auth)
	c.OIDC = override.Nil(c.OIDC, other.OIDC)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.auth.oidc")
	v.NotNil("auth", c.Auth)
	v.NotNil("oidc", c.OIDC)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service links, lists, and unlinks provider accounts.
type Service struct {
	auth *svcauth.Service
	oidc *oidc.Service
	rbac *rbac.Service
}

// NewService opens a new OIDC API service.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{auth: cfg.Auth, oidc: cfg.OIDC, rbac: cfg.RBAC}, nil
}

// LinkRequest links a provider account to the caller.
type LinkRequest struct {
	// Credentials prove the provider account.
	Credentials svcauth.OIDCCredentials `json:"credentials" msgpack:"credentials"`
}

// LinkResponse is a new provider link.
type LinkResponse struct {
	// Link is the stored link.
	Link oidc.Link `json:"link" msgpack:"link"`
}

// Link links the provider account that the credentials prove to the caller. Only a
// user can hold a link. It returns a validation error when the account or the caller
// already has a link for the provider.
func (s *Service) Link(
	ctx context.Context,
	tx gorp.Tx,
	req LinkRequest,
) (LinkResponse, error) {
	if err := s.auth.Accepts(svcauth.MethodOIDC); err != nil {
		return LinkResponse{}, err
	}
	caller := auth.GetSubject(ctx)
	if caller.Type != ontology.ResourceTypeUser {
		return LinkResponse{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot link a provider account",
			caller.Type,
		)
	}
	account, err := s.oidc.Verify(ctx, req.Credentials)
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
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// RetrieveResponse lists provider links.
type RetrieveResponse struct {
	// Links are the links that matched.
	Links []oidc.Link `json:"links" msgpack:"links"`
}

// Retrieve lists the caller's provider links, or the links of the given users when the
// caller may retrieve those users.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	caller := auth.GetSubject(ctx)
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []ontology.ID{caller}
	}
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		caller,
		access.ActionRetrieve,
		subjects,
	); err != nil {
		return RetrieveResponse{}, err
	}
	links := make([]oidc.Link, 0)
	if err := s.oidc.NewRetrieve().
		Where(oidc.MatchSubjects(subjects...)).
		Entries(&links).
		Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Links: links}, nil
}

// UnlinkRequest names provider links to remove.
type UnlinkRequest struct {
	// Keys are the links to remove.
	Keys []oidc.Key `json:"keys" msgpack:"keys"`
}

// Unlink removes the given provider links and ends the sessions they opened. A caller
// may remove its own links; removing another user's needs permission to delete that
// user. Unknown keys are ignored.
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
	subjects := lo.Uniq(lo.Map(links, func(l oidc.Link, _ int) ontology.ID {
		return l.Subject
	}))
	if err := auth.EnforceOthers(
		ctx,
		s.rbac,
		auth.GetSubject(ctx),
		access.ActionDelete,
		subjects,
	); err != nil {
		return struct{}{}, err
	}
	stored := lo.Map(links, func(l oidc.Link, _ int) oidc.Key { return l.Key })
	if err := s.oidc.NewWriter(tx).Delete(ctx, stored...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, nil, stored...)
}
