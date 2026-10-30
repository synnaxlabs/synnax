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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates and deletes links in one transaction. It makes no identity check and
// ends no sessions: the caller authorizes the change and revokes the sessions of
// deleted links.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Create links account to subject and returns the link. It returns a validation error
// when the account is already linked or subject already has a link for the provider.
func (w Writer) Create(
	ctx context.Context,
	subject ontology.ID,
	account Account,
) (Link, error) {
	l := Link{
		Key:      uuid.New(),
		Subject:  subject,
		Provider: account.Provider,
		Account:  account.ID,
	}
	if err := l.Validate(); err != nil {
		return Link{}, err
	}
	linked, err := w.svc.NewRetrieve().
		Where(MatchProviders(account.Provider)).
		Where(Or(MatchAccounts(account.ID), MatchSubjects(subject))).
		Exists(ctx, w.tx)
	if err != nil {
		return Link{}, err
	}
	if linked {
		return Link{}, errors.Wrapf(
			validate.ErrValidation,
			"a %q account is already linked to this user, or this account to another",
			account.Provider,
		)
	}
	if err = w.svc.table.NewCreate().Entry(&l).Exec(ctx, w.tx); err != nil {
		return Link{}, err
	}
	return l, nil
}

// Delete removes the links with the given keys. Unknown keys are ignored.
func (w Writer) Delete(ctx context.Context, keys ...LinkKey) error {
	return w.svc.table.NewDelete().
		Where(gorp.MatchKeys[LinkKey, Link](keys...)).
		Exec(ctx, w.tx)
}

func (w Writer) deleteFor(ctx context.Context, subjects ...ontology.ID) error {
	return w.svc.table.NewDelete().
		Where(MatchSubjects(subjects...)(w.svc.NewRetrieve())).
		Exec(ctx, w.tx)
}
