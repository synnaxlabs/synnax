// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"context"
	"maps"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
)

// Resources returns the resource.Provider of the nodes that the cluster holds when each
// of its methods is called. The free node always exists.
func (s *Service) Resources() resource.Provider { return provider{svc: s} }

type provider struct{ svc *Service }

// Search implements resource.Provider.
func (p provider) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return p.svc.Search(ctx, tx, q)
}

// Retrieve implements resource.Provider.
func (p provider) Retrieve(
	_ context.Context,
	_ gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	nodes := p.svc.cfg.Cluster.CopyState().Nodes
	resources := make([]resource.Resource, 0, len(ids))
	for _, id := range ids {
		k, err := strconv.ParseUint(id.Key, 10, 16)
		if err != nil {
			return nil, err
		}
		key := Key(k)
		if _, ok := nodes[key]; ok || key.IsFree() {
			resources = append(
				resources,
				resource.Resource{ID: ResourceID(key), Name: name(key)},
			)
		}
	}
	return resources, nil
}

// List implements resource.Provider. It lists the nodes in order of key.
func (p provider) List(
	_ context.Context,
	_ gorp.Tx,
	limit int,
) ([]resource.Resource, error) {
	keys := slices.Sorted(maps.Keys(p.svc.cfg.Cluster.CopyState().Nodes))
	return lo.Map(keys[:min(limit, len(keys))], func(key Key, _ int) resource.Resource {
		return resource.Resource{ID: ResourceID(key), Name: name(key)}
	}), nil
}
