// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"
	"iter"
	"strconv"

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

func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(key Key, _ int) resource.ID { return key.ResourceID() })
}

func ResourceIDsFromRacks(racks []Rack) []resource.ID {
	return lo.Map(racks, func(r Rack, _ int) resource.ID { return r.ResourceID() })
}

func KeyFromResourceID(id resource.ID) (Key, error) {
	k, err := strconv.ParseUint(id.Key, 10, 32)
	if err != nil {
		return 0, err
	}
	return Key(k), nil
}

func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.Uint32().Coerce(),
	"name": zyn.String(),
})

func newResource(r Rack) ontology.Resource {
	return ontology.NewResource(schema, r.ResourceID(), r.Name, r)
}

type change = xchange.Change[Key, Rack]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() resource.Type { return resource.TypeRack }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := strconv.ParseUint(key, 10, 32)
	if err != nil {
		return ontology.Resource{}, err
	}
	var r Rack
	if err = s.NewRetrieve().
		Where(MatchKeys(Key(k))).
		Entry(&r).
		Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(r), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     c.Key.ResourceID().String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Rack]) {
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
	return ontology.SearchIndex(ctx, tx, s.indexes.search, Key.ResourceID, q)
}
