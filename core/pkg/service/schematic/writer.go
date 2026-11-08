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
	"encoding/json/v2"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete schematics within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	svc        *Service
	tx         gorp.Tx
	table      *gorp.Table[Key, Schematic]
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

// Create creates the given schematic, assigning a new key when s.Key is zero. The
// schematic must belong to exactly one of a project and a range. An existing schematic
// keeps its group, project, and range. A new schematic in a group belongs to the
// project that holds the group.
func (w Writer) Create(ctx context.Context, s *Schematic) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, s) })
}

func (w Writer) create(ctx context.Context, s *Schematic) (err error) {
	var (
		exists bool
		stored Schematic
	)
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	} else {
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Schematic](s.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			s.Group, s.Project, s.Range = stored.Group, stored.Project, stored.Range
		}
	}
	if !exists && s.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, s.Group, resource.TypeSchematic)
		if err != nil {
			return err
		}
		s.Project = proj
	}
	if err = s.Validate(); err != nil {
		return err
	}
	if err = validateScales(stored, *s); err != nil {
		return err
	}
	if err = project.ValidateOwner(s.Project, s.Range); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		s.Key,
		NewCreateAction(CreatePayload{Schematic: *s}),
	)
	return nil
}

// SetGroup moves the schematics with keys into g, or out of their group when g is nil.
// A schematic in a group belongs to the project that holds the group.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		p, err := project.GroupProject(ctx, w.tx, g, resource.TypeSchematic)
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
	})
}

// SetProject moves the schematics with keys out of their group and into the project p.
// It returns a validation error when a schematic belongs to a range.
func (w Writer) SetProject(ctx context.Context, p project.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := setGroup(w.table, nil, keys...).Exec(ctx, w.tx); err != nil {
			return err
		}
		w.notifyGroup(ctx, nil, keys)
		return w.setProject(ctx, p, keys...)
	})
}

// setProject moves the schematics with keys into the project p, and notifies a project
// change for each one that was not in p.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Schematic](keys...)).
		ChangeErr(func(_ gorp.Context, e Schematic) (Schematic, error) {
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

// CreateMany creates the given schematics. If schematics with the same key already
// exist, they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, schematics *[]Schematic) error {
	for i := range *schematics {
		if err := w.Create(ctx, &(*schematics)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Copy creates a copy of the schematic with the given key and name, and binds it into
// result. A snapshot copy is no longer editable. The copy sits in no group. When rng is
// set, the copy belongs to that range. Otherwise it keeps the owner of the original.
func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
	rng *ranger.Key,
	result *Schematic,
) error {
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Schematic](key)).
		Entry(result).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	result.Key = uuid.New()
	result.Name = name
	result.Snapshot = snapshot
	result.Group = nil
	if rng != nil {
		result.Project, result.Range = nil, rng
	}
	return w.Create(ctx, result)
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

// validateScales returns an error wrapping validate.ErrValidation when a config of next
// carries a scale that is not positive and differs from the scale prev stores for it. A
// zero scale sizes a symbol to nothing. A stored scale that the write keeps passes, so
// a stored zero does not block an edit to another field.
func validateScales(prev, next Schematic) error {
	for key, cfg := range next.Configs {
		scale, ok, err := scaleOf(cfg)
		if err != nil {
			return err
		}
		if !ok || scale > 0 {
			continue
		}
		if old, stored := prev.Configs[key]; stored {
			oldScale, hadScale, err := scaleOf(old)
			if err != nil {
				return err
			}
			if hadScale && oldScale == scale {
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
func scaleOf(cfg ElementConfig) (float64, bool, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return 0, false, err
	}
	var scaled struct {
		Scale *float64 `json:"scale"`
	}
	if err = json.Unmarshal(b, &scaled); err != nil {
		return 0, false, err
	}
	if scaled.Scale == nil {
		return 0, false, nil
	}
	return *scaled.Scale, true, nil
}
