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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/oidc/invite"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// CreateInviteRequest invites a user.
type CreateInviteRequest struct {
	// User is the user to invite.
	User user.Key `json:"user" msgpack:"user"`
}

// CreateInviteResponse is a new invite.
type CreateInviteResponse struct {
	// Token redeems the invite. It is the only copy of its secret.
	Token string `json:"token" msgpack:"token"`
	// Invite is the stored invite.
	Invite invite.Invite `json:"invite" msgpack:"invite"`
}

// CreateInvite replaces the invite of the user. It returns a validation error when the
// oidc method is off or the user does not exist.
func (s *Service) CreateInvite(
	ctx context.Context,
	tx gorp.Tx,
	req CreateInviteRequest,
) (CreateInviteResponse, error) {
	if err := s.auth.Accepts(auth.MethodOIDC); err != nil {
		return CreateInviteResponse{}, err
	}
	i, token, err := s.invite.NewWriter(tx).Create(ctx, req.User)
	if err != nil {
		return CreateInviteResponse{}, err
	}
	return CreateInviteResponse{Token: token, Invite: i}, nil
}

// RetrieveInvitesRequest selects invites to list.
type RetrieveInvitesRequest struct {
	// Subjects limits the result to these users' invites. Empty means the caller.
	Subjects []resource.ID `json:"subjects" msgpack:"subjects"`
}

// Holders implements bind.Held.
func (r RetrieveInvitesRequest) Holders() []resource.ID { return r.Subjects }

// RetrieveInvitesResponse lists invites.
type RetrieveInvitesResponse struct {
	// Invites are the live invites that matched.
	Invites []invite.Invite `json:"invites" msgpack:"invites"`
}

// RetrieveInvites lists the live invites of the given users in snap, or the caller's
// invite when the request names no user.
func (s *Service) RetrieveInvites(
	ctx context.Context,
	snap gorp.Snapshot,
	req RetrieveInvitesRequest,
) (RetrieveInvitesResponse, error) {
	keys, err := userKeys(ctx, req.Subjects)
	if err != nil {
		return RetrieveInvitesResponse{}, err
	}
	invites := make([]invite.Invite, 0)
	if err = s.invite.NewRetrieve().
		Where(invite.MatchUsers(keys...)).
		Where(s.invite.MatchLive()).
		Entries(&invites).
		Exec(ctx, snap); err != nil {
		return RetrieveInvitesResponse{}, err
	}
	return RetrieveInvitesResponse{Invites: invites}, nil
}

// DeleteInvitesRequest names invites to revoke.
type DeleteInvitesRequest struct {
	// Keys are the invites to revoke.
	Keys []invite.Key `json:"keys" msgpack:"keys"`
}

// DeleteInvites revokes the given invites. Unknown keys are ignored.
func (s *Service) DeleteInvites(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteInvitesRequest,
) (struct{}, error) {
	return struct{}{}, s.invite.NewWriter(tx).Delete(ctx, req.Keys...)
}

// RedeemInviteRequest links a provider account through an invite.
type RedeemInviteRequest struct {
	// Token is the invite's token.
	Token string `json:"token" msgpack:"token"`
	// Credentials prove the provider account.
	Credentials auth.OIDCCredentials `json:"credentials" msgpack:"credentials"`
	// Cookie asks for the session token in a cookie, as session.CreateRequest does.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// RedeemInvite links the provider account that the credentials prove to the invited
// user, spends the invite, and opens a session as a login does. It needs no session.
// It returns [auth.ErrInvalidCredentials] for an unknown, used, or expired token, and
// a validation error when the oidc method is off, the account is already linked, or
// the user already has a link for the provider. A failed redeem keeps the invite.
func (s *Service) RedeemInvite(
	ctx context.Context,
	req RedeemInviteRequest,
) (session.CreateResponse, error) {
	if err := s.auth.Accepts(auth.MethodOIDC); err != nil {
		return session.CreateResponse{}, err
	}
	account, err := s.oidc.Verify(ctx, req.Credentials)
	if err != nil {
		return session.CreateResponse{}, err
	}
	var (
		ses   auth.Session
		token string
	)
	if err = s.invite.Redeem(ctx, req.Token, func(tx gorp.Tx, i invite.Invite) error {
		link, err := s.oidc.NewWriter(tx).Create(ctx, i.User, account)
		if err != nil {
			return err
		}
		ses, token, err = s.auth.NewWriter(tx).
			Open(ctx, user.ResourceID(i.User), link.Key)
		return err
	}); err != nil {
		return session.CreateResponse{}, err
	}
	return s.session.Respond(ctx, ses, token, req.Cookie)
}

// RedeemInviteWithPasswordRequest gives an invited user its first username and
// password.
type RedeemInviteWithPasswordRequest struct {
	// Token is the invite's token.
	Token string `json:"token" msgpack:"token"`
	// Credentials are the username and password the invited user logs in with.
	Credentials auth.Credentials `json:"credentials" msgpack:"credentials"`
	// Cookie asks for the session token in a cookie, as session.CreateRequest does.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// RedeemInviteWithPassword gives the invited user its first username and password and
// opens a session, in the transaction that consumes the invite. It returns
// auth.ErrInvalidCredentials for an unknown, used, or expired token,
// auth.ErrRepeatedUsername for a taken username, and a validation error when the Core
// does not accept passwords, a credential is empty, or the user has a password. A
// refused request keeps the invite.
func (s *Service) RedeemInviteWithPassword(
	ctx context.Context,
	req RedeemInviteWithPasswordRequest,
) (session.CreateResponse, error) {
	if err := s.auth.Accepts(auth.MethodPassword); err != nil {
		return session.CreateResponse{}, err
	}
	if err := req.Credentials.Validate(); err != nil {
		return session.CreateResponse{}, err
	}
	// a hash is slow, so a token that redeems nothing costs none
	if err := s.invite.Check(ctx, req.Token); err != nil {
		return session.CreateResponse{}, err
	}
	hash, err := s.password.Hash(req.Credentials.Password)
	if err != nil {
		return session.CreateResponse{}, err
	}
	var (
		ses   auth.Session
		token string
	)
	if err = s.invite.Redeem(ctx, req.Token, func(tx gorp.Tx, i invite.Invite) error {
		subject := user.ResourceID(i.User)
		exists, err := s.password.NewRetrieve().
			Where(password.MatchSubjects(subject)).
			Exists(ctx, tx)
		if err != nil {
			return err
		}
		if exists {
			return errors.Wrapf(
				validate.ErrValidation,
				"%s already has a password",
				subject,
			)
		}
		if err = s.password.NewWriter(tx).
			Set(ctx, subject, req.Credentials.Username, hash); err != nil {
			return err
		}
		var p password.Password
		if err = s.password.NewRetrieve().
			Where(password.MatchSubjects(subject)).
			Entry(&p).
			Exec(ctx, tx); err != nil {
			return err
		}
		ses, token, err = s.auth.NewWriter(tx).Open(ctx, subject, p.Key)
		return err
	}); err != nil {
		return session.CreateResponse{}, err
	}
	return s.session.Respond(ctx, ses, token, req.Cookie)
}
