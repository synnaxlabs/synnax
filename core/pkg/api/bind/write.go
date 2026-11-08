// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// Write returns a handler that runs apply in one write transaction, checks access on
// what the transaction changed, and commits. A denial or an error from apply rolls back
// every write and returns a zero response. The subject of the request needs:
//
//   - create on a new record, update on a changed record, and delete on a deleted
//     record, cascades and detaches included;
//   - update on the target of each place link added or removed;
//   - retrieve on the target of each use link added.
//
// Policies are read from committed state, so a policy or role that apply stages grants
// nothing. A denial wraps access.ErrDenied and names the record.
func Write[RQ, RS freighter.Payload](
	s *Service,
	apply func(context.Context, gorp.Tx, RQ) (RS, error),
) freighter.UnaryHandler[RQ, RS] {
	return func(ctx context.Context, req RQ) (RS, error) {
		var res, zero RS
		if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			if res, err = apply(ctx, tx, req); err != nil {
				return err
			}
			return s.check(ctx, tx)
		}); err != nil {
			return zero, err
		}
		return res, nil
	}
}

// check returns an error wrapping access.ErrDenied when the subject of ctx cannot make
// the changes of tx.
func (s *Service) check(ctx context.Context, tx gorp.Tx) error {
	needs := make(map[access.Action][]resource.ID, len(access.AllActions))
	need := func(action access.Action, target gorp.Target) error {
		id, err := resourceOf(target)
		if err != nil {
			return err
		}
		needs[action] = append(needs[action], id)
		return nil
	}
	for _, c := range gorp.ChangesOf(tx) {
		if err := need(actionOf(c.Kind), c.Record); err != nil {
			return err
		}
		for _, l := range c.Added {
			if err := need(addedLinkAction(l.Kind), l.Target); err != nil {
				return err
			}
		}
		for _, l := range c.Removed {
			if l.Kind != gorp.Place {
				continue
			}
			if err := need(access.ActionUpdate, l.Target); err != nil {
				return err
			}
		}
	}
	enforcer := s.cfg.RBAC.NewEnforcer(nil)
	subject := auth.GetSubject(ctx)
	for _, action := range access.AllActions {
		objects := needs[action]
		if len(objects) == 0 {
			continue
		}
		if err := enforcer.Enforce(ctx, access.Request{
			Subject: subject,
			Action:  action,
			Objects: lo.Uniq(objects),
		}); err != nil {
			return err
		}
	}
	return nil
}

// resourceOf returns the resource of a record. It returns an error when no resource
// type matches the table of the record.
func resourceOf(t gorp.Target) (resource.ID, error) {
	id := resource.ID{Type: resource.Type(t.Table), Key: t.Key}
	if !id.Type.IsValid() {
		return resource.ID{}, errors.Newf(
			"[bind] - table %s has no resource type",
			t.Table,
		)
	}
	return id, nil
}

func actionOf(kind gorp.ChangeKind) access.Action {
	switch kind {
	case gorp.ChangeCreate:
		return access.ActionCreate
	case gorp.ChangeUpdate:
		return access.ActionUpdate
	case gorp.ChangeDelete:
		return access.ActionDelete
	}
	panic(errors.Newf("[bind] - unknown change kind %d", kind))
}

func addedLinkAction(kind gorp.LinkKind) access.Action {
	switch kind {
	case gorp.Place:
		return access.ActionUpdate
	case gorp.Use:
		return access.ActionRetrieve
	}
	panic(errors.Newf("[bind] - unknown link kind %d", kind))
}
