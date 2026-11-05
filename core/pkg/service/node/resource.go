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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
)

var _ resource.Provider = (*Service)(nil)

// Search implements resource.Provider. It searches the nodes that the cluster holds
// when Search is called.
func (s *Service) Search(
	_ context.Context,
	_ gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	nodes := s.cfg.Cluster.CopyState().Nodes
	idx := search.New[Key]()
	for _, key := range slices.Sorted(maps.Keys(nodes)) {
		idx.Set(key, name(key))
	}
	res := idx.Search(search.Query[Key]{
		Term:   q.Term,
		Except: func(key Key) bool { return q.Except != nil && q.Except(ResourceID(key)) },
		Limit:  q.Limit,
	})
	return search.MapKeys(res, ResourceID), nil
}

// RetrieveResources implements resource.Provider. It reads the nodes that the cluster
// holds when RetrieveResources is called. The free node always exists.
func (s *Service) RetrieveResources(
	_ context.Context,
	_ gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	nodes := s.cfg.Cluster.CopyState().Nodes
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
