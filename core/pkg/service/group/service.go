// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"
	"io"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// MoveMembers moves the records in groups that belong to the top parent of their group,
// in tx, after groups move under the top parent top. groups holds each moved group and
// every group under it.
type MoveMembers func(
	ctx context.Context,
	tx gorp.Tx,
	groups []Key,
	top *resource.ID,
) error

type ServiceConfig struct {
	alamos.Instrumentation
	DB *gorp.DB
	// MoveMembers runs in each write that moves groups under a new top parent. Nil
	// means no record belongs to the top parent of its group.
	MoveMembers MoveMembers
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements ServiceConfig.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.MoveMembers = override.Nil(c.MoveMembers, other.MoveMembers)
	return c
}

// Validate implements ServiceConfig.
func (c ServiceConfig) Validate() error {
	v := validate.New("group")
	v.NotNil("db", c.DB)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	signals io.Closer
	table   *gorp.Table[Key, Group]
	indexes indexes
	state   *actions.State[Key, Action]
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:     cfg,
		indexes: newIndexes(),
		state:   actions.NewState[Key, Action](cfg.DB),
	}
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Group]{
		DB:              cfg.DB,
		Name:            tableName,
		Indexes:         s.indexes.all(),
		References:      s.indexes.references(),
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// BuiltinDeleteMigration returns a migration that deletes every group with the builtin
// scope. The service layer runs it after the ontology drop.
func (s *Service) BuiltinDeleteMigration() gorp.Migration {
	return migrate.New(
		"v61_delete_builtin_groups",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return s.table.NewDelete().
				Where(gorp.Match(func(_ gorp.Context, g *Group) (bool, error) {
					return g.Scope == resource.TypeBuiltin, nil
				})).
				Exec(ctx, tx)
		},
	)
}

// CreateOrRetrieve returns the top-level group of scope named name, creating it when no
// such group exists.
func (s *Service) CreateOrRetrieve(
	ctx context.Context,
	name string,
	scope resource.Type,
) (Group, error) {
	var g Group
	if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.NewRetrieve().
			Where(MatchNames(name)).
			Where(MatchScopes(scope)).
			Where(MatchHasParent(false)).
			Entry(&g).
			Exec(ctx, tx)
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		g = Group{Name: name, Scope: scope}
		return s.NewWriter(tx).Create(ctx, &g)
	}); err != nil {
		return Group{}, err
	}
	return g, nil
}

// Changes returns the changes that c made to groups.
func (s *Service) Changes(
	ctx context.Context,
	c gorp.Commit,
) gorp.TxReader[Key, Group] {
	return s.table.Changes(ctx, c)
}

// Actions returns the action batches that Writer attaches to its transactions.
func (s *Service) Actions() *actions.State[Key, Action] { return s.state }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         tx,
		table:      s.table,
		onMove:     s.cfg.MoveMembers,
		dispatcher: s.state.Dispatcher(),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}

// Ancestors returns every group above the groups with keys that is not one of them,
// each once. The walk stops at a parent that is not a group.
func (s *Service) Ancestors(
	ctx context.Context,
	snap gorp.Snapshot,
	keys ...Key,
) ([]Group, error) {
	var (
		out     []Group
		level   []Group
		visited = set.New(keys...)
	)
	if err := s.NewRetrieve().
		Where(MatchKeys(keys...)).
		Entries(&level).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	for {
		var parents []Key
		for _, g := range level {
			if g.Parent == nil || g.Parent.Type != resource.TypeGroup {
				continue
			}
			key, err := KeyFromResourceID(*g.Parent)
			if err != nil {
				return nil, err
			}
			if !visited.Contains(key) {
				visited.Add(key)
				parents = append(parents, key)
			}
		}
		if len(parents) == 0 {
			return out, nil
		}
		level = nil
		if err := s.NewRetrieve().
			Where(MatchKeys(parents...)).
			Entries(&level).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		out = append(out, level...)
	}
}

// Children returns the ID of every record and group that sits in the group with key, as
// seen by snap.
func (s *Service) Children(
	ctx context.Context,
	snap gorp.Snapshot,
	key Key,
) ([]resource.ID, error) {
	return resource.Dependents(ctx, snap, s.table, key)
}

func (s *Service) Close() error {
	if s.signals != nil {
		return errors.Combine(s.signals.Close(), s.table.Close())
	}
	return s.table.Close()
}

type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, Group]
	onMove     MoveMembers
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates g, assigning a new key when g.Key is zero. A group with a parent takes
// the parent's scope, and a scope that differs from the parent's is a validation error.
// The commit carries a create action that holds the stored group. Create returns an
// error wrapping validate.ErrValidation when a group with g.Key exists.
func (w Writer) Create(ctx context.Context, g *Group) error {
	if g.Key == uuid.Nil() {
		g.Key = uuid.New()
	}
	if g.Parent != nil {
		scope, err := w.parentScope(ctx, *g.Parent)
		if err != nil {
			return err
		}
		if g.Scope == "" {
			g.Scope = scope
		}
		if err = checkScope(*g, scope); err != nil {
			return err
		}
	}
	if err := g.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(g).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, g.Key, NewCreateAction(CreatePayload{Group: *g}))
	return nil
}

// Apply applies the actions to the group with the given key and attaches them to the
// transaction of w, so its commit broadcasts them with the dispatch key of w. A set
// parent action follows the rules of SetParent. Apply returns an error wrapping
// validate.ErrValidation when an action is a create, the result fails to decode or
// validate, or a move breaks a rule, and an error wrapping query.ErrNotFound when no
// group has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var (
		top   *resource.ID
		moved bool
	)
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Group](key)).
		ChangeErr(func(ctx gorp.Context, g Group) (Group, error) {
			next, err := Reduce(g, acts...)
			if err != nil {
				return Group{}, err
			}
			if err = next.Validate(); err != nil {
				return Group{}, err
			}
			if sameID(g.Parent, next.Parent) {
				return next, nil
			}
			var scope resource.Type
			if next.Parent != nil {
				if scope, err = w.parentScope(ctx, *next.Parent); err != nil {
					return Group{}, err
				}
			}
			if top, moved, err = w.checkMove(ctx, g, next.Parent); err != nil {
				return Group{}, err
			}
			if next.Parent != nil {
				if err = checkScope(next, scope); err != nil {
					return Group{}, err
				}
			}
			return next, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if moved {
		if err := w.moveMembers(ctx, []Key{key}, top); err != nil {
			return err
		}
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// SetParent moves the groups with keys under parent, or to the top level when parent is
// nil, by applying a set parent action to each. Each group's scope must match the scope
// of parent. A group that holds a rack, device, or project cannot move under a new top
// parent. A move under a new top parent runs the MoveMembers hook.
func (w Writer) SetParent(ctx context.Context, parent *resource.ID, keys ...Key) error {
	act := NewSetParentAction(SetParentPayload{Parent: parent})
	for _, k := range keys {
		if err := w.Apply(ctx, k, act); err != nil {
			return err
		}
	}
	return nil
}

// parentScope returns the scope a group takes under parent: the scope of a parent
// group, project for a project, and device for a rack.
func (w Writer) parentScope(
	ctx context.Context,
	parent resource.ID,
) (resource.Type, error) {
	switch parent.Type {
	case resource.TypeProject:
		return resource.TypeProject, nil
	case resource.TypeRack:
		return resource.TypeDevice, nil
	case resource.TypeGroup:
		key, err := KeyFromResourceID(parent)
		if err != nil {
			return "", validate.PathedError(err, "parent")
		}
		var p Group
		if err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Group](key)).
			Entry(&p).
			Exec(ctx, w.tx); err != nil {
			if errors.Is(err, query.ErrNotFound) {
				return "", validate.PathedError(errors.Wrapf(
					validate.ErrValidation, "group %s does not exist", key,
				), "parent")
			}
			return "", err
		}
		return p.Scope, nil
	}
	return "", validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"must be a group, project, or rack, got %q",
		parent.Type,
	), "parent")
}

func checkScope(g Group, parentScope resource.Type) error {
	if g.Scope == parentScope {
		return nil
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"a %s group cannot sit under a %s parent",
		g.Scope,
		parentScope,
	), "parent")
}

// Delete deletes the groups with keys. It returns a validation error when a group holds
// a member or a group that keys does not name.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	named := set.New(
		lo.Map(keys, func(k Key, _ int) string { return gorp.FormatKey(k) })...,
	)
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Group](keys...)).
		Guard(func(ctx gorp.Context, g Group) error {
			deps, err := w.table.Dependents(ctx, ctx.Snapshot, g.Key)
			if err != nil {
				return err
			}
			for _, d := range deps {
				if d.Table == string(resource.TypeGroup) && named.Contains(d.Key) {
					continue
				}
				return errors.Wrapf(
					validate.ErrValidation,
					"cannot delete group %s because it is not empty",
					g.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx)
}

// Rename renames the group with the given key by applying a set of the name.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}
