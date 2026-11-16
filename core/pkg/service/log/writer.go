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
	"slices"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// validateWrite adds the one-entry-per-channel rule to Log.Validate, as Oracle has no
// rule for uniqueness.
func validateWrite(l Log) error {
	if err := l.Validate(); err != nil {
		return err
	}
	v := validate.New("Log")
	seen := make(set.Set[channel.Key], len(l.Channels))
	for i, e := range l.Channels {
		v.Ternaryf(
			"channels."+strconv.Itoa(i)+".channel",
			seen.Contains(e.Channel),
			"channel %d already has an entry",
			e.Channel,
		)
		seen.Add(e.Channel)
	}
	return v.Error()
}

// Writer is used to create, update, and delete logs within Synnax. The writer executes
// all operations within the transaction provided to the Service.NewWriter method.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Log]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given log, assigning a new key when l.Key is zero. The log must
// belong to a project. A log in a group belongs to the project that holds the group.
// Create returns an error wrapping validate.ErrValidation when a log with l.Key exists.
func (w Writer) Create(ctx context.Context, l *Log) error {
	if l.Key == uuid.Nil() {
		l.Key = uuid.New()
	}
	if l.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, l.Group, resource.TypeLog)
		if err != nil {
			return err
		}
		l.Project = proj
	}
	if err := validateWrite(*l); err != nil {
		return err
	}
	if err := project.CheckMember(l.Project); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		l.Key,
		NewCreateAction(CreatePayload{Log: *l}),
	)
	return nil
}

// setProject moves the logs with keys that are not in the project p into p, and
// attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, Log](keys...),
			gorp.Match(func(_ gorp.Context, e *Log) (bool, error) {
				return e.Project == nil || *e.Project != p, nil
			}),
		)).
		Change(func(_ gorp.Context, e Log) Log {
			moved = append(moved, e.Key)
			e.Project = &p
			return e
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range moved {
		w.dispatcher.Attach(
			w.tx,
			k,
			NewSetAction(SetPayload{Path: []string{"project"}, Value: p}),
		)
	}
	return nil
}

// CreateMany creates the given logs. It returns an error wrapping
// validate.ErrValidation when a log with one of their keys exists.
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
	return w.
		table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Log](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies a sequence of actions to the log with the given key and attaches them
// to the transaction of w, so its commit broadcasts them with the dispatch key of w. A
// move into a group also moves the log into the project that holds the group, and a
// move into another project moves it out of its group. Apply attaches the actions of
// each such move after acts and returns them. It returns a validation error, and stores
// nothing, if the actions leave the log invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Log](key)).
		ChangeErr(func(_ gorp.Context, prev Log) (Log, error) {
			next, err := Reduce(prev, acts...)
			if err != nil {
				return Log{}, err
			}
			var placed project.Placement
			if placed, added, err = project.Place(
				ctx,
				w.tx,
				resource.TypeLog,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return Log{}, err
			}
			next.Group, next.Project = placed.Group, placed.Project
			if err = validateWrite(next); err != nil {
				return Log{}, err
			}
			return next, project.CheckMember(next.Project)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}
