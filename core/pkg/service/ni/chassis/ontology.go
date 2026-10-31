// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns the ontology ID of the chassis under key.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeNiChassis, Key: key.String()}
}

// OntologyIDs returns the ontology IDs of the chassis under keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(key Key, _ int) ontology.ID { return OntologyID(key) })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":    zyn.UUID(),
	"rack":   zyn.Uint32().Coerce(),
	"name":   zyn.String(),
	"serial": zyn.String(),
	"model":  zyn.String(),
})

func newResource(c Chassis) ontology.Resource {
	return ontology.NewResource(schema, c.OntologyID(), c.Name, c)
}

func translateChange(c change.Change[Key, Chassis]) ontology.Change {
	out := ontology.Change{Variant: c.Variant, Key: OntologyID(c.Key).String()}
	// A delete carries only its key, and the zero value does not pass the schema.
	if c.Variant == change.VariantSet {
		out.Value = newResource(c.Value)
	}
	return out
}

var _ ontology.Service = (*Service)(nil)

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeNiChassis }

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
	var c Chassis
	if err = s.NewRetrieve().
		Where(MatchKeys(k)).
		Entry(&c).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(c), nil
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	return s.table.Observe().OnChange(
		func(ctx context.Context, reader gorp.TxReader[Key, Chassis]) {
			f(ctx, xiter.Map(reader, translateChange))
		},
	)
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
