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
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
// exactly one of a project and a range. A view in a group belongs to the project that
// holds the group. Create returns an error wrapping validate.ErrValidation when a view
// with v.Key exists.
func (w Writer) Create(ctx context.Context, v *Video) error {
	if v.Key == uuid.Nil() {
		v.Key = uuid.New()
	}
	if v.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, v.Group, resource.TypeVideo)
		if err != nil {
			return err
		}
		v.Project = proj
	}
	if err := v.Validate(); err != nil {
		return err
	}
	if err := project.ValidateOwner(v.Project, v.Range); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(v).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, v.Key, NewCreateAction(CreatePayload{Video: *v}))
	return nil
}

// CreateMany creates each of videos. It returns an error wrapping
// validate.ErrValidation when a view with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, videos *[]Video) error {
	for i := range *videos {
		if err := w.Create(ctx, &(*videos)[i]); err != nil {
			return err
		}
	}
	return nil
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

// Delete deletes the video views with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Video](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies acts to the video view with key and attaches them to the transaction
// of w, so its commit broadcasts them with the dispatch key of w. A move into a group
// also moves the view into the project that holds the group, and a move into another
// project moves it out of its group. Apply attaches the actions of each such move after
// acts and returns them. It returns a validation error, and stores nothing, when the
// actions leave the view invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Video](key)).
		ChangeErr(func(_ gorp.Context, prev Video) (Video, error) {
			next, err := Reduce(prev, acts...)
			if err != nil {
				return Video{}, err
			}
			var placed project.Placement
			if placed, added, err = project.Place(
				ctx,
				w.tx,
				resource.TypeVideo,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return Video{}, err
			}
			next.Group, next.Project = placed.Group, placed.Project
			if err = next.Validate(); err != nil {
				return Video{}, err
			}
			return next, project.ValidateOwner(next.Project, next.Range)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}
