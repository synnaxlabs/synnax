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
)

// Writer creates and deletes links in one transaction. It makes no identity check and
// ends no sessions: the caller authorizes the change and revokes the sessions of
// deleted links.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Create links account to u and returns the link. It returns [ErrProviderLinked] when u
// already has a link for the account's issuer, and [ErrAccountLinked] when another
// user holds the account.
func (w Writer) Create(
	ctx context.Context,
	u user.Key,
	account Account,
) (Link, error) {
	l := Link{
		Key:     uuid.New(),
		User:    u,
		Issuer:  account.Issuer,
		Account: account.ID,
		Label:   account.Label,
	}
	if err := l.Validate(); err != nil {
		return Link{}, err
	}
	if err := w.refuseLinked(ctx, u, account); err != nil {
		return Link{}, err
	}
	if err := w.svc.table.NewCreate().Entry(&l).Exec(ctx, w.tx); err != nil {
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

// refuseLinked returns ErrProviderLinked or ErrAccountLinked when account cannot link
// to u.
func (w Writer) refuseLinked(ctx context.Context, u user.Key, account Account) error {
	held, err := w.svc.NewRetrieve().
		Where(MatchIssuers(account.Issuer)).
		Where(MatchUsers(u)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if held {
		return ErrProviderLinked
	}
	taken, err := w.svc.NewRetrieve().
		Where(MatchIssuers(account.Issuer)).
		Where(MatchAccounts(account.ID)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if taken {
		return ErrAccountLinked
	}
	return nil
}
