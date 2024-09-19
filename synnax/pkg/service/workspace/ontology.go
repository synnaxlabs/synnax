// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

import (
	"context"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

const OntologyType ontology.Type = "workspace"

func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

func OntologyIDs(keys []uuid.UUID) (ids []ontology.ID) {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID {
		return OntologyID(k)
	})
}

func OntologyIDsFromWorkspaces(workspaces []Workspace) (ids []ontology.ID) {
	return lo.Map(workspaces, func(w Workspace, _ int) ontology.ID {
		return OntologyID(w.Key)
	})
}

func KeysFromOntologyIds(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var _schema = &ontology.Schema{
	Type: OntologyType,
	Fields: map[string]schema.Field{
		"key":  {Type: schema.String},
		"name": {Type: schema.String},
	},
}

func newResource(ws Workspace) schema.Resource {
	e := schema.NewResource(_schema, OntologyID(ws.Key), ws.Name)
	schema.Set(e, "key", ws.Key.String())
	schema.Set(e, "name", ws.Name)
	return e
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Workspace]

// Schema implements ontology.Service.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := uuid.MustParse(key)
	var schematic Workspace
	err := s.NewRetrieve().WhereKeys(k).Entry(&schematic).Exec(ctx, tx)
	return newResource(schematic), err
}

func translateChange(c change) schema.Change {
	return schema.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Workspace]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, Workspace](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Workspace](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Workspace, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
