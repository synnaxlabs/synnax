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
	"time"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// SessionCookieParam is the request parameter a handler sets to ask the transport to
// set or clear the session cookie. Its value is a [SessionCookie]. Only the HTTP
// transport reads it.
const SessionCookieParam = "Synnax-Session-Cookie"

// SessionCookie asks the transport to set the session cookie for Lifetime, or to clear
// it when Lifetime is zero. An empty Token keeps the token the request carried.
type SessionCookie struct {
	// Token is the session token to put in the cookie.
	Token string
	// Lifetime is how long the browser keeps the cookie.
	Lifetime time.Duration
}

// SessionCreateRequest logs a subject in with a proof.
type SessionCreateRequest struct {
	// Proof is what the subject presents. Its method selects the authenticator.
	Proof auth.Proof `json:"proof" msgpack:"proof"`
	// Cookie asks for the token in a cookie that page scripts cannot read, in place of
	// the response body. A browser client sets it. Only the HTTP transport honors it.
	Cookie bool `json:"cookie" msgpack:"cookie"`
}

// SessionCreateResponse is the session a proof opened.
type SessionCreateResponse struct {
	// Token is the session token to send with every later request.
	Token string `json:"token" msgpack:"token"`
	// Session is the session the proof opened.
	Session auth.Session `json:"session" msgpack:"session"`
	// User is the user record of the subject, when the subject is a user.
	User *user.User `json:"user,omitempty" msgpack:"user,omitempty"`
}

// CreateSession logs the subject in with the proof in req and returns the session, its
// token, and the user record when the subject is a user. With req.Cookie set, the token
// goes to the transport through [SessionCookieParam] and the body carries none.
func (s *Service) CreateSession(
	ctx context.Context,
	req SessionCreateRequest,
) (SessionCreateResponse, error) {
	session, tk, err := s.auth.Login(ctx, req.Proof)
	if err != nil {
		return SessionCreateResponse{}, err
	}
	res := SessionCreateResponse{
		Token:   s.deliver(ctx, tk, req.Cookie),
		Session: session,
	}
	if res.User, err = s.userOf(ctx, session.Subject); err != nil {
		return SessionCreateResponse{}, err
	}
	return res, nil
}

// deliver returns token, or with cookie set hands it to the transport through
// [SessionCookieParam] and returns an empty token.
func (s *Service) deliver(ctx context.Context, token string, cookie bool) string {
	if !cookie {
		return token
	}
	freighter.MDFromContext(ctx).Set(SessionCookieParam, SessionCookie{
		Token:    token,
		Lifetime: s.auth.IdleTimeout(),
	})
	return ""
}

// SessionRenewResponse is the caller's session, so a client that holds only a token
// learns who it is.
type SessionRenewResponse struct {
	// Session is the caller's session.
	Session auth.Session `json:"session" msgpack:"session"`
	// User is the user record of the subject, when the subject is a user.
	User *user.User `json:"user,omitempty" msgpack:"user,omitempty"`
}

// RenewSession reports activity on the caller's session and returns it. The middleware
// records the activity when it checks the token.
func (s *Service) RenewSession(
	ctx context.Context,
	_ struct{},
) (SessionRenewResponse, error) {
	session := getSession(ctx)
	u, err := s.userOf(ctx, session.Subject)
	if err != nil {
		return SessionRenewResponse{}, err
	}
	return SessionRenewResponse{Session: session, User: u}, nil
}

func (s *Service) userOf(ctx context.Context, subject ontology.ID) (*user.User, error) {
	if subject.Type != ontology.ResourceTypeUser {
		return nil, nil
	}
	key, err := uuid.Parse(subject.Key)
	if err != nil {
		return nil, err
	}
	var u user.User
	if err := s.user.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return &u, nil
}

// SessionRetrieveRequest selects sessions to list.
type SessionRetrieveRequest struct {
	// Subjects limits the result to these subjects' sessions. Empty means the caller.
	Subjects []ontology.ID `json:"subjects" msgpack:"subjects"`
}

// SessionRetrieveResponse lists sessions.
type SessionRetrieveResponse struct {
	// Sessions are the sessions that matched.
	Sessions []auth.Session `json:"sessions" msgpack:"sessions"`
}

// RetrieveSessions lists the caller's sessions, or the sessions of the given subjects
// when the caller may retrieve those subjects.
func (s *Service) RetrieveSessions(
	ctx context.Context,
	req SessionRetrieveRequest,
) (SessionRetrieveResponse, error) {
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
		return SessionRetrieveResponse{}, err
	}
	sessions := make([]auth.Session, 0)
	if err := s.auth.NewRetrieve().
		Where(auth.MatchSubjects(subjects...)).
		Entries(&sessions).
		Exec(ctx, nil); err != nil {
		return SessionRetrieveResponse{}, err
	}
	return SessionRetrieveResponse{Sessions: sessions}, nil
}

// SessionDeleteRequest names sessions to end.
type SessionDeleteRequest struct {
	// Keys are the sessions to end.
	Keys []auth.SessionKey `json:"keys" msgpack:"keys"`
}

// DeleteSessions ends the given sessions. A caller may end its own sessions; ending
// another subject's needs permission to delete that subject. Unknown keys are ignored.
// Ending the caller's own session asks the transport to clear its cookie, if it set
// one.
func (s *Service) DeleteSessions(
	ctx context.Context,
	req SessionDeleteRequest,
) (struct{}, error) {
	var sessions []auth.Session
	if err := s.auth.NewRetrieve().
		Where(auth.MatchKeys(req.Keys...)).
		Entries(&sessions).
		Exec(ctx, nil); err != nil && !errors.Is(err, query.ErrNotFound) {
		return struct{}{}, err
	}
	subjects := lo.Uniq(lo.Map(sessions, func(e auth.Session, _ int) ontology.ID {
		return e.Subject
	}))
	if err := s.enforceOthers(
		ctx,
		GetSubject(ctx),
		access.ActionDelete,
		subjects,
	); err != nil {
		return struct{}{}, err
	}
	if lo.Contains(req.Keys, getSession(ctx).Key) {
		freighter.MDFromContext(ctx).Set(SessionCookieParam, SessionCookie{})
	}
	return struct{}{}, s.auth.Logout(ctx, req.Keys...)
}

func (s *Service) enforceOthers(
	ctx context.Context,
	caller ontology.ID,
	action access.Action,
	subjects []ontology.ID,
) error {
	others := lo.Filter(subjects, func(id ontology.ID, _ int) bool {
		return id != caller
	})
	if len(others) == 0 {
		return nil
	}
	return s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: caller,
		Action:  action,
		Objects: others,
	})
}
