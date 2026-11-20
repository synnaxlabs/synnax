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
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// writer creates links in one transaction. It makes no identity check.
type writer struct {
	svc *Service
	tx  gorp.Tx
}

// create links acc to subject and returns the link. It returns a validation error when
// the account is already linked or subject already has a link for the provider.
func (w writer) create(
	ctx context.Context,
	subject user.Key,
	acc account,
) (Link, error) {
	l := Link{
		Key:      uuid.New(),
		Subject:  subject,
		Provider: acc.Provider,
		Issuer:   acc.Issuer,
		Account:  acc.ID,
	}
	if err := l.Validate(); err != nil {
		return Link{}, err
	}
	linked, err := w.svc.newRetrieve().
		Where(MatchProviders(acc.Provider)).
		Where(Or(MatchAccounts(acc.ID), MatchSubjects(subject))).
		Exists(ctx, w.tx)
	if err != nil {
		return Link{}, err
	}
	if linked {
		return Link{}, errors.Wrapf(
			validate.ErrValidation,
			"a %q account is already linked to this user, or this account to another",
			acc.Provider,
		)
	}
	if err = w.svc.table.NewCreate().Entry(&l).Exec(ctx, w.tx); err != nil {
		return Link{}, err
	}
	return l, nil
}
