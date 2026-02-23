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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Codec    gorp.Codec[Group]
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements ServiceConfig.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Codec = override.Nil(c.Codec, other.Codec)
	return c
}

// Validate implements ServiceConfig.
func (c ServiceConfig) Validate() error {
	v := validate.New("group")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

type Service struct {
	cfg     ServiceConfig
	signals io.Closer
	table   *gorp.Table[uuid.UUID, Group]
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	table, err := gorp.OpenTable[uuid.UUID, Group](ctx, gorp.TableConfig[Group]{
		DB:    cfg.DB,
		Codec: cfg.Codec,
		Migrations: []gorp.Migration{
			gorp.NewCodecTransition[uuid.UUID, Group]("msgpack_to_protobuf", cfg.Codec),
		},
	})
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg, table: table}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

func (s *Service) CreateOrRetrieve(ctx context.Context, groupName string, parent ontology.ID) (Group, error) {
	var g Group
	err := s.NewRetrieve().Entry(&g).WhereNames(groupName).Exec(ctx, nil)
	if errors.Skip(err, query.ErrNotFound) != nil {
		return Group{}, err
	}
	w := s.NewWriter(nil)
	if errors.Is(err, query.ErrNotFound) {
		return w.Create(ctx, groupName, parent)
	}
	return w.CreateWithKey(ctx, g.Key, groupName, parent)
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.cfg.DB, tx), otg: s.cfg.Ontology.NewWriter(tx), table: s.table}
}

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.cfg.DB, s.table)
}

func (s *Service) Observe() observe.Observable[gorp.TxReader[uuid.UUID, Group]] {
	return s.table.Observe()
}

func (s *Service) Close() error {
	if s.signals != nil {
		return errors.Combine(s.signals.Close(), s.table.Close())
	}
	return s.table.Close()
}

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	table *gorp.Table[uuid.UUID, Group]
}

// Create creates a new Group with the given name and parent.
func (w Writer) Create(
	ctx context.Context,
	name string,
	parent ontology.ID,
) (g Group, err error) {
	g.Key = uuid.New()
	g.Name = name
	id := OntologyID(g.Key)
	if err = w.table.NewCreate().Entry(&g).Exec(ctx, w.tx); err != nil {
		return
	}
	if err = w.otg.DefineResource(ctx, id); err != nil {
		return
	}
	if err = w.otg.DefineRelationship(ctx, parent, ontology.RelationshipTypeParentOf, id); err != nil {
		return
	}
	return g, err
}

func (w Writer) CreateWithKey(
	ctx context.Context,
	key uuid.UUID,
	name string,
	parent ontology.ID,
) (g Group, err error) {
	g.Key = key
	if key == uuid.Nil {
		g.Key = uuid.New()
	}
	g.Name = name
	id := OntologyID(g.Key)
	if err = w.table.NewCreate().Entry(&g).Exec(ctx, w.tx); err != nil {
		return
	}
	if err = w.otg.DefineResource(ctx, id); err != nil {
		return
	}
	if err = w.otg.DefineRelationship(ctx, parent, ontology.RelationshipTypeParentOf, id); err != nil {
		return
	}
	return g, err
}

// Delete deletes the Groups with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...uuid.UUID) error {
	keyStrings := lo.Map(keys, func(item uuid.UUID, _ int) string {
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
			return errors.Wrap(validate.ErrValidation, "cannot delete a group with children")
		}
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return w.table.NewDelete().WhereKeys(keys...).Exec(ctx, w.tx)
}

// Rename renames the Group with the given key.
func (w Writer) Rename(ctx context.Context, key uuid.UUID, name string) error {
	return w.table.NewUpdate().
		WhereKeys(key).
		Change(func(_ gorp.Context, g Group) Group {
			g.Name = name
			return g
		}).
		Exec(ctx, w.tx)
}
