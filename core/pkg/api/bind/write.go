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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Copy is a request that reads records to make what it writes, as a copy or a replace
// does.
type Copy interface {
	// Sources returns the records that the request reads.
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
//     record, cascades and detaches included. An owned record (ServiceConfig.Owned)
//     that the subject owns before and after the change needs none of these;
//   - for a changed record with no resource type, update on the record that it is
//     placed in, its owner. A held record (ServiceConfig.Held) needs the action of the
//     change on its owner instead, unless the subject is that owner;
//   - update on the target of each place link added or removed, except a link to the
//     owner or to an owned record that the subject owns. For an owned record, a target
//     in ServiceConfig.OwnedPlaces needs only retrieve;
//   - retrieve on the target of each use link added. A rack is denied a new use of a
//     record that ServiceConfig.Vouched reports, and a move onto a rack of a record
//     that uses one or holds a record that uses one;
//   - retrieve on each source of a request that implements Copy;
//   - retrieve on each record that a response implementing Returns holds and the
//     transaction did not change.
//
// A link or an owner needs nothing when the write creates or deletes it. A record with
// no resource type must be placed in exactly one record. Policies are read from
// committed state, so a policy or role that apply stages grants nothing. A denial wraps
// access.ErrDenied and names the record.
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

// WriteUnchecked returns a handler that runs apply in one write transaction of db and
// commits it, with no access check. Use it only for an apply that checks its own
// access. An error from apply rolls back every write and returns a zero response.
func WriteUnchecked[RQ, RS freighter.Payload](
	db *gorp.DB,
	apply func(context.Context, gorp.Tx, RQ) (RS, error),
) freighter.UnaryHandler[RQ, RS] {
	return func(ctx context.Context, req RQ) (RS, error) {
		var res, zero RS
		if err := db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			res, err = apply(ctx, tx, req)
			return err
		}); err != nil {
			return zero, err
		}
		return res, nil
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
		if id, ok := resourceOf(c.Record); ok {
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
	needs := make(accessNeeds, len(access.AllActions))
	needs[access.ActionRetrieve] = slices.Clone(sources)
	subject := auth.GetSubject(ctx)
	changes := gorp.ChangesOf(tx)
	createdOrDeleted := make(set.Set[gorp.Target], len(changes))
	for _, c := range changes {
		if c.Kind != gorp.ChangeUpdate {
			createdOrDeleted.Add(c.Record)
		}
	}
	for _, c := range changes {
		if err := s.changeNeeds(
			ctx, tx, subject, c, createdOrDeleted, needs,
		); err != nil {
			return err
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

// accessNeeds maps each action to the resources that a write needs it on.
type accessNeeds map[access.Action][]resource.ID

// add records that a write needs action on the resource of target. It returns an error
// when the table of target has no resource type.
func (n accessNeeds) add(action access.Action, target gorp.Target) error {
	id, ok := resourceOf(target)
	if !ok {
		return errors.Newf("[bind] - table %s has no resource type", target.Table)
	}
	n[action] = append(n[action], id)
	return nil
}

// changeNeeds adds the access that subject needs to make c to needs.
func (s *Service) changeNeeds(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
	c gorp.Change,
	createdOrDeleted set.Set[gorp.Target],
	needs accessNeeds,
) error {
	owned := s.owned(c, subject)
	owner, err := s.recordNeeds(c, owned, subject, createdOrDeleted, needs)
	if err != nil {
		return err
	}
	if err := s.checkVouched(ctx, tx, subject, c); err != nil {
		return err
	}
	return s.linkNeeds(ctx, tx, subject, c, owner, owned, createdOrDeleted, needs)
}

// recordNeeds adds the access that subject needs on the record of c, or on its owner
// for a record with no resource type, to needs. It returns the owner, which is zero for
// a record with a resource type.
func (s *Service) recordNeeds(
	c gorp.Change,
	owned bool,
	subject resource.ID,
	createdOrDeleted set.Set[gorp.Target],
	needs accessNeeds,
) (gorp.Target, error) {
	if _, ok := resourceOf(c.Record); ok {
		if owned {
			return gorp.Target{}, nil
		}
		return gorp.Target{}, needs.add(actionOf(c.Kind), c.Record)
	}
	owner, err := ownerOf(c)
	if err != nil {
		return gorp.Target{}, err
	}
	if action, ok := s.ownerAction(c, owner, subject, createdOrDeleted); ok {
		if err = needs.add(action, owner); err != nil {
			return gorp.Target{}, err
		}
	}
	return owner, nil
}

// linkNeeds adds the access that the links c adds and removes need to needs. A link to
// owner, a link to a record that the write creates or deletes, and a place link into a
// record that subject owns need none.
func (s *Service) linkNeeds(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
	c gorp.Change,
	owner gorp.Target,
	owned bool,
	createdOrDeleted set.Set[gorp.Target],
	needs accessNeeds,
) error {
	need := func(l gorp.Link, action access.Action) error {
		if l.Target == owner || createdOrDeleted.Contains(l.Target) {
			return nil
		}
		if l.Kind == gorp.Place {
			// The target reads as the write leaves it. A write that makes subject its
			// owner changes the target, which needs update on its own.
			if own, err := s.owns(ctx, tx, subject, l.Target); err != nil || own {
				return err
			}
			if owned && s.ownedPlace(l) {
				action = access.ActionRetrieve
			}
		}
		return needs.add(action, l.Target)
	}
	for _, l := range c.Added {
		if err := need(l, addedLinkAction(l.Kind)); err != nil {
			return err
		}
	}
	for _, l := range c.Removed {
		if l.Kind != gorp.Place {
			continue
		}
		if err := need(l, access.ActionUpdate); err != nil {
			return err
		}
	}
	return nil
}

// owned reports whether the change is to a record of an owned table that the subject
// owns both before and after the change.
func (s *Service) owned(c gorp.Change, subject resource.ID) bool {
	if !slices.Contains(s.cfg.Owned, c.Record.Table) {
		return false
	}
	// Links holds the links after a create or an update, and before a delete.
	if c.Kind != gorp.ChangeUpdate {
		return uses(c.Links, subject)
	}
	before := append(
		lo.Reject(c.Links, func(l gorp.Link, _ int) bool {
			return slices.Contains(c.Added, l)
		}),
		c.Removed...,
	)
	return uses(c.Links, subject) && uses(before, subject)
}

// owns reports whether target is a record of an owned table that uses subject.
func (s *Service) owns(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
	target gorp.Target,
) (bool, error) {
	if !slices.Contains(s.cfg.Owned, target.Table) {
		return false, nil
	}
	links, err := gorp.LinksOf(ctx, tx, target)
	if err != nil {
		return false, err
	}
	return uses(links, subject), nil
}

// uses reports whether links hold a use link to subject.
func uses(links []gorp.Link, subject resource.ID) bool {
	return slices.ContainsFunc(links, func(l gorp.Link) bool {
		id, ok := resourceOf(l.Target)
		return l.Kind == gorp.Use && ok && id == subject
	})
}

// checkVouched returns an error wrapping access.ErrDenied when subject is a rack and c
// gives a record on a rack a use of a record that ServiceConfig.Vouched reports:
// through a new use link, or through a new place link onto a rack of a record that uses
// one, or that holds a record placed in it that uses one.
func (s *Service) checkVouched(
	ctx context.Context,
	tx gorp.Tx,
	subject resource.ID,
	c gorp.Change,
) error {
	if s.cfg.Vouched == nil ||
		subject.Type != resource.TypeRack ||
		c.Kind == gorp.ChangeDelete {
		return nil
	}
	moved := slices.ContainsFunc(c.Added, func(l gorp.Link) bool {
		target, ok := resourceOf(l.Target)
		return l.Kind == gorp.Place && ok && target.Type == resource.TypeRack
	})
	for _, l := range c.Links {
		added := slices.Contains(c.Added, l)
		if !added && !moved {
			continue
		}
		target, vouched, err := s.vouchedUse(ctx, tx, l)
		if err != nil {
			return err
		}
		if !vouched {
			continue
		}
		if added {
			return errors.Wrapf(
				access.ErrDenied,
				"%s cannot make a record use %s",
				subject,
				target,
			)
		}
		return errMove(subject, target)
	}
	if !moved {
		return nil
	}
	placed, err := gorp.PlacedIn(ctx, tx, c.Record)
	if err != nil {
		return err
	}
	for _, p := range placed {
		links, err := gorp.LinksOf(ctx, tx, p)
		if err != nil {
			return err
		}
		for _, l := range links {
			target, vouched, err := s.vouchedUse(ctx, tx, l)
			if err != nil {
				return err
			}
			if vouched {
				return errMove(subject, target)
			}
		}
	}
	return nil
}

// vouchedUse returns the target of l, and whether l is a use of a record that
// ServiceConfig.Vouched reports.
func (s *Service) vouchedUse(
	ctx context.Context,
	tx gorp.Tx,
	l gorp.Link,
) (resource.ID, bool, error) {
	target, ok := resourceOf(l.Target)
	if l.Kind != gorp.Use || !ok {
		return target, false, nil
	}
	vouched, err := s.cfg.Vouched(ctx, tx, target)
	return target, vouched, err
}

// errMove returns the denial of subject moving a record that uses target onto a rack.
func errMove(subject, target resource.ID) error {
	return errors.Wrapf(
		access.ErrDenied,
		"%s cannot move a record that uses %s onto a rack",
		subject,
		target,
	)
}

// ownedPlace reports whether l places an owned record in a table of OwnedPlaces.
func (s *Service) ownedPlace(l gorp.Link) bool {
	return l.Kind == gorp.Place && slices.Contains(s.cfg.OwnedPlaces, l.Target.Table)
}

// ownerOf returns the record that the record of c is placed in. It returns an error
// unless the record holds exactly one place link.
func ownerOf(c gorp.Change) (gorp.Target, error) {
	places := lo.Filter(c.Links, func(l gorp.Link, _ int) bool {
		return l.Kind == gorp.Place
	})
	if len(places) != 1 {
		return gorp.Target{}, errors.Newf(
			"[bind] - table %s has no resource type, and %s is placed in %d records",
			c.Record.Table,
			c.Record.Key,
			len(places),
		)
	}
	return places[0].Target, nil
}

// ownerAction returns the action that c needs on owner, and false when it needs none.
func (s *Service) ownerAction(
	c gorp.Change,
	owner gorp.Target,
	subject resource.ID,
	createdOrDeleted set.Set[gorp.Target],
) (access.Action, bool) {
	if createdOrDeleted.Contains(owner) {
		return "", false
	}
	if !slices.Contains(s.cfg.Held, c.Record.Table) {
		return access.ActionUpdate, true
	}
	if id, ok := resourceOf(owner); ok && id == subject {
		return "", false
	}
	return actionOf(c.Kind), true
}

// resourceOf returns the resource of a record, and false when the table of the record
// has no resource type.
func resourceOf(t gorp.Target) (resource.ID, bool) {
	id := resource.ID{Type: resource.Type(t.Table), Key: t.Key}
	return id, id.Type.IsValid()
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
