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
	"github.com/synnaxlabs/synnax/pkg/service/group/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	alamos.Instrumentation
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Search   *search.Index
	// IdentifierRoots names the groups directly below root under which every group
	// name must be an identifier: letters, digits, and underscores.
	IdentifierRoots []string
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements ServiceConfig.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	c.IdentifierRoots = override.Slice(c.IdentifierRoots, other.IdentifierRoots)
	return c
}

// Validate implements ServiceConfig.
func (c ServiceConfig) Validate() error {
	v := validate.New("group")
	v.NotNil("db", c.DB)
	v.NotNil("ontology", c.Ontology)
	v.NotNil("search", c.Search)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	signals io.Closer
	table   *gorp.Table[Key, Group]
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	table, err := gorp.OpenTable(ctx, gorp.TableConfig[Key, Group]{
		DB:              cfg.DB,
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	})
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, table: table}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	return s, nil
}

func (s *Service) CreateOrRetrieve(
	ctx context.Context,
	groupName string,
	parent ontology.ID,
) (Group, error) {
	var res Group
	if err := s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		g, err := s.retrieveChild(ctx, tx, groupName, parent)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		w := s.NewWriter(tx)
		if errors.Is(err, query.ErrNotFound) {
			res, err = w.Create(ctx, groupName, parent)
		} else {
			res, err = w.CreateWithKey(ctx, g.Key, groupName, parent)
		}
		return err
	}); err != nil {
		return Group{}, err
	}
	return res, nil
}

// retrieveChild returns the group with the given name whose parent is parent. A group
// with that name elsewhere in the tree does not match.
func (s *Service) retrieveChild(
	ctx context.Context,
	tx gorp.Tx,
	name string,
	parent ontology.ID,
) (Group, error) {
	var groups []Group
	if err := s.NewRetrieve().
		Entries(&groups).
		Where(MatchNames(name)).
		Exec(ctx, tx); err != nil {
		return Group{}, err
	}
	for _, g := range groups {
		isChild, err := s.cfg.Ontology.RelationshipExists(ctx, tx, ontology.Relationship{
			From: parent,
			Type: ontology.RelationshipTypeParentOf,
			To:   g.OntologyID(),
		})
		if err != nil || isChild {
			return g, err
		}
	}
	return Group{}, errors.Wrapf(query.ErrNotFound, "group %q under %s", name, parent)
}

// Observe returns an observable that notifies callers of changes to group entries.
func (s *Service) Observe() observe.Observable[gorp.TxReader[Key, Group]] {
	return s.table.Observe()
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:              gorp.OverrideTx(s.cfg.DB, tx),
		otgWriter:       s.cfg.Ontology.NewWriter(tx),
		otg:             s.cfg.Ontology,
		identifierRoots: set.New(s.cfg.IdentifierRoots...),
		table:           s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{baseTX: s.cfg.DB, gorp: s.table.NewRetrieve()}
}

func (s *Service) Close() error {
	if s.signals != nil {
		return errors.Combine(s.signals.Close(), s.table.Close())
	}
	return s.table.Close()
}

type Writer struct {
	tx              gorp.Tx
	otgWriter       ontology.Writer
	otg             *ontology.Ontology
	identifierRoots set.Set[string]
	table           *gorp.Table[Key, Group]
}

// validateName applies the identifier rule to name when the resource at id sits at or
// below one of the identifier roots.
func (w Writer) validateName(ctx context.Context, id ontology.ID, name string) error {
	below, err := w.belowIdentifierRoot(ctx, id)
	if err != nil || !below || validate.IsIdentifier(name) {
		return err
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"group name %q must contain only letters, digits, and underscores",
		name,
	)
}

// belowIdentifierRoot walks from id up to the root and reports whether id or one of
// its ancestors is an identifier root: a group with a configured name whose parent is
// root.
func (w Writer) belowIdentifierRoot(
	ctx context.Context,
	id ontology.ID,
) (bool, error) {
	if len(w.identifierRoots) == 0 {
		return false, nil
	}
	var (
		frontier = []ontology.ID{id}
		visited  = set.New(id)
	)
	for len(frontier) > 0 {
		keys := make([]Key, 0, len(frontier))
		for _, fid := range frontier {
			if fid.Type != ontology.ResourceTypeGroup {
				continue
			}
			key, err := uuid.Parse(fid.Key)
			if err != nil {
				return false, err
			}
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			var groups []Group
			if err := w.table.NewRetrieve().
				Where(gorp.MatchKeys[Key, Group](keys...)).
				Entries(&groups).
				Exec(ctx, w.tx); err != nil {
				return false, err
			}
			for _, g := range groups {
				if !w.identifierRoots.Contains(g.Name) {
					continue
				}
				atRoot, err := w.otg.RelationshipExists(ctx, w.tx, ontology.Relationship{
					From: ontology.RootID,
					Type: ontology.RelationshipTypeParentOf,
					To:   g.OntologyID(),
				})
				if err != nil || atRoot {
					return atRoot, err
				}
			}
		}
		var parents []ontology.Resource
		if err := w.otg.NewRetrieve().
			WhereIDs(frontier...).
			TraverseTo(ontology.ParentsTraverser).
			ExcludeFieldData(true).
			Entries(&parents).
			Exec(ctx, w.tx); err != nil {
			return false, err
		}
		frontier = frontier[:0]
		for _, p := range parents {
			if visited.Contains(p.ID) {
				continue
			}
			visited.Add(p.ID)
			frontier = append(frontier, p.ID)
		}
	}
	return false, nil
}

// Create creates a new Group with the given name and parent.
func (w Writer) Create(
	ctx context.Context,
	name string,
	parent ontology.ID,
) (Group, error) {
	if err := w.validateName(ctx, parent, name); err != nil {
		return Group{}, err
	}
	g := Group{Key: uuid.New(), Name: name}
	id := g.OntologyID()
	if err := w.table.NewCreate().Entry(&g).Exec(ctx, w.tx); err != nil {
		return Group{}, err
	}
	if err := w.otgWriter.DefineResources(ctx, id); err != nil {
		return Group{}, err
	}
	if err := w.otgWriter.DefineRelationships(
		ctx,
		parent,
		ontology.RelationshipTypeParentOf,
		id,
	); err != nil {
		return Group{}, err
	}
	return g, nil
}

func (w Writer) CreateWithKey(
	ctx context.Context,
	key Key,
	name string,
	parent ontology.ID,
) (Group, error) {
	if err := w.validateName(ctx, parent, name); err != nil {
		return Group{}, err
	}
	g := Group{Key: key, Name: name}
	if g.Key == uuid.Nil() {
		g.Key = uuid.New()
	}
	id := g.OntologyID()
	if err := w.table.NewCreate().Entry(&g).Exec(ctx, w.tx); err != nil {
		return Group{}, err
	}
	if err := w.otgWriter.DefineResources(ctx, id); err != nil {
		return Group{}, err
	}
	if err := w.otgWriter.DefineRelationships(
		ctx,
		parent,
		ontology.RelationshipTypeParentOf,
		id,
	); err != nil {
		return Group{}, err
	}
	return g, nil
}

// Delete deletes the Groups with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	keyStrings := lo.Map(keys, func(item Key, _ int) string {
		return item.String()
	})
	for _, key := range keys {
		var children []ontology.Resource
		if err := w.otg.NewRetrieve().
			WhereIDs(OntologyID(key)).
			TraverseTo(ontology.ChildrenTraverser).
			ExcludeFieldData(true).
			Entries(&children).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		children = lo.Filter(children, func(item ontology.Resource, index int) bool {
			return !lo.Contains(keyStrings, item.ID.Key)
		})
		if len(children) > 0 {
			return errors.Wrap(
				validate.ErrValidation,
				"cannot delete a group with children",
			)
		}
		if err := w.otgWriter.DeleteResources(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Group](keys...)).
		Exec(ctx, w.tx)
}

// Rename renames the Group with the given key.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	if err := w.validateName(ctx, OntologyID(key), name); err != nil {
		return err
	}
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Group](key)).
		Change(func(_ gorp.Context, g Group) Group {
			g.Name = name
			return g
		}).
		Exec(ctx, w.tx)
}
