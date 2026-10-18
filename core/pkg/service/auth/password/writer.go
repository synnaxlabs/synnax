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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer sets, renames, and deletes passwords in one transaction. It makes no identity
// check: the caller authorizes the change.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Set stores creds as the subject's password, replacing any it already holds. It
// returns [auth.ErrRepeatedUsername] when another subject holds creds.Username and a
// validation error when creds has an empty username or password.
func (w Writer) Set(
	ctx context.Context,
	subject ontology.ID,
	creds auth.Credentials,
) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	if err := w.assertUsernameFree(ctx, subject, creds.Username); err != nil {
		return err
	}
	hash, err := w.svc.current.hash(creds.Password)
	if err != nil {
		return err
	}
	p, err := w.retrieve(ctx, subject)
	if errors.Is(err, query.ErrNotFound) {
		return w.svc.table.NewCreate().Entry(&Password{
			Key:      uuid.New(),
			Subject:  subject,
			Username: creds.Username,
			Hash:     hash,
			Created:  telem.Now(),
		}).Exec(ctx, w.tx)
	}
	if err != nil {
		return err
	}
	return w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Password](p.Key)).
		Change(func(_ gorp.Context, p Password) Password {
			p.Username = creds.Username
			p.Hash = hash
			p.Created = telem.Now()
			return p
		}).
		Exec(ctx, w.tx)
}

// Rename changes the username of the subject's password. It returns
// [auth.ErrRepeatedUsername] when another subject holds username and an error wrapping
// query.ErrNotFound when the subject has no password.
func (w Writer) Rename(
	ctx context.Context,
	subject ontology.ID,
	username string,
) error {
	if err := w.assertUsernameFree(ctx, subject, username); err != nil {
		return err
	}
	p, err := w.retrieve(ctx, subject)
	if err != nil {
		return err
	}
	return w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Password](p.Key)).
		Change(func(_ gorp.Context, p Password) Password {
			p.Username = username
			return p
		}).
		Exec(ctx, w.tx)
}

// DeleteFor removes the passwords of the given subjects.
func (w Writer) DeleteFor(ctx context.Context, subjects ...ontology.ID) error {
	return w.svc.table.NewDelete().
		Where(gorp.Match(func(_ gorp.Context, p *Password) (bool, error) {
			return lo.Contains(subjects, p.Subject), nil
		})).
		Exec(ctx, w.tx)
}

func (w Writer) retrieve(ctx context.Context, subject ontology.ID) (Password, error) {
	var p Password
	err := w.svc.NewRetrieve().
		Where(MatchSubjects(subject)).
		Entry(&p).
		Exec(ctx, w.tx)
	return p, err
}

func (w Writer) assertUsernameFree(
	ctx context.Context,
	subject ontology.ID,
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
