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
	"github.com/synnaxlabs/synnax/pkg/service/group/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
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
	DB       *gorp.DB
	Ontology *ontology.Ontology
	// MoveMembers runs in each write that moves groups under a new top parent. Nil
	// means no record belongs to the top parent of its group.
	MoveMembers MoveMembers
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements ServiceConfig.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.MoveMembers = override.Nil(c.MoveMembers, other.MoveMembers)
	return c
}

// Validate implements ServiceConfig.
func (c ServiceConfig) Validate() error {
	v := validate.New("group")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	signals io.Closer
	table   *gorp.Table[Key, Group]
	indexes indexes
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, indexes: newIndexes()}
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
	cfg.Ontology.RegisterService(s)
	return s, nil
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

// Observe returns an observable that notifies callers of changes to group entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Group]] {
	return s.table.Observe()
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		svc:       s,
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		table:     s.table,
		onMove:    s.cfg.MoveMembers,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		baseTX:  s.cfg.DB,
		gorp:    s.table.NewRetrieve(),
		indexes: s.indexes,
	}
}

// Ancestors returns every group above the groups with keys on their parent chains, each
// once. The walk stops at a parent that is not a group.
func (s *Service) Ancestors(
	ctx context.Context,
	tx gorp.Tx,
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
		Exec(ctx, tx); err != nil {
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
			Exec(ctx, tx); err != nil {
			return nil, err
		}
		out = append(out, level...)
	}
}

// Children returns the ID of every record and group that sits in the group with key, as
// seen by tx.
func (s *Service) Children(
	ctx context.Context,
	tx gorp.Tx,
	key Key,
) ([]resource.ID, error) {
	return resource.Dependents(ctx, gorp.OverrideTx(s.cfg.DB, tx), s.table, key)
}

func (s *Service) Close() error {
	if s.signals != nil {
		return errors.Combine(s.signals.Close(), s.table.Close())
	}
	return s.table.Close()
}

type Writer struct {
	svc       *Service
	tx        gorp.Tx
	otgWriter ontology.Writer
	table     *gorp.Table[Key, Group]
	onMove    MoveMembers
}

// inTx calls f with w, or with a Writer on a transaction of its own when w has no
// transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	db, ok := w.tx.(*gorp.DB)
	if !ok {
		return f(w)
	}
	return db.WithTx(ctx, func(tx gorp.Tx) error { return f(w.svc.NewWriter(tx)) })
}

// Create creates g, assigning a new key when g.Key is zero. A group with a parent takes
// the parent's scope, and a scope that differs from the parent's is a validation error.
// An existing group with g.Key keeps its stored parent and scope; only SetParent moves
// a group.
func (w Writer) Create(ctx context.Context, g *Group) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, g) })
}

func (w Writer) create(ctx context.Context, g *Group) error {
	if g.Key == uuid.Nil() {
		g.Key = uuid.New()
	}
	var existing Group
	err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Group](g.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	if errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}
	if err == nil {
		g.Parent, g.Scope = existing.Parent, existing.Scope
	} else if g.Parent != nil {
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
	if err = g.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(g).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DefineResources(ctx, g.ResourceID())
}

// SetParent moves the groups with keys under parent, or to the top level when parent is
// nil. Each group's scope must match the scope of parent. A group that holds a rack,
// device, or project cannot move under a new top parent. A move under a new top parent
// runs the MoveMembers hook.
func (w Writer) SetParent(ctx context.Context, parent *resource.ID, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		return w.setParent(ctx, parent, keys...)
	})
}

func (w Writer) setParent(ctx context.Context, parent *resource.ID, keys ...Key) error {
	var scope resource.Type
	if parent != nil {
		var err error
		if scope, err = w.parentScope(ctx, *parent); err != nil {
			return err
		}
	}
	var (
		top   *resource.ID
		moved []Key
	)
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Group](keys...)).
		ChangeErr(func(ctx gorp.Context, g Group) (Group, error) {
			after, changed, err := w.checkMove(ctx, g, parent)
			if err != nil {
				return g, err
			}
			if changed {
				top, moved = after, append(moved, g.Key)
			}
			g.Parent = parent
			if parent == nil {
				return g, nil
			}
			return g, checkScope(g, scope)
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.moveMembers(ctx, moved, top)
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
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Group](keys...)).
		Guard(func(ctx gorp.Context, g Group) error {
			deps, err := w.table.Dependents(ctx, ctx.Tx, g.Key)
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
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, ResourceIDs(keys)...)
}

// Rename renames the Group with the given key.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Group](key)).
		ChangeErr(func(_ gorp.Context, g Group) (Group, error) {
			g.Name = name
			return g, g.Validate()
		}).
		Exec(ctx, w.tx)
}
