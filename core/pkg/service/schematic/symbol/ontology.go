// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns unique identifier for the symbol within the ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeSchematicSymbol, Key: k.String()}
}

// OntologyIDs returns unique identifiers for the symbols within the ontology.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID { return OntologyID(key) })
}

// KeysFromOntologyIDs extracts the keys of the symbols from the ontology IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		return uuid.Parse(id.Key)
	})
}

// OntologyIDsFromSymbols returns the ontology IDs of the symbols.
func OntologyIDsFromSymbols(symbols []Symbol) []ontology.ID {
	return lo.Map(symbols, func(s Symbol, _ int) ontology.ID { return s.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID(), "name": zyn.String()})

func newResource(s Symbol) ontology.Resource {
	return ontology.NewResource(schema, s.OntologyID(), s.Name, s)
}

type change = xchange.Change[Key, Symbol]

var _ ontology.Service = (*Service)(nil)

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType {
	return ontology.ResourceTypeSchematicSymbol
}

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var symbol Symbol
	if err = s.NewRetrieve().
		Where(MatchKeys(k)).
		Entry(&symbol).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(symbol), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Symbol]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.indexes.search, term, limit, OntologyID)
}
