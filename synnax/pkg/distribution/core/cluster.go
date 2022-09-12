package core

import (
	"github.com/arya-analytics/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"strconv"
)

type (
	Node         = aspen.Node
	NodeID       = aspen.NodeID
	NodeState    = aspen.NodeState
	Cluster      = aspen.Cluster
	HostResolver = aspen.HostResolver
	Resolver     = aspen.Resolver
	ClusterState = aspen.ClusterState
)

const nodeResourceType = "node"

func NodeOntologyID(id NodeID) ontology.ID {
	return ontology.ID{Type: nodeResourceType, Key: strconv.Itoa(int(id))}
}
