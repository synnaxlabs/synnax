// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v5"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
)

func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	parent ontology.ID,
	env imex.Envelope,
) error {
	migrated, err := s.migrateData(env.Version, env.Data)
	if err != nil {
		return err
	}
	key, err := uuid.Parse(env.Key)
	if err != nil {
		key = uuid.New()
	}
	wsKey, err := uuid.Parse(parent.Key)
	if err != nil {
		return err
	}
	name := env.Name
	if name == "" {
		name = "Imported Schematic"
	}
	schematic := convertToSchematic(key, name, migrated)
	return s.NewWriter(tx).Create(ctx, wsKey, &schematic)
}

func (s *Service) Export(
	ctx context.Context,
	tx gorp.Tx,
	key string,
) (imex.Envelope, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var sc Schematic
	if err := s.NewRetrieve().WhereKeys(k).Entry(&sc).Exec(ctx, tx); err != nil {
		return imex.Envelope{}, err
	}
	data := make(map[string]any)
	data["key"] = sc.Key.String()
	data["name"] = sc.Name
	data["snapshot"] = sc.Snapshot
	data["authority"] = float64(sc.Authority)
	data["nodes"] = nodesToAny(sc.Nodes)
	data["edges"] = edgesToAny(sc.Edges)
	data["props"] = propsToAny(sc.Props)
	data["legend"] = legendToAny(sc.Legend)
	return imex.Envelope{
		Type: string(ontology.ResourceTypeSchematic),
		Key:  sc.Key.String(),
		Name: sc.Name,
		Data: data,
	}, nil
}

func (s *Service) migrateData(version int, data map[string]any) (v5.Data, error) {
	switch {
	case version >= v5.Version:
		var d v5.Data
		if err := v5.Schema.Parse(data, &d); err != nil {
			return v5.Data{}, err
		}
		return d, nil
	case version >= v4.Version:
		var d v4.Data
		if err := v4.Schema.Parse(data, &d); err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(d)
	case version >= v3.Version:
		var d v3.Data
		if err := v3.Schema.Parse(data, &d); err != nil {
			return v5.Data{}, err
		}
		m4, err := v4.Migrate(d)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(m4)
	case version >= v2.Version:
		var d v2.Data
		if err := v2.Schema.Parse(data, &d); err != nil {
			return v5.Data{}, err
		}
		m3, err := v3.Migrate(d)
		if err != nil {
			return v5.Data{}, err
		}
		m4, err := v4.Migrate(m3)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(m4)
	case version >= v1.Version:
		var d v1.Data
		if err := v1.Schema.Parse(data, &d); err != nil {
			return v5.Data{}, err
		}
		m2, err := v2.Migrate(d)
		if err != nil {
			return v5.Data{}, err
		}
		m3, err := v3.Migrate(m2)
		if err != nil {
			return v5.Data{}, err
		}
		m4, err := v4.Migrate(m3)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(m4)
	case version >= v0.Version:
		var d v0.Data
		if err := v0.Schema.Parse(data, &d); err != nil {
			return v5.Data{}, err
		}
		m1, err := v1.Migrate(d)
		if err != nil {
			return v5.Data{}, err
		}
		m2, err := v2.Migrate(m1)
		if err != nil {
			return v5.Data{}, err
		}
		m3, err := v3.Migrate(m2)
		if err != nil {
			return v5.Data{}, err
		}
		m4, err := v4.Migrate(m3)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(m4)
	default:
		return v5.Data{}, errors.Newf("unknown schematic data version %d", version)
	}
}

func convertToSchematic(key uuid.UUID, name string, d v5.Data) Schematic {
	nodes := make([]Node, len(d.Nodes))
	for i, n := range d.Nodes {
		nodes[i] = Node{
			Key:      n.Key,
			Position: spatial.XY{X: n.Position.X, Y: n.Position.Y},
			Measured: spatial.Dimensions{Width: n.Measured.Width, Height: n.Measured.Height},
		}
	}
	edges := make([]Edge, len(d.Edges))
	for i, e := range d.Edges {
		edges[i] = Edge{
			Key:    e.Key,
			Source: Handle{Node: e.Source.Node, Param: e.Source.Param},
			Target: Handle{Node: e.Target.Node, Param: e.Target.Param},
		}
	}
	props := make(map[string]msgpack.EncodedJSON, len(d.Props))
	for k, v := range d.Props {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		props[k] = msgpack.EncodedJSON(m)
	}
	legend := convertLegend(d.Legend)
	return Schematic{
		Key:       key,
		Name:      name,
		Snapshot:  d.Snapshot,
		Authority: control.Authority(d.Authority),
		Legend:    legend,
		Nodes:     nodes,
		Edges:     edges,
		Props:     props,
	}
}

func convertLegend(ld v1.LegendData) Legend {
	leg := Legend{
		Visible:  ld.Visible,
		Position: spatial.StickyXY{X: ld.Position.X, Y: ld.Position.Y},
	}
	if ld.Position.Units != nil {
		leg.Position.Units.X = ld.Position.Units["x"]
		leg.Position.Units.Y = ld.Position.Units["y"]
	}
	if ld.Position.Root != nil {
		leg.Position.Root.X = ld.Position.Root["x"]
		leg.Position.Root.Y = ld.Position.Root["y"]
	}
	if ld.Colors != nil {
		leg.Colors = make(map[string]color.Color, len(ld.Colors))
		for k, v := range ld.Colors {
			leg.Colors[k] = color.MustFromHex(v)
		}
	}
	return leg
}

func nodesToAny(nodes []Node) []any {
	result := make([]any, len(nodes))
	for i, n := range nodes {
		result[i] = map[string]any{
			"key":      n.Key,
			"position": map[string]any{"x": n.Position.X, "y": n.Position.Y},
		}
	}
	return result
}

func edgesToAny(edges []Edge) []any {
	result := make([]any, len(edges))
	for i, e := range edges {
		result[i] = map[string]any{
			"key":    e.Key,
			"source": map[string]any{"node": e.Source.Node, "param": e.Source.Param},
			"target": map[string]any{"node": e.Target.Node, "param": e.Target.Param},
		}
	}
	return result
}

func propsToAny(props map[string]msgpack.EncodedJSON) map[string]any {
	result := make(map[string]any, len(props))
	for k, v := range props {
		result[k] = map[string]any(v)
	}
	return result
}

func legendToAny(leg Legend) map[string]any {
	colors := make(map[string]any, len(leg.Colors))
	for k, v := range leg.Colors {
		colors[k] = v.Hex()
	}
	return map[string]any{
		"visible": leg.Visible,
		"position": map[string]any{
			"x": leg.Position.X,
			"y": leg.Position.Y,
		},
		"colors": colors,
	}
}
