// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/color"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
)

// ResourceID returns the unique ID to identify the range within the Synnax ontology.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeRange, Key: key.String()}
}

// ResourceIDs converts a slice of keys to a slice of resource IDs.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

func KeyFromResourceID(id resource.ID) (Key, error) { return resource.ParseUUIDKey(id) }

// KeysFromResourceIDs converts a slice of resource IDs to a slice of keys, returning an
// error if any of the IDs are invalid.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return KeyFromResourceID(id)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.UUID(),
	"name":       zyn.String(),
	"color":      color.Schema.Optional(),
	"time_range": telem.TimeRangeSchema,
})

func newResource(r Range) ontology.Resource {
	return ontology.NewResource(schema, r.ResourceID(), r.Name, r)
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[Key, Range]

func (s *Service) Type() resource.Type { return resource.TypeRange }

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
	var r Range
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&r).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(r), nil
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Range]) {
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
