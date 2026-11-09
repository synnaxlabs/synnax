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
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
)

// Copy is a request that reads records and copies them into what it writes.
type Copy interface {
	// Sources returns the records that the request copies.
	Sources() []resource.ID
}

// Returns is a response that returns records.
type Returns interface {
	// Returned returns the records that the response holds.
	Returned() []resource.ID
}

// Write returns a handler that runs apply in one write transaction, checks access on
// what the transaction changed, and commits. A denial or an error from apply rolls back
// every write and returns a zero response. The subject of the request needs:
//
//   - create on a new record, update on a changed record, and delete on a deleted
//     record, cascades and detaches included;
//   - update on the range of each range key-value pair that changes;
//   - create, update, or delete on the user or rack of each API key, password, or
//     session that is created, updated, or deleted, unless the subject is that user or
//     rack;
//   - update on the target of each place link added or removed, except a link to the
//     range, user, or rack that the two rules above cover;
//   - retrieve on the target of each use link added;
//   - retrieve on each source of a request that implements Copy;
//   - retrieve on each record that a response implementing Returns holds and the
//     transaction did not change.
//
// A link needs nothing when the write creates or deletes its target. Policies are read
// from committed state, so a policy or role that apply stages grants nothing. A denial
// wraps access.ErrDenied and names the record.
func Write[RQ, RS freighter.Payload](
	s *Service,
	apply func(context.Context, gorp.Tx, RQ) (RS, error),
) freighter.UnaryHandler[RQ, RS] {
	return func(ctx context.Context, req RQ) (RS, error) {
		var (
			res, zero RS
			sources   []resource.ID
		)
		if c, ok := any(req).(Copy); ok {
			sources = c.Sources()
		}
		if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			if res, err = apply(ctx, tx, req); err != nil {
				return err
			}
			return s.check(ctx, tx, append(slices.Clip(sources), unchanged(tx, res)...))
		}); err != nil {
			return zero, err
		}
		return res, nil
	}
}

// WritePrepared returns a handler that runs prepare outside any transaction, then runs
// apply with what prepare returns, as Write runs it. Put slow work that needs no
// transaction, such as a password hash, in prepare. An error from prepare returns
// before the transaction opens.
func WritePrepared[RQ, RS freighter.Payload, P any](
	s *Service,
	prepare func(context.Context, RQ) (P, error),
	apply func(context.Context, gorp.Tx, RQ, P) (RS, error),
) freighter.UnaryHandler[RQ, RS] {
	return func(ctx context.Context, req RQ) (RS, error) {
		prepared, err := prepare(ctx, req)
		if err != nil {
			var zero RS
			return zero, err
		}
		return Write(s, func(ctx context.Context, tx gorp.Tx, req RQ) (RS, error) {
			return apply(ctx, tx, req, prepared)
		})(ctx, req)
	}
}

// unchanged returns the records that res returns and tx did not change.
func unchanged[RS any](tx gorp.Tx, res RS) []resource.ID {
	r, ok := any(res).(Returns)
	if !ok {
		return nil
	}
	changed := make(set.Set[resource.ID])
	for _, c := range gorp.ChangesOf(tx) {
		// A record with no resource type is never returned.
		if id, err := resourceOf(c.Record); err == nil {
			changed.Add(id)
		}
	}
	return lo.Reject(r.Returned(), func(id resource.ID, _ int) bool {
		return changed.Contains(id)
	})
}

// check returns an error wrapping access.ErrDenied when the subject of ctx cannot make
// the changes of tx or read sources.
func (s *Service) check(ctx context.Context, tx gorp.Tx, sources []resource.ID) error {
	needs := make(map[access.Action][]resource.ID, len(access.AllActions))
	needs[access.ActionRetrieve] = slices.Clone(sources)
	subject := auth.GetSubject(ctx)
	// owner is the owner of the record being checked, or zero. The owner rule covers
	// the links of the record to its owner, so need skips them.
	var owner resource.ID
	need := func(action access.Action, target gorp.Target) error {
		id, err := resourceOf(target)
		if err != nil {
			return err
		}
		if owner.IsZero() || id != owner {
			needs[action] = append(needs[action], id)
		}
		return nil
	}
	changes := gorp.ChangesOf(tx)
	createdOrDeleted := make(set.Set[gorp.Target], len(changes))
	for _, c := range changes {
		if c.Kind != gorp.ChangeUpdate {
			createdOrDeleted.Add(c.Record)
		}
	}
	for _, c := range changes {
		var (
			action access.Action
			err    error
		)
		if owner, action, err = s.ownerOf(ctx, c); err != nil {
			return err
		}
		if owner.IsZero() {
			if err = need(actionOf(c.Kind), c.Record); err != nil {
				return err
			}
		} else if owner != subject {
			needs[action] = append(needs[action], owner)
		}
		for _, l := range c.Added {
			if createdOrDeleted.Contains(l.Target) {
				continue
			}
			if err := need(addedLinkAction(l.Kind), l.Target); err != nil {
				return err
			}
		}
		for _, l := range c.Removed {
			if l.Kind != gorp.Place || createdOrDeleted.Contains(l.Target) {
				continue
			}
			if err := need(access.ActionUpdate, l.Target); err != nil {
				return err
			}
		}
	}
	// The write lock holds commits off, and committed policies decide, so a write
	// cannot grant itself access.
	enforcer := s.cfg.RBAC.NewEnforcer(s.cfg.DB)
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

// ownerOf returns the resource that owns the record of c when its table has no resource
// type, and the action that c needs on it. A range key-value pair needs update on its
// range. An API key, password, or session needs the action of c on its user or rack. It
// returns a zero ID when the table has a resource type.
func (s *Service) ownerOf(
	ctx context.Context,
	c gorp.Change,
) (resource.ID, access.Action, error) {
	switch c.Record.Table {
	case types.Name[kv.Pair]():
		rng, err := kv.RangeOf(c.Record.Key)
		if err != nil {
			return resource.ID{}, "", err
		}
		return ranger.ResourceID(rng), access.ActionUpdate, nil
	case types.Name[apikey.APIKey]():
		return subjectOf(ctx, s.cfg.DB, c, func(k apikey.APIKey) resource.ID {
			return k.Subject
		})
	case types.Name[password.Password]():
		return subjectOf(ctx, s.cfg.DB, c, func(p password.Password) resource.ID {
			return p.Subject
		})
	case types.Name[svcauth.Session]():
		return subjectOf(ctx, s.cfg.DB, c, func(e svcauth.Session) resource.ID {
			return e.Subject
		})
	}
	return resource.ID{}, "", nil
}

// subjectOf returns the subject that holds the auth record of c and the action of c.
func subjectOf[E any](
	ctx context.Context,
	db *gorp.DB,
	c gorp.Change,
	subject func(E) resource.ID,
) (resource.ID, access.Action, error) {
	action := actionOf(c.Kind)
	if c.Kind != gorp.ChangeDelete {
		var e E
		if err := db.Decode(ctx, c.Value, &e); err != nil {
			return resource.ID{}, "", err
		}
		return subject(e), action, nil
	}
	// A deleted record names its subject only through the link it removed.
	for _, l := range c.Removed {
		if l.Kind == gorp.Place {
			id, err := resourceOf(l.Target)
			return id, action, err
		}
	}
	return resource.ID{}, "", errors.Newf(
		"[bind] - deleted %s %s removed no link to its subject",
		c.Record.Table,
		c.Record.Key,
	)
}

// resourceOf returns the resource of a record. A task config record maps to its task.
// It returns an error when no resource type matches the table of the record.
func resourceOf(t gorp.Target) (resource.ID, error) {
	if id, ok := taskconfig.TaskOf(t); ok {
		return id, nil
	}
	id := resource.ID{Type: resource.Type(t.Table), Key: t.Key}
	if t.Table == types.Name[alias.Alias]() {
		id.Type = resource.TypeRangeAlias
	}
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
