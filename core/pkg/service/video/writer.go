// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Writer creates, edits, and deletes video views in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Video]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that puts dispatchKey on the broadcasts
// it sends, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates v, and gives it a new key when v.Key is zero. The view must belong to
// exactly one of a project and a range. An existing view keeps its group, project, and
// range. A new view in a group belongs to the project that holds the group.
func (w Writer) Create(ctx context.Context, v *Video) error {
	var (
		exists bool
		err    error
	)
	if v.Key == uuid.Nil() {
		v.Key = uuid.New()
	} else {
		var stored Video
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Video](v.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			v.Group, v.Project, v.Range = stored.Group, stored.Project, stored.Range
		}
	}
	if !exists && v.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, v.Group, resource.TypeVideo)
		if err != nil {
			return err
		}
		v.Project = proj
	}
	if err = v.Validate(); err != nil {
		return err
	}
	if err = project.ValidateOwner(v.Project, v.Range); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(v).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, v.Key, NewCreateAction(CreatePayload{Video: *v}))
	return nil
}

// CreateMany creates each of videos. A view with the key of an existing view replaces
// it.
func (w Writer) CreateMany(ctx context.Context, videos *[]Video) error {
	for i := range *videos {
		if err := w.Create(ctx, &(*videos)[i]); err != nil {
			return err
		}
	}
	return nil
}

// SetGroup moves the video views with keys into g, or out of their group when g is
// nil. A view in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	p, err := project.GroupProject(ctx, w.tx, g, resource.TypeVideo)
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

// SetProject moves the video views with keys out of their group and into the project
// p. It returns a validation error when a view belongs to a range.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.attachGroup(nil, keys)
	return w.setProject(ctx, p, keys...)
}

// setProject moves the video views with keys that are not in the project p into p,
// and attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, Video](keys...),
			gorp.Match(func(_ gorp.Context, v *Video) (bool, error) {
				return v.Project == nil || *v.Project != p, nil
			}),
		)).
		ChangeErr(func(_ gorp.Context, v Video) (Video, error) {
			moved = append(moved, v.Key)
			v.Project = &p
			return v, project.ValidateOwner(v.Project, v.Range)
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

// attachGroup attaches a set of the group of each video view with keys, or a remove of
// the group when g is nil.
func (w Writer) attachGroup(g *group.Key, keys []Key) {
	for _, k := range keys {
		a := NewRemoveAction(RemovePayload{Key: "group"})
		if g != nil {
			a = NewSetAction(SetPayload{Path: []string{"group"}, Value: *g})
		}
		w.dispatcher.Attach(w.tx, k, a)
	}
}

// Delete deletes the video views with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Video](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies acts to the video view with key and attaches them to the transaction
// of w, so its commit broadcasts them with the dispatch key of w. Returns a validation
// error, and stores nothing, when the actions leave the view invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Video](key)).
		ChangeErr(func(_ gorp.Context, v Video) (Video, error) {
			reduced, err := Reduce(v, acts...)
			if err != nil {
				return Video{}, err
			}
			if err = reduced.Validate(); err != nil {
				return Video{}, err
			}
			return reduced, nil
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}
