// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const ontologyType ontology.Type = "table"

// OntologyID returns unique identifier for the table within the ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the tables within the ontology.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(key uuid.UUID, _ int) ontology.ID {
		return OntologyID(key)
	})
}

// OntologyIDsFromTables returns the ontology IDs of the tables.
func OntologyIDsFromTables(tables []Table) []ontology.ID {
	return lo.Map(tables, func(l Table, _ int) ontology.ID {
		return OntologyID(l.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newResource(t Table) ontology.Resource {
	return core.NewResource(schema, OntologyID(t.Key), t.Name, t)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Table]

func (s *Service) Type() ontology.Type { return ontologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := uuid.MustParse(key)
	var table Table
	err := s.NewRetrieve().WhereKeys(k).Entry(&table).Exec(ctx, tx)
	return newResource(table), err
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Table]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, Table](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Table](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Table, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
