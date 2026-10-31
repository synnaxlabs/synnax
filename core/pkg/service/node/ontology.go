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
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns a unique identifier for a Node to use within a resource Ontology.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeNode, Key: strconv.Itoa(int(key))}
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":     zyn.Uint16().Coerce(),
	"address": zyn.String(),
	"state":   zyn.Uint32().Coerce(),
})

var _ ontology.Service = (*Service)(nil)

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeNode }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	_ context.Context,
	key string,
	_ gorp.Tx,
) (ontology.Resource, error) {
	intKey, err := strconv.ParseUint(key, 10, 16)
	if err != nil {
		return ontology.Resource{}, err
	}
	nKey := Key(intKey)
	if nKey.IsFree() {
		return newResource(Node{Key: nKey}), nil
	}
	n, err := s.cfg.Cluster.Node(nKey)
	if err != nil {
		return ontology.Resource{}, err
	}
	return newResource(n), nil
}

// Search implements ontology.Service. It searches the nodes that the cluster holds
// when Search is called.
func (s *Service) Search(
	_ context.Context,
	_ gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	nodes := s.cfg.Cluster.CopyState().Nodes
	idx := search.New[Key]()
	for _, key := range slices.Sorted(maps.Keys(nodes)) {
		idx.Set(key, name(key))
	}
	return search.MapKeys(idx.Search(term, limit), OntologyID), nil
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	onChange := func(ctx context.Context, ch cluster.Change) {
		f(ctx, slices.Values(lo.Map(ch.Changes, translateChange)))
	}
	return s.cfg.Cluster.OnChange(onChange)
}

func translateChange(ch Change, _ int) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key).String(),
		Value:   newResource(ch.Value),
	}
}

func name(key Key) string { return fmt.Sprintf("Node %v", key) }

func newResource(n Node) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(n.Key), name(n.Key), n)
}
