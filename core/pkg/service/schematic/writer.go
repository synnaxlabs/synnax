// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete schematics within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method.
type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Schematic]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given schematic, assigning a new key when s.Key is zero. The
// schematic must belong to a project. A schematic in a group belongs to the project
// that holds the group. Create returns an error wrapping validate.ErrValidation when a
// schematic with s.Key exists.
func (w Writer) Create(ctx context.Context, s *Schematic) error {
	return w.create(ctx, s, nil)
}

// create creates s. A copy passes its source, whose scales s may keep.
func (w Writer) create(ctx context.Context, s, source *Schematic) error {
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	}
	if s.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, s.Group, resource.TypeSchematic)
		if err != nil {
			return err
		}
		s.Project = proj
	}
	if err := s.Validate(); err != nil {
		return err
	}
	var prev Schematic
	if source != nil {
		prev = *source
	}
	if err := validateScales(prev, *s); err != nil {
		return err
	}
	if err := project.CheckMember(s.Project); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		s.Key,
		NewCreateAction(CreatePayload{Schematic: *s}),
	)
	return nil
}

// setProject moves the schematics with keys that are not in the project p into p, and
// attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, Schematic](keys...),
			gorp.Match(func(_ gorp.Context, e *Schematic) (bool, error) {
				return e.Project == nil || *e.Project != p, nil
			}),
		)).
		Change(func(_ gorp.Context, e Schematic) Schematic {
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

// CreateMany creates the given schematics. It returns an error wrapping
// validate.ErrValidation when a schematic with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, schematics *[]Schematic) error {
	for i := range *schematics {
		if err := w.Create(ctx, &(*schematics)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Copy creates a copy of the schematic with the given key and name, and binds it into
// result. The copy sits in no group and belongs to the project of the original.
func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	result *Schematic,
) error {
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Schematic](key)).
		Entry(result).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	source := *result
	result.Key = uuid.New()
	result.Name = name
	result.Group = nil
	return w.create(ctx, result, &source)
}

// Delete deletes the schematics with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Schematic](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies a sequence of actions to the schematic with the given key and attaches
// them to the transaction of w, so its commit broadcasts them with the dispatch key of
// w. A move into a group also moves the schematic into the project that holds the
// group, and a move into another project moves it out of its group. Apply attaches the
// actions of each such move after acts and returns them. It returns a validation error,
// and stores nothing, if the actions leave the schematic invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Schematic](key)).
		ChangeErr(func(_ gorp.Context, prev Schematic) (Schematic, error) {
			// Reduce edits the configs map in place, so keep the stored ones.
			stored := maps.Clone(prev.Configs)
			next, err := Reduce(prev, acts...)
			if err != nil {
				return Schematic{}, err
			}
			var placed project.Placement
			if placed, added, err = project.Place(
				ctx,
				w.tx,
				resource.TypeSchematic,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return Schematic{}, err
			}
			next.Group, next.Project = placed.Group, placed.Project
			if err = next.Validate(); err != nil {
				return Schematic{}, err
			}
			if err = validateScales(Schematic{Configs: stored}, next); err != nil {
				return Schematic{}, err
			}
			return next, project.CheckMember(next.Project)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}

// validateScales returns an error wrapping validate.ErrValidation when a config of next
// carries a scale that is not positive and differs from the scale prev stores for it. A
// zero scale sizes a symbol to nothing. A stored scale that the write keeps passes, so
// a stored zero does not block an edit to another field.
func validateScales(prev, next Schematic) error {
	for key, cfg := range next.Configs {
		scale, ok := scaleOf(cfg)
		if !ok || scale > 0 {
			continue
		}
		if old, stored := prev.Configs[key]; stored {
			if oldScale, hadScale := scaleOf(old); hadScale && oldScale == scale {
				continue
			}
		}
		return errors.Wrapf(
			validate.ErrValidation,
			"configs.%s.scale: must be positive, got %v",
			key,
			scale,
		)
	}
	return nil
}

// scaleOf returns the scale that cfg carries, and false when its variant has none.
func scaleOf(cfg NodeConfig) (float64, bool) {
	v := reflect.ValueOf(cfg.Variant)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0, false
	}
	f := v.FieldByName("Scale")
	if !f.IsValid() || f.Kind() != reflect.Float64 {
		return 0, false
	}
	return f.Float(), true
}
