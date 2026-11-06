// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

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

// ResourceID constructs a unique resource.ID for the Policy with the given key.
func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypePolicy, Key: k.String()}
}

// ResourceIDs constructs a slice of unique resource.IDs for the Policys with the given
// keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromPolicies constructs a slice of unique resource.IDs for the given
// Policys.
func ResourceIDsFromPolicies(policies []Policy) []resource.ID {
	return lo.Map(policies, func(p Policy, _ int) resource.ID { return p.ResourceID() })
}

// KeysFromResourceIDs extracts the Policy keys from the given resource.IDs.
func KeysFromResourceIDs(ids []resource.ID) ([]Key, error) {
	return lo.MapErr(ids, func(id resource.ID, _ int) (Key, error) {
		return resource.ParseUUIDKey(id)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.UUID(),
	"name":     zyn.String(),
	"internal": zyn.Bool(),
})

func newResource(p Policy) ontology.Resource {
	return ontology.NewResource(schema, p.ResourceID(), p.Name, p)
}

type change = xchange.Change[Key, Policy]

func (s *Service) Type() resource.Type { return resource.TypePolicy }

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
	var p Policy
	if err := s.NewRetrieve().Where(MatchKeys(k)).Entry(&p).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(p), nil
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
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Policy]) {
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
