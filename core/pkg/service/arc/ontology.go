// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// ResourceID returns unique identifier for the Arc within the ontology.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeArc, Key: k.String()}
}

// ResourceIDs returns unique identifiers for the Arcs within the ontology.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return ResourceID(key) })
}

// KeysFromResourceIDs extracts the keys of the arcs from the resource IDs.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
}

// ResourceIDsFromArcs returns the resource IDs of the arcs.
func ResourceIDsFromArcs(arcs []Arc) []resource.ID {
	return lo.Map(arcs, func(a Arc, _ int) resource.ID { return a.ResourceID() })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"type": zyn.String(),
})

func newResource(a Arc) ontology.Resource {
	b := a.Base()
	return ontology.NewResource(schema, a.ResourceID(), b.Name, struct {
		Key  Key
		Type Type
	}{Key: b.Key, Type: a.Type()})
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[Key, Arc]

func (s *Service) Type() resource.Type { return resource.TypeArc }

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
	var arc Arc
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&arc).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(arc), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     ResourceID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Arc]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return resource.SearchIndex(ctx, tx, s.indexes.search, ResourceID, q)
}
