// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer sets and changes passwords in one transaction. It makes no identity check: the
// caller authorizes the change.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Set stores h as the password of the subject under username, replacing any password
// it holds. It returns [auth.ErrRepeatedUsername] when another subject holds username,
// a validation error when username is empty, and the errors of a write of h.
func (w Writer) Set(
	ctx context.Context,
	subject resource.ID,
	username string,
	h Hash,
) error {
	if username == "" {
		return errors.Wrap(validate.ErrValidation, "username: required")
	}
	if err := w.assertUsernameFree(ctx, subject, username); err != nil {
		return err
	}
	p, err := w.retrieve(ctx, subject)
	if errors.Is(err, query.ErrNotFound) {
		return w.svc.table.NewCreate().Entry(&Password{
			Key:      uuid.New(),
			Subject:  subject,
			Username: username,
			Hash:     h.value,
			Created:  telem.Now(),
		}).Exec(ctx, w.tx)
	}
	if err != nil {
		return err
	}
	if err = h.check(p); err != nil {
		return err
	}
	return w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Password](p.Key)).
		Change(func(_ gorp.Context, p Password) Password {
			p.Username = username
			p.Hash = h.value
			p.Created = next(p.Created)
			return p
		}).
		Exec(ctx, w.tx)
}

// Action is a change to a stored password. It never leaves the Core, since it can carry
// a hash. NewSetHashAction and NewSetUsernameAction make one.
type Action struct {
	hash     *Hash
	username string
}

// NewSetHashAction returns an action that stores h as the password and keeps the
// username.
func NewSetHashAction(h Hash) Action { return Action{hash: &h} }

// NewSetUsernameAction returns an action that changes the username of the password.
func NewSetUsernameAction(username string) Action { return Action{username: username} }

// Apply applies the actions in order to the password of the subject. It returns
// [auth.ErrInvalidCredentials] when a hash was proven against a password the subject no
// longer holds, [auth.ErrRepeatedUsername] when another subject holds a username that
// an action sets, a validation error when that username is empty, and an error wrapping
// query.ErrNotFound when the subject has no password.
func (w Writer) Apply(ctx context.Context, subject resource.ID, acts ...Action) error {
	p, err := w.retrieve(ctx, subject)
	if err != nil {
		return err
	}
	for _, a := range acts {
		if a.hash != nil {
			if err = a.hash.check(p); err != nil {
				return err
			}
			continue
		}
		if a.username == "" {
			return errors.Wrap(validate.ErrValidation, "username: required")
		}
		if err = w.assertUsernameFree(ctx, subject, a.username); err != nil {
			return err
		}
	}
	return w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Password](p.Key)).
		Change(func(_ gorp.Context, p Password) Password {
			for _, a := range acts {
				if a.hash == nil {
					p.Username = a.username
					continue
				}
				p.Hash = a.hash.value
				p.Created = next(p.Created)
			}
			return p
		}).
		Exec(ctx, w.tx)
}

// next always moves past prev, since sessions treat Created as the password's version.
func next(prev telem.TimeStamp) telem.TimeStamp { return max(telem.Now(), prev+1) }

func (w Writer) retrieve(ctx context.Context, subject resource.ID) (Password, error) {
	var p Password
	err := w.svc.NewRetrieve().
		Where(MatchSubjects(subject)).
		Entry(&p).
		Exec(ctx, w.tx)
	return p, err
}

func (w Writer) assertUsernameFree(
	ctx context.Context,
	subject resource.ID,
	username string,
) error {
	var holders []Password
	if err := w.svc.NewRetrieve().
		Where(MatchUsernames(username)).
		Entries(&holders).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, h := range holders {
		if h.Subject != subject {
			return errors.Wrapf(
				auth.ErrRepeatedUsername,
				"username %s already exists",
				username,
			)
		}
	}
	return nil
}
