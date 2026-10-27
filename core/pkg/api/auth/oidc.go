// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// OIDCLogin is where a client sends a person to log in through a provider.
type OIDCLogin struct {
	// Provider is the name of the provider.
	Provider string `json:"provider" msgpack:"provider"`
	// AuthorizationEndpoint is the provider's login address.
	AuthorizationEndpoint string `json:"authorization_endpoint" msgpack:"authorization_endpoint"`
	// ClientID is the Core's client ID at the provider.
	ClientID string `json:"client_id" msgpack:"client_id"`
}

// OIDCLinkRequest links a provider account to the caller.
type OIDCLinkRequest struct {
	// Credentials prove the provider account.
	Credentials auth.OIDCCredentials `json:"credentials" msgpack:"credentials"`
}

// OIDCLinkResponse is a new provider link.
type OIDCLinkResponse struct {
	// Link is the stored link.
	Link oidc.Link `json:"link" msgpack:"link"`
}

// LinkOIDC links the provider account that the credentials prove to the caller. Only a
// user can hold a link. It returns a validation error when the account or the caller
// already has a link for the provider.
func (s *Service) LinkOIDC(
	ctx context.Context,
	tx gorp.Tx,
	req OIDCLinkRequest,
) (OIDCLinkResponse, error) {
	if err := s.auth.Accepts(auth.MethodOidc); err != nil {
		return OIDCLinkResponse{}, err
	}
	caller := GetSubject(ctx)
	if caller.Type != ontology.ResourceTypeUser {
		return OIDCLinkResponse{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot link a provider account",
			caller.Type,
		)
	}
	account, err := s.oidc.Verify(ctx, req.Credentials)
	if err != nil {
		return OIDCLinkResponse{}, err
	}
	link, err := s.oidc.NewWriter(tx).Create(ctx, caller, account)
	if err != nil {
		return OIDCLinkResponse{}, err
	}
	return OIDCLinkResponse{Link: link}, nil
}

// OIDCRetrieveRequest selects provider links to list.
type OIDCRetrieveRequest struct {
	// Subjects limits the result to these users' links. Empty means the caller.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// OIDCRetrieveResponse lists provider links.
type OIDCRetrieveResponse struct {
	// Links are the links that matched.
	Links []oidc.Link `json:"links" msgpack:"links"`
}

// RetrieveOIDCLinks lists the caller's provider links, or the links of the given users
// when the caller may retrieve those users.
func (s *Service) RetrieveOIDCLinks(
	ctx context.Context,
	req OIDCRetrieveRequest,
) (OIDCRetrieveResponse, error) {
	caller := GetSubject(ctx)
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []ontology.ID{caller}
	}
	if err := s.enforceOthers(
		ctx,
		caller,
		access.ActionRetrieve,
		subjects,
	); err != nil {
		return OIDCRetrieveResponse{}, err
	}
	links := make([]oidc.Link, 0)
	if err := s.oidc.NewRetrieve().
		Where(oidc.MatchSubjects(subjects...)).
		Entries(&links).
		Exec(ctx, nil); err != nil {
		return OIDCRetrieveResponse{}, err
	}
	return OIDCRetrieveResponse{Links: links}, nil
}

// OIDCUnlinkRequest names provider links to remove.
type OIDCUnlinkRequest struct {
	// Keys are the links to remove.
	Keys []oidc.Key `json:"keys" msgpack:"keys"`
}

// UnlinkOIDC removes the given provider links and ends the sessions they opened. A
// caller may remove its own links; removing another user's needs permission to delete
// that user. Unknown keys are ignored.
func (s *Service) UnlinkOIDC(
	ctx context.Context,
	tx gorp.Tx,
	req OIDCUnlinkRequest,
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
	if err := s.enforceOthers(
		ctx,
		GetSubject(ctx),
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
