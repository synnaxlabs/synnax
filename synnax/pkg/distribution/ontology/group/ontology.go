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
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

const ontologyType ontology.Type = "group"

func OntologyID(k uuid.UUID) schema.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

func OntologyIDs(keys []uuid.UUID) []schema.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) schema.ID { return OntologyID(k) })
}

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":  {Type: schema.UUID},
		"name": {Type: schema.String},
	},
}

func newResource(g Group) schema.Resource {
	r := schema.NewResource(_schema, OntologyID(g.Key), g.Name)
	schema.Set(r, "key", g.Key)
	schema.Set(r, "name", g.Name)
	return r
}

type change = changex.Change[uuid.UUID, Group]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Schema() *ontology.Schema {
	return _schema
}

func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (r ontology.Resource, err error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return r, err
	}
	var g Group
	err = s.NewRetrieve().Entry(&g).WhereKeys(k).Exec(ctx, tx)
	return newResource(g), err
}

func translateChange(c change) schema.Change {
	return schema.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Group]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, Group](s.DB).OnChange(handleChange)
}

func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Group](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Group, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
