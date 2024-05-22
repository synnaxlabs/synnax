// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"strconv"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

const (
	nodeOntologyType    ontology.Type = "node"
	clusterOntologyType ontology.Type = "cluster"
)

// NodeOntologyID returns a unique identifier for a Node to use within a resource
// Ontology.
func NodeOntologyID(key core.NodeKey) ontology.ID {
	return ontology.ID{Type: nodeOntologyType, Key: strconv.Itoa(int(key))}
}

// ClusterOntologyID returns a unique identifier for a Cluster to use with a
// resource Ontology.
func ClusterOntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: clusterOntologyType, Key: key.String()}
}

var (
	_nodeSchema = &ontology.Schema{
		Type: nodeOntologyType,
		Fields: map[string]schema.Field{
			"key":     {Type: schema.Uint32},
			"address": {Type: schema.String},
			"state":   {Type: schema.Uint32},
		},
	}
	_clusterSchema = &ontology.Schema{
		Type: clusterOntologyType,
		Fields: map[string]schema.Field{
			"key": {Type: schema.String},
		},
	}
)

// NodeOntologyService implements the ontology.Service interface to provide resource access
// to a cluster's nodes.
type NodeOntologyService struct {
	alamos.Instrumentation
	Ontology *ontology.Ontology
	Cluster  core.Cluster
}

var _ ontology.Service = (*NodeOntologyService)(nil)

// ListenForChanges starts listening for changes to the cluster topology (nodes leaving,
// joining, changing state, etc.) and updates the ontolgoy accordinly.
func (s *NodeOntologyService) ListenForChanges(ctx context.Context) {
	if err := s.Ontology.NewWriter(nil).DefineResource(ctx, NodeOntologyID(core.Free)); err != nil {
		s.L.Error("failed to define free node ontology resource", zap.Error(err))
	}
	s.update(ctx, s.Cluster.PeekState())
	s.Cluster.OnChange(func(ctx context.Context, change core.ClusterChange) {
		s.update(ctx, change.State)
	})
}

// OnChange implements ontology.Service.
func (s *NodeOntologyService) OnChange(f func(context.Context, iter.Nexter[schema.Change])) observe.Disconnect {
	var (
		translate = func(ch core.NodeChange, _ int) schema.Change {
			return schema.Change{
				Variant: ch.Variant,
				Key:     NodeOntologyID(ch.Key),
				Value:   newNodeResource(ch.Value),
			}
		}
		onChange = func(ctx context.Context, ch core.ClusterChange) {
			f(ctx, iter.All(lo.Map(ch.Changes, translate)))
		}
	)
	return s.Cluster.OnChange(onChange)
}

// OpenNexter implements ontology.Service.
func (s *NodeOntologyService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	return iter.NexterNopCloser(
		iter.All(lo.MapToSlice(s.Cluster.PeekState().Nodes, func(_ core.NodeKey, n core.Node) ontology.Resource {
			return newNodeResource(n)
		})),
	), nil
}

func (s *NodeOntologyService) update(ctx context.Context, state core.ClusterState) {
	err := s.Ontology.DB.WithTx(ctx, func(txn gorp.Tx) error {
		//w := s.Ontology.NewWriter(txn)
		//clusterID := ClusterOntologyID(s.Cluster.Key())
		//if err := w.DefineResource(ctx, clusterID); err != nil {
		//	return err
		//}
		//if err := w.DefineRelationship(ctx, ontology.RootID, ontology.ParentOf, clusterID); err != nil {
		//	return err
		//}
		//for _, n := range state.Nodes {
		//	nodeID := NodeOntologyID(n.Key)
		//	if err := w.DefineResource(ctx, NodeOntologyID(n.Key)); err != nil {
		//		return err
		//	}
		//	if err := w.DefineRelationship(ctx, clusterID, ontology.ParentOf, nodeID); err != nil {
		//		return err
		//	}
		//}
		return nil
	})
	if err != nil {
		s.L.Error("failed to update node ontology", zap.Error(err))
	}
}

// Schema implements ontology.Service.
func (s *NodeOntologyService) Schema() *schema.Schema { return _nodeSchema }

// RetrieveResource implements ontology.Service.
func (s *NodeOntologyService) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (ontology.Resource, error) {
	_nKey, err := strconv.Atoi(key)
	if err != nil {
		return schema.Resource{}, err
	}
	nKey := core.NodeKey(_nKey)
	if nKey.IsFree() {
		return newNodeResource(core.Node{Key: nKey}), nil
	}
	n, err := s.Cluster.Node(nKey)
	return newNodeResource(n), err
}

func newNodeResource(n core.Node) schema.Resource {
	r := schema.NewResource(
		_nodeSchema,
		NodeOntologyID(n.Key),
		fmt.Sprintf("Node %v", n.Key),
	)
	schema.Set(r, "key", uint32(n.Key))
	schema.Set(r, "address", n.Address.String())
	schema.Set(r, "state", uint32(n.State))
	return r
}

// OntologyService implements the ontology.Service to provide resource access
// to metadata about a Cluster.
type OntologyService struct {
	Cluster core.Cluster
	// Nothing will ever change about the cluster.
	observe.Noop[iter.Nexter[schema.Change]]
}

var _ ontology.Service = (*OntologyService)(nil)

// Schema implements ontology.Service.
func (s *OntologyService) Schema() *schema.Schema { return _clusterSchema }

// RetrieveResource implements ontology.Service.
func (s *OntologyService) RetrieveResource(context.Context, string, gorp.Tx) (ontology.Resource, error) {
	return newClusterResource(s.Cluster.Key()), nil
}

// OpenNexter implements ontology.Service.Relationship
func (s *OntologyService) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	return iter.NexterNopCloser(
		iter.All[schema.Resource]([]schema.Resource{
			//newClusterResource(s.Cluster.Key()),
		}),
	), nil
}

func newClusterResource(key uuid.UUID) ontology.Resource {
	r := schema.NewResource(_clusterSchema, ClusterOntologyID(key), "Cluster")
	schema.Set(r, "key", key.String())
	return r
}
