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
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/invite"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// CreateInviteRequest invites a user.
type CreateInviteRequest struct {
	// Subject is the user to invite.
	Subject ontology.ID `json:"subject" msgpack:"subject"`
}

// CreateInviteResponse is a new invite.
type CreateInviteResponse struct {
	// Token redeems the invite. It is the only copy of its secret.
	Token string `json:"token" msgpack:"token"`
	// Invite is the stored invite.
	Invite invite.Invite `json:"invite" msgpack:"invite"`
}

// CreateInvite replaces the invite of a user the caller may update. It returns a
// validation error when the oidc method is off or the subject is not a user, and an
// error wrapping query.ErrNotFound when the user does not exist.
func (s *Service) CreateInvite(
	ctx context.Context,
	req CreateInviteRequest,
) (CreateInviteResponse, error) {
	if err := s.auth.Accepts(svcauth.MethodOIDC); err != nil {
		return CreateInviteResponse{}, err
	}
	if req.Subject.Type != ontology.ResourceTypeUser {
		return CreateInviteResponse{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot be invited",
			req.Subject.Type,
		)
	}
	if err := s.enforce(
		ctx,
		nil,
		access.ActionUpdate,
		[]ontology.ID{req.Subject},
	); err != nil {
		return CreateInviteResponse{}, err
	}
	if _, err := s.retrieveUser(ctx, nil, req.Subject); err != nil {
		return CreateInviteResponse{}, err
	}
	i, token, err := s.invite.Create(ctx, req.Subject)
	if err != nil {
		return CreateInviteResponse{}, err
	}
	return CreateInviteResponse{Token: token, Invite: i}, nil
}

// RetrieveInvitesRequest selects invites to list.
type RetrieveInvitesRequest struct {
	// Subjects are the users whose invites to list.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// RetrieveInvitesResponse lists invites.
type RetrieveInvitesResponse struct {
	// Invites are the live invites that matched.
	Invites []invite.Invite `json:"invites" msgpack:"invites"`
}

// RetrieveInvites lists the live invites of users the caller may retrieve. It returns
// a validation error when the oidc method is off.
func (s *Service) RetrieveInvites(
	ctx context.Context,
	req RetrieveInvitesRequest,
) (RetrieveInvitesResponse, error) {
	if err := s.auth.Accepts(svcauth.MethodOIDC); err != nil {
		return RetrieveInvitesResponse{}, err
	}
	if err := s.enforce(ctx, nil, access.ActionRetrieve, req.Subjects); err != nil {
		return RetrieveInvitesResponse{}, err
	}
	invites := make([]invite.Invite, 0)
	if err := s.invite.NewRetrieve().
		Where(invite.MatchSubjects(req.Subjects...)).
		Where(s.invite.MatchLive()).
		Entries(&invites).
		Exec(ctx, nil); err != nil {
		return RetrieveInvitesResponse{}, err
	}
	return RetrieveInvitesResponse{Invites: invites}, nil
}

// DeleteInvitesRequest names invites to revoke.
type DeleteInvitesRequest struct {
	// Keys are the invites to revoke.
	Keys []invite.Key `json:"keys" msgpack:"keys"`
}

// DeleteInvites revokes invites of users the caller may update. Unknown keys are
// ignored. It returns a validation error when the oidc method is off.
func (s *Service) DeleteInvites(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteInvitesRequest,
) (struct{}, error) {
	if err := s.auth.Accepts(svcauth.MethodOIDC); err != nil {
		return struct{}{}, err
	}
	var invites []invite.Invite
	if err := s.invite.NewRetrieve().
		Where(invite.MatchKeys(req.Keys...)).
		Entries(&invites).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	subjects := lo.Uniq(lo.Map(invites, func(i invite.Invite, _ int) ontology.ID {
		return i.Subject
	}))
	if err := s.enforce(ctx, tx, access.ActionUpdate, subjects); err != nil {
		return struct{}{}, err
	}
	stored := lo.Map(invites, func(i invite.Invite, _ int) invite.Key { return i.Key })
	return struct{}{}, s.invite.NewWriter(tx).Delete(ctx, stored...)
}

// RedeemInviteRequest links a provider account through an invite.
type RedeemInviteRequest struct {
	// Token is the invite's token.
	Token string `json:"token" msgpack:"token"`
	// Credentials prove the provider account.
	Credentials svcauth.OIDCCredentials `json:"credentials" msgpack:"credentials"`
	// Cookie asks for the session token in a cookie, as session.CreateRequest does.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// RedeemInvite links the provider account that the credentials prove to the invited
// user, spends the invite, and opens a session as a login does. It needs no session.
// It returns [svcauth.ErrInvalidCredentials] for an unknown, used, or expired token,
// and a validation error when the oidc method is off, the account is already linked,
// or the user already has a link for the provider. A failed redeem keeps the invite.
func (s *Service) RedeemInvite(
	ctx context.Context,
	req RedeemInviteRequest,
) (session.CreateResponse, error) {
	if err := s.auth.Accepts(svcauth.MethodOIDC); err != nil {
		return session.CreateResponse{}, err
	}
	account, err := s.oidc.Verify(ctx, req.Credentials)
	if err != nil {
		return session.CreateResponse{}, err
	}
	var (
		res   session.CreateResponse
		token string
	)
	if err = s.invite.Redeem(ctx, req.Token, func(tx gorp.Tx, i invite.Invite) error {
		link, err := s.oidc.NewWriter(tx).Create(ctx, i.Subject, account)
		if err != nil {
			return err
		}
		if res.User, err = s.retrieveUser(ctx, tx, i.Subject); err != nil {
			return err
		}
		res.Session, token, err = s.auth.NewWriter(tx).Open(ctx, i.Subject, link.Key)
		return err
	}); err != nil {
		return session.CreateResponse{}, err
	}
	res.Token = auth.Deliver(ctx, s.auth, token, req.Cookie)
	return res, nil
}

func (s *Service) enforce(
	ctx context.Context,
	tx gorp.Tx,
	action access.Action,
	subjects []ontology.ID,
) error {
	return s.rbac.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  action,
		Objects: subjects,
	})
}

func (s *Service) retrieveUser(
	ctx context.Context,
	tx gorp.Tx,
	subject ontology.ID,
) (*user.User, error) {
	key, err := user.KeyFromOntologyID(subject)
	if err != nil {
		return nil, err
	}
	var u user.User
	if err = s.user.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return &u, nil
}
