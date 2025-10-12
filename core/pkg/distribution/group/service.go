// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate implements Config.
func (c Config) Validate() error {
	v := validate.New("group")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	return v.Error()
}

type Service struct {
	Config
	signals io.Closer
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

func (s *Service) CreateOrRetrieve(ctx context.Context, groupName string, parent ontology.ID) (g Group, err error) {
	err = s.NewRetrieve().Entry(&g).WhereNames(groupName).Exec(ctx, nil)
	w := s.NewWriter(nil)
	if errors.Is(err, query.NotFound) {
		return w.Create(ctx, groupName, parent)
	}
	return w.CreateWithKey(ctx, g.Key, groupName, parent)
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.DB, tx), otg: s.Ontology.NewWriter(tx)}
}

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.DB)
}

func (s *Service) Close() error {
	if s.signals != nil {
		return s.signals.Close()
	}
	return nil
}

type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
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
	if err = gorp.NewCreate[uuid.UUID, Group]().Entry(&g).Exec(ctx, w.tx); err != nil {
		return
	}
	if err = w.otg.DefineResource(ctx, id); err != nil {
		return
	}
	if err = w.otg.DefineRelationship(ctx, parent, ontology.ParentOf, id); err != nil {
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
	if err = gorp.NewCreate[uuid.UUID, Group]().Entry(&g).Exec(ctx, w.tx); err != nil {
		return
	}
	if err = w.otg.DefineResource(ctx, id); err != nil {
		return
	}
	if err = w.otg.DefineRelationship(ctx, parent, ontology.ParentOf, id); err != nil {
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
			TraverseTo(ontology.Children).
			ExcludeFieldData(true).
			Entries(&children).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		children = lo.Filter(children, func(item ontology.Resource, index int) bool {
			return !lo.Contains(keyStrings, item.ID.Key)
		})
		if len(children) > 0 {
			return errors.Wrap(validate.Error, "cannot delete a group with children")
		}
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return gorp.NewDelete[uuid.UUID, Group]().WhereKeys(keys...).Exec(ctx, w.tx)
}

// Rename renames the Group with the given key.
func (w Writer) Rename(ctx context.Context, key uuid.UUID, name string) error {
	return gorp.NewUpdate[uuid.UUID, Group]().
		WhereKeys(key).
		Change(func(_ gorp.Context, g Group) Group {
			g.Name = name
			return g
		}).
		Exec(ctx, w.tx)
}
