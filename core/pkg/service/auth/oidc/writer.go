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

// create links acc to u and returns the link. It returns a validation error when the
// account is already linked or u already has a link for the account's issuer.
func (w writer) create(
	ctx context.Context,
	u user.Key,
	acc account,
) (Link, error) {
	l := Link{
		Key:     uuid.New(),
		User:    u,
		Issuer:  acc.Issuer,
		Account: acc.ID,
	}
	if err := l.Validate(); err != nil {
		return Link{}, err
	}
	linked, err := w.svc.newRetrieve().
		Where(MatchIssuers(acc.Issuer)).
		Where(Or(MatchAccounts(acc.ID), MatchUsers(u))).
		Exists(ctx, w.tx)
	if err != nil {
		return Link{}, err
	}
	if linked {
		return Link{}, errors.Wrapf(
			validate.ErrValidation,
			"an account of %s is already linked to this user, or this account to another",
			acc.Issuer,
		)
	}
	if err = w.svc.table.NewCreate().Entry(&l).Exec(ctx, w.tx); err != nil {
		return Link{}, err
	}
	return l, nil
}
