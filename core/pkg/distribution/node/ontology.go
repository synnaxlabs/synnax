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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
	"go.uber.org/zap"
)

const OntologyType ontology.Type = "node"

// OntologyID returns a unique identifier for a Node to use within a resource
// Ontology.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: strconv.Itoa(int(key))}
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":     zyn.Uint16().Coerce(),
	"address": zyn.String(),
	"state":   zyn.Uint32().Coerce(),
})

// OntologyService implements the ontology.Service interface to provide resource access
// to a cluster's nodes.
type OntologyService struct {
	Cluster  Cluster
	Ontology *ontology.Ontology
	alamos.Instrumentation
}

var _ ontology.Service = (*OntologyService)(nil)

func (s *OntologyService) Type() ontology.Type { return OntologyType }

// ListenForChanges starts listening for changes to the cluster topology (nodes leaving,
// joining, changing state, etc.) and updates the ontology accordingly.
func (s *OntologyService) ListenForChanges(ctx context.Context) {
	if err := s.Ontology.NewWriter(nil).DefineResource(ctx, OntologyID(KeyFree)); err != nil {
		s.L.Error("failed to define free node ontology resource", zap.Error(err))
	}
}

func translateChange(ch NodeChange, _ int) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *OntologyService) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	onChange := func(ctx context.Context, ch Change) {
		f(ctx, slices.Values(lo.Map(ch.Changes, translateChange)))
	}
	return s.Cluster.OnChange(onChange)
}

// OpenNexter implements ontology.Service.
func (s *OntologyService) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values(lo.MapToSlice(s.Cluster.CopyState().Nodes, func(_ Key, n Node) ontology.Resource {
		return newResource(n)
	})), xio.NopCloser, nil
}

// Schema implements ontology.Service.
func (s *OntologyService) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *OntologyService) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (ontology.Resource, error) {
	_nKey, err := strconv.Atoi(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	nKey := Key(_nKey)
	if nKey.IsFree() {
		return newResource(Node{Key: nKey}), nil
	}
	n, err := s.Cluster.Node(nKey)
	if err != nil {
		return ontology.Resource{}, err
	}
	return newResource(n), nil
}

func newResource(n Node) ontology.Resource {
	return ontology.NewResource(
		schema,
		OntologyID(n.Key),
		fmt.Sprintf("Node %v", n.Key),
		n,
	)
}
