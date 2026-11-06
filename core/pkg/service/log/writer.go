// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create, update, and delete logs within Synnax. The writer executes
// all operations within the transaction provided to the Service.NewWriter method. If no
// transaction is provided, the writer will execute operations directly on the database.
type Writer struct {
	svc        *Service
	tx         gorp.Tx
	otgWriter  ontology.Writer
	otg        *ontology.Ontology
	table      *gorp.Table[Key, Log]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the create
// broadcasts it emits, so the client that sent the create recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// inTx calls f with w, or with a Writer on a transaction of its own that keeps the
// dispatch key of w when w has no transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	db, ok := w.tx.(*gorp.DB)
	if !ok {
		return f(w)
	}
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		bound := w.svc.NewWriter(tx)
		bound.dispatcher = w.dispatcher
		return f(bound)
	})
}

// Create creates the given log, assigning a new key when l.Key is zero. The log must
// belong to exactly one of a project and a range. An existing log keeps its group,
// project, and range. A new log in a group belongs to the project that holds the group.
func (w Writer) Create(ctx context.Context, l *Log) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, l) })
}

func (w Writer) create(ctx context.Context, l *Log) error {
	var (
		exists bool
		err    error
	)
	if l.Key == uuid.Nil() {
		l.Key = uuid.New()
	} else {
		var stored Log
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Log](l.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			l.Group, l.Project, l.Range = stored.Group, stored.Project, stored.Range
		}
	}
	if !exists && l.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, l.Group, resource.TypeLog)
		if err != nil {
			return err
		}
		l.Project = proj
	}
	l.ApplyDefaults()
	if err = l.Validate(); err != nil {
		return err
	}
	if err = project.ValidateOwner(l.Project, l.Range); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		otgID := l.ResourceID()
		if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		l.Key,
		NewCreateAction(CreatePayload{Log: *l}),
	)
	return nil
}

// SetGroup moves the logs with keys into g, or out of their group when g is nil. A log
// in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	p, err := project.GroupProject(ctx, w.tx, g, resource.TypeLog)
	if err != nil {
		return err
	}
	if err = setGroup(w.table, g, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.notifyGroup(ctx, g, keys)
	if p == nil {
		return nil
	}
	return w.setProject(ctx, *p, keys...)
}

// SetProject moves the logs with keys out of their group and into the project p. It
// returns a validation error when a log belongs to a range.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.notifyGroup(ctx, nil, keys)
	return w.setProject(ctx, p, keys...)
}

// setProject moves the logs with keys into the project p, and notifies a project
// change for each one that was not in p.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Log](keys...)).
		ChangeErr(func(_ gorp.Context, e Log) (Log, error) {
			if e.Project == nil || *e.Project != p {
				moved = append(moved, e.Key)
			}
			e.Project = &p
			return e, project.ValidateOwner(e.Project, e.Range)
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range moved {
		w.dispatcher.NotifyOnCommit(
			ctx,
			w.tx,
			k,
			NewSetProjectAction(SetProjectPayload{Project: p}),
		)
	}
	return nil
}

func (w Writer) notifyGroup(ctx context.Context, g *group.Key, keys []Key) {
	for _, k := range keys {
		w.dispatcher.NotifyOnCommit(
			ctx,
			w.tx,
			k,
			NewSetGroupAction(SetGroupPayload{Group: g}),
		)
	}
}

// CreateMany creates the given logs. If logs with the same key already exist, they will
// be overwritten.
func (w Writer) CreateMany(ctx context.Context, logs *[]Log) error {
	for i := range *logs {
		if err := w.Create(ctx, &(*logs)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the logs with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.
		table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Log](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, ResourceIDs(keys)...)
}
