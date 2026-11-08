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
	"github.com/synnaxlabs/x/types"
)

// Copy is a request that reads records and copies them into what it writes.
type Copy interface {
	// Sources returns the records that the request copies.
	Sources() []resource.ID
}

// Write returns a handler that runs apply in one write transaction, checks access on
// what the transaction changed, and commits. A denial or an error from apply rolls back
// every write and returns a zero response. The subject of the request needs:
//
//   - create on a new record, update on a changed record, and delete on a deleted
//     record, cascades and detaches included;
//   - update on the range of each range key-value pair that changes;
//   - update on the user or rack of each API key, password, or session that changes,
//     unless the subject is that user or rack;
//   - update on the target of each place link added or removed;
//   - retrieve on the target of each use link added;
//   - retrieve on each source of a request that implements Copy.
//
// Policies are read from committed state, so a policy or role that apply stages grants
// nothing. A denial wraps access.ErrDenied and names the record.
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
			return s.check(ctx, tx, sources)
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

// check returns an error wrapping access.ErrDenied when the subject of ctx cannot make
// the changes of tx or read sources.
func (s *Service) check(ctx context.Context, tx gorp.Tx, sources []resource.ID) error {
	needs := make(map[access.Action][]resource.ID, len(access.AllActions))
	needs[access.ActionRetrieve] = slices.Clone(sources)
	subject := auth.GetSubject(ctx)
	// self is true while checking a credential that the subject holds. The subject
	// writes it, and links it to itself, with no check.
	var self bool
	need := func(action access.Action, target gorp.Target) error {
		id, err := resourceOf(target)
		if err != nil {
			return err
		}
		if !self || id != subject {
			needs[action] = append(needs[action], id)
		}
		return nil
	}
	for _, c := range gorp.ChangesOf(tx) {
		owner, owned, err := s.ownerOf(ctx, c)
		if err != nil {
			return err
		}
		self = owned && owner == subject
		if owned {
			if !self {
				needs[access.ActionUpdate] = append(needs[access.ActionUpdate], owner)
			}
		} else if err = need(actionOf(c.Kind), c.Record); err != nil {
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
// type: the range of a range key-value pair, or the user or rack of an API key,
// password, or session. It reports false when the table has a resource type.
func (s *Service) ownerOf(
	ctx context.Context,
	c gorp.Change,
) (resource.ID, bool, error) {
	switch c.Record.Table {
	case types.Name[kv.Pair]():
		rng, err := kv.RangeOf(c.Record.Key)
		if err != nil {
			return resource.ID{}, false, err
		}
		return ranger.ResourceID(rng), true, nil
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
	return resource.ID{}, false, nil
}

// subjectOf returns the user or rack that holds the auth record of c.
func subjectOf[E any](
	ctx context.Context,
	db *gorp.DB,
	c gorp.Change,
	subject func(E) resource.ID,
) (resource.ID, bool, error) {
	if c.Kind != gorp.ChangeDelete {
		var e E
		if err := db.Decode(ctx, c.Value, &e); err != nil {
			return resource.ID{}, false, err
		}
		return subject(e), true, nil
	}
	// A deleted record names its subject only through the link it removed.
	for _, l := range c.Removed {
		if l.Kind == gorp.Place {
			id, err := resourceOf(l.Target)
			return id, true, err
		}
	}
	return resource.ID{}, false, errors.Newf(
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
