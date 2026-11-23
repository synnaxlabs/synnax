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
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
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

// PrepareLinkOIDC verifies the provider account that the credentials of req prove. It
// returns a validation error when the oidc method is off or the caller is not a user,
// and [auth.ErrInvalidCredentials] when the provider rejects the credentials.
func (s *Service) PrepareLinkOIDC(
	ctx context.Context,
	req OIDCLinkRequest,
) (oidc.Account, error) {
	if err := s.auth.Accepts(auth.MethodOIDC); err != nil {
		return oidc.Account{}, err
	}
	if caller := GetSubject(ctx); caller.Type != resource.TypeUser {
		return oidc.Account{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot link a provider account",
			caller.Type,
		)
	}
	return s.oidc.Verify(ctx, req.Credentials)
}

// LinkOIDC links the account that PrepareLinkOIDC verified to the caller. It returns a
// validation error when the account or the caller already has a link for the provider.
func (s *Service) LinkOIDC(
	ctx context.Context,
	tx gorp.Tx,
	_ OIDCLinkRequest,
	account oidc.Account,
) (OIDCLinkResponse, error) {
	caller, err := user.KeyFromResourceID(GetSubject(ctx))
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
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
}

// Holders implements bind.Held.
func (r OIDCRetrieveRequest) Holders() []resource.ID { return r.Subjects }

// OIDCRetrieveResponse lists provider links.
type OIDCRetrieveResponse struct {
	// Links are the links that matched.
	Links []oidc.Link `json:"links" msgpack:"links"`
}

// RetrieveOIDCLinks lists the provider links of the given users in snap, or the
// caller's links when the request names no user.
func (s *Service) RetrieveOIDCLinks(
	ctx context.Context,
	snap gorp.Snapshot,
	req OIDCRetrieveRequest,
) (OIDCRetrieveResponse, error) {
	subjects := req.Subjects
	if len(subjects) == 0 {
		subjects = []resource.ID{GetSubject(ctx)}
	}
	keys := make([]user.Key, 0, len(subjects))
	for _, subject := range subjects {
		if subject.Type != resource.TypeUser {
			continue
		}
		key, err := user.KeyFromResourceID(subject)
		if err != nil {
			return OIDCRetrieveResponse{}, err
		}
		keys = append(keys, key)
	}
	links := make([]oidc.Link, 0)
	if err := s.oidc.NewRetrieve().
		Where(oidc.MatchUsers(keys...)).
		Entries(&links).
		Exec(ctx, snap); err != nil {
		return OIDCRetrieveResponse{}, err
	}
	return OIDCRetrieveResponse{Links: links}, nil
}

// OIDCUnlinkRequest names provider links to remove.
type OIDCUnlinkRequest struct {
	// Keys are the links to remove.
	Keys []oidc.LinkKey `json:"keys" msgpack:"keys"`
}

// UnlinkOIDC removes the given provider links and ends the sessions they opened.
// Unknown keys are ignored.
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
	stored := lo.Map(links, func(l oidc.Link, _ int) oidc.LinkKey { return l.Key })
	if err := s.oidc.NewWriter(tx).Delete(ctx, stored...); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.auth.NewWriter(tx).RevokeCredential(ctx, stored...)
}
