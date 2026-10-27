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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// SessionCreateRequest logs a subject in with a proof.
type SessionCreateRequest struct {
	// Proof is what the subject presents. Its method selects the authenticator.
	Proof auth.Proof `json:"proof" msgpack:"proof"`
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
// token, and the user record when the subject is a user.
func (s *Service) CreateSession(
	ctx context.Context,
	req SessionCreateRequest,
) (SessionCreateResponse, error) {
	session, tk, err := s.auth.Login(ctx, req.Proof)
	if err != nil {
		return SessionCreateResponse{}, err
	}
	res := SessionCreateResponse{Token: tk, Session: session}
	if session.Subject.Type == ontology.ResourceTypeUser {
		key, err := uuid.Parse(session.Subject.Key)
		if err != nil {
			return SessionCreateResponse{}, err
		}
		var u user.User
		if err := s.user.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u).
			Exec(ctx, nil); err != nil {
			return SessionCreateResponse{}, err
		}
		var passwords []password.Password
		if err := s.password.NewRetrieve().
			Where(password.MatchSubjects(u.OntologyID())).
			Entries(&passwords).
			Exec(ctx, nil); err != nil {
			return SessionCreateResponse{}, err
		}
		if len(passwords) > 0 {
			u.Username = passwords[0].Username
		}
		res.User = &u
	}
	return res, nil
}

// RenewSession reports activity on the caller's session. The middleware records the
// activity when it checks the token, so the handler has nothing left to do.
func (s *Service) RenewSession(context.Context, struct{}) (struct{}, error) {
	return struct{}{}, nil
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
