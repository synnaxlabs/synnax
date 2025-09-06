// Copyright 2025 Synnax Labs, Inc.
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
	"strconv"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	ontologycore "github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
	"go.uber.org/zap"
)

const (
	nodeOntologyType    ontology.Type = "node"
	clusterOntologyType ontology.Type = "cluster"
)

// NodeOntologyID returns a unique identifier for a Node to use within a resource
// Ontology.
func NodeOntologyID(key NodeKey) ontology.ID {
	return ontology.ID{Type: nodeOntologyType, Key: strconv.Itoa(int(key))}
}

// OntologyID returns a unique identifier for a Cluster to use with a
// resource Ontology.
func OntologyID(key uuid.UUID) ontology.ID {
	return ontology.ID{Type: clusterOntologyType, Key: key.String()}
}

var (
	nodeSchema = zyn.Object(map[string]zyn.Schema{
		"key":     zyn.Uint16().Coerce(),
		"address": zyn.String(),
		"state":   zyn.Uint32().Coerce(),
	})
	schema = zyn.Object(map[string]zyn.Schema{"key": zyn.UUID()})
)

// NodeOntologyService implements the ontology.Service interface to provide resource access
// to a cluster's nodes.
type NodeOntologyService struct {
	alamos.Instrumentation
	Ontology *ontology.Ontology
	Cluster  Cluster
}

func (s *NodeOntologyService) Type() ontology.Type { return nodeOntologyType }

// ListenForChanges starts listening for changes to the cluster topology (nodes leaving,
// joining, changing state, etc.) and updates the ontology accordingly.
func (s *NodeOntologyService) ListenForChanges(ctx context.Context) {
	if err := s.Ontology.NewWriter(nil).DefineResource(ctx, NodeOntologyID(Free)); err != nil {
		s.L.Error("failed to define free node ontology resource", zap.Error(err))
	}
}

func translateNodeChange(ch NodeChange, _ int) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     NodeOntologyID(ch.Key),
		Value:   newNodeResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *NodeOntologyService) OnChange(f func(context.Context, iter.Nexter[ontology.Change])) observe.Disconnect {
	var (
		onChange = func(ctx context.Context, ch Change) {
			f(ctx, iter.All(lo.Map(ch.Changes, translateNodeChange)))
		}
	)
	return s.Cluster.OnChange(onChange)
}

// OpenNexter implements ontology.Service.
func (s *NodeOntologyService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	return iter.NexterNopCloser(
		iter.All(lo.MapToSlice(s.Cluster.CopyState().Nodes, func(_ NodeKey, n Node) ontology.Resource {
			return newNodeResource(n)
		})),
	), nil
}

// schema implements ontology.Service.
func (s *NodeOntologyService) Schema() zyn.Schema { return nodeSchema }

// RetrieveResource implements ontology.Service.
func (s *NodeOntologyService) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (ontology.Resource, error) {
	_nKey, err := strconv.Atoi(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	nKey := NodeKey(_nKey)
	if nKey.IsFree() {
		return newNodeResource(Node{Key: nKey}), nil
	}
	n, err := s.Cluster.Node(nKey)
	return newNodeResource(n), err
}

func newNodeResource(n Node) ontology.Resource {
	return ontologycore.NewResource(
		nodeSchema,
		NodeOntologyID(n.Key),
		fmt.Sprintf("Node %v", n.Key),
		n,
	)
}

// OntologyService implements the ontology.Service to provide resource access
// to metadata about a Cluster.
type OntologyService struct {
	Cluster Cluster
	// Nothing will ever change about the cluster.
	observe.Noop[iter.Nexter[ontology.Change]]
}

var _ ontology.Service = (*OntologyService)(nil)

func (s *OntologyService) Type() ontology.Type { return clusterOntologyType }

// schema implements ontology.Service.
func (s *OntologyService) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *OntologyService) RetrieveResource(context.Context, string, gorp.Tx) (ontology.Resource, error) {
	return newClusterResource(s.Cluster.Key()), nil
}

// OpenNexter implements ontology.Service.Relationship
func (s *OntologyService) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	return iter.NexterNopCloser(iter.All([]ontology.Resource{})), nil
}

func newClusterResource(key uuid.UUID) ontology.Resource {
	return ontologycore.NewResource(
		schema,
		OntologyID(key),
		"Cluster",
		map[string]any{"key": key},
	)
}
