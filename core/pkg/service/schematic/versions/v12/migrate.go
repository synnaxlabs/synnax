// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v12

import (
	"context"
	"encoding/json/v2"

	"github.com/synnaxlabs/gorp/migrate"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	"github.com/synnaxlabs/x/set"
)

// MigrateSchematic lifts a v11 schematic into v12 by moving each entry of the configs
// map onto the node or edge it configures. A node with no node config draws nothing, so
// it is dropped, and so is an edge whose source or target is dropped. An edge with no
// edge config gets the default pipe, which the Console drew for it before v0.55.
func MigrateSchematic(ctx context.Context, old v11.Schematic) (Schematic, error) {
	s, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	kept := make(set.Set[string], len(s.Nodes))
	nodes := s.Nodes[:0]
	for _, n := range s.Nodes {
		ok, err := convert[NodeConfig, EdgeConfig](old.Configs, n.Key, &n.Config)
		if err != nil {
			return Schematic{}, err
		}
		if ok {
			nodes = append(nodes, n)
			kept.Add(n.Key)
		}
	}
	s.Nodes = nodes
	edges := s.Edges[:0]
	for _, e := range s.Edges {
		if !kept.Contains(e.Source.Node) || !kept.Contains(e.Target.Node) {
			continue
		}
		ok, err := convert[EdgeConfig, NodeConfig](old.Configs, e.Key, &e.Config)
		if err != nil {
			return Schematic{}, err
		}
		if !ok {
			e.Config = EdgeConfig{
				Variant: PipeEdgeConfig{
					SegmentedEdgeConfig: SegmentedEdgeConfig{Segments: []Segment{}},
				},
			}
		}
		edges = append(edges, e)
	}
	s.Edges = edges
	return s, nil
}

// MigrateNode copies the node. MigrateSchematic sets its config from the configs map.
func MigrateNode(ctx context.Context, old v11.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// MigrateEdge copies the edge. MigrateSchematic sets its config from the configs map.
func MigrateEdge(ctx context.Context, old v11.Edge) (Edge, error) {
	return autoMigrateEdge(ctx, old)
}

// convert decodes the config stored under key into dst. It returns false when no
// config is stored under key, or when the stored config is of the Other element kind.
func convert[C, Other any](
	configs map[string]v11.ElementConfig,
	key string,
	dst *C,
) (bool, error) {
	cfg, ok := configs[key]
	if !ok || cfg.Variant == nil {
		return false, nil
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return false, err
	}
	if err = json.Unmarshal(b, dst); err == nil {
		return true, nil
	}
	var other Other
	if json.Unmarshal(b, &other) == nil {
		return false, nil
	}
	return false, err
}

var Migration = migrate.NewEntry("v62_configs_on_nodes", MigrateSchematic)
