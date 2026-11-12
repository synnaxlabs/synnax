// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Project]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates p in p.Group, assigning a new key when p.Key is zero. The commit
// carries a create action that holds the stored project. Create returns an error
// wrapping validate.ErrValidation when a project with p.Key exists.
func (w Writer) Create(ctx context.Context, p *Project) error {
	if p.Key == uuid.Nil() {
		p.Key = uuid.New()
	}
	if err := group.CheckMember(
		ctx,
		w.tx,
		p.Group,
		resource.TypeProject,
		nil,
	); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, p.Key, NewCreateAction(CreatePayload{Project: *p}))
	return nil
}

// Apply applies the actions to the project with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. Apply
// returns an error wrapping validate.ErrValidation when an action is a create, the
// result fails to decode or validate, or the new group cannot hold a project, and an
// error wrapping query.ErrNotFound when no project has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Project](key)).
		ChangeErr(func(_ gorp.Context, p Project) (Project, error) {
			next, err := Reduce(p, acts...)
			if err != nil {
				return Project{}, err
			}
			if err = next.Validate(); err != nil {
				return Project{}, err
			}
			if lo.FromPtr(next.Group) == lo.FromPtr(p.Group) {
				return next, nil
			}
			if err = group.CheckMember(
				ctx,
				w.tx,
				next.Group,
				resource.TypeProject,
				nil,
			); err != nil {
				return Project{}, err
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// SetGroup moves the projects with keys into g, or out of their group when g is nil.
// Each move applies a set of the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	set := NewSetAction(SetPayload{Path: []string{"group"}, Value: g})
	for _, k := range keys {
		if err := w.Apply(ctx, k, set); err != nil {
			return err
		}
	}
	return nil
}

// CreateMany creates the given projects. It returns an error wrapping
// validate.ErrValidation when a project with one of their keys exists.
func (w Writer) CreateMany(
	ctx context.Context,
	projects *[]Project,
) error {
	for i := range *projects {
		if err := w.Create(ctx, &(*projects)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the project with the given key by applying a set of the name. It
// returns query.ErrNotFound if no project with the key exists and a validation error if
// name is empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Project](keys...)).
		Exec(ctx, w.tx)
}
