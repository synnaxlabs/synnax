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
	"io"
	"iter"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
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

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeNode }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (ontology.Resource, error) {
	intKey, err := strconv.Atoi(key)
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

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values(lo.MapToSlice(s.cfg.Cluster.CopyState().Nodes, func(_ Key, n Node) ontology.Resource {
		return newResource(n)
	})), xio.NopCloser, nil
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	onChange := func(ctx context.Context, ch ClusterChange) {
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

func newResource(n Node) ontology.Resource {
	return ontology.NewResource(
		schema,
		OntologyID(n.Key),
		fmt.Sprintf("Node %v", n.Key),
		n,
	)
}
