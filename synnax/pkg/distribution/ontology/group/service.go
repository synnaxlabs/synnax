// Copyright 2023 Synnax Labs, Inc.
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
	"errors"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
}

var _ config.Config[Config] = Config{}

// Override implements Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements Config.
func (c Config) Validate() error {
	v := validate.New("group")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

type Service struct{ Config }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: tx}
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
	if err = w.validateNoChildrenWithName(ctx, name, parent); err != nil {
		return
	}
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

// Delete deletes the Groups with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...uuid.UUID) error {
	return gorp.NewDelete[uuid.UUID, Group]().
		WhereKeys(keys...).
		Exec(ctx, w.tx)
}

func (w Writer) AddChildren(ctx context.Context, group uuid.UUID, children ...ontology.ID) error {
	for _, child := range children {
		if err := w.otg.DefineRelationship(ctx, OntologyID(group), ontology.ParentOf, child); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) RemoveChildren(ctx context.Context, group uuid.UUID, children ...ontology.ID) error {
	for _, child := range children {
		if err := w.otg.DeleteRelationship(ctx, OntologyID(group), ontology.ParentOf, child); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) validateNoChildrenWithName(ctx context.Context, name string, parent ontology.ID) error {
	var children []ontology.Resource
	if err := w.otg.NewRetrieve().WhereIDs(parent).TraverseTo(ontology.Children).Entries(&children).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, child := range children {
		if child.Name == name {
			return errors.New("[group] - a child of the parent exists with the same name")
		}
	}
	return nil
}
