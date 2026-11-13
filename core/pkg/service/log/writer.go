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
	"strconv"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
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
// belong to a project. An existing log keeps its group and project. A new log in a
// group belongs to the project that holds the group.
func (w Writer) Create(ctx context.Context, l *Log) error {
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
			l.Group, l.Project = stored.Group, stored.Project
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
	if err = validateWrite(*l); err != nil {
		return err
	}
	if err = project.ValidateOwner(l.Project); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
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
	w.attachGroup(g, keys)
	if p == nil {
		return nil
	}
	return w.setProject(ctx, *p, keys...)
}

// SetProject moves the logs with keys out of their group and into the project p.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.attachGroup(nil, keys)
	return w.setProject(ctx, p, keys...)
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

// attachGroup attaches a set of the group of each log with keys, or a remove of the
// group when g is nil.
func (w Writer) attachGroup(g *group.Key, keys []Key) {
	for _, k := range keys {
		a := NewRemoveAction(RemovePayload{Key: "group"})
		if g != nil {
			a = NewSetAction(SetPayload{Path: []string{"group"}, Value: *g})
		}
		w.dispatcher.Attach(w.tx, k, a)
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
	return w.
		table.
		NewDelete().
		Where(gorp.MatchKeys[Key, Log](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies a sequence of actions to the log with the given key and attaches them
// to the transaction of w, so its commit broadcasts them with the dispatch key of w.
// Returns a validation error, and stores nothing, if the actions leave the log invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Log](key)).
		ChangeErr(func(_ gorp.Context, l Log) (Log, error) {
			reduced, err := Reduce(l, acts...)
			if err != nil {
				return Log{}, err
			}
			if err = validateWrite(reduced); err != nil {
				return Log{}, err
			}
			return reduced, nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}
