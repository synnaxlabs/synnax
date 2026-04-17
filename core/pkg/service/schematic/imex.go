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
	"encoding/json"

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
	var wsKey uuid.UUID
	if !parent.IsZero() {
		wsKey, err = uuid.Parse(parent.Key)
		if err != nil {
			return err
		}
	}
	name := env.Name
	if name == "" {
		name = "Imported Schematic"
	}
	sc, err := convertToSchematic(key, name, migrated)
	if err != nil {
		return err
	}
	return s.NewWriter(tx).Create(ctx, wsKey, &sc)
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
	d, err := schematicToData(sc)
	if err != nil {
		return imex.Envelope{}, err
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return imex.Envelope{}, err
	}
	return imex.Envelope{
		Type: string(ontology.ResourceTypeSchematic),
		Key:  sc.Key.String(),
		Name: sc.Name,
		Data: raw,
	}, nil
}

func (s *Service) migrateData(version int, raw json.RawMessage) (v5.Data, error) {
	switch {
	case version >= v5.Version:
		var d v5.Data
		if err := imex.Decode(raw, &d); err != nil {
			return v5.Data{}, err
		}
		return d, nil
	case version >= v4.Version:
		var d v4.Data
		if err := imex.Decode(raw, &d); err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(d)
	case version >= v3.Version:
		var d v3.Data
		if err := imex.Decode(raw, &d); err != nil {
			return v5.Data{}, err
		}
		m4, err := v4.Migrate(d)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(m4)
	case version >= v2.Version:
		var d v2.Data
		if err := imex.Decode(raw, &d); err != nil {
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
		if err := imex.Decode(raw, &d); err != nil {
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
		if err := imex.Decode(raw, &d); err != nil {
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

func convertToSchematic(key uuid.UUID, name string, d v5.Data) (Schematic, error) {
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
			Source: Handle{Node: e.Source, Param: e.SourceHandle},
			Target: Handle{Node: e.Target, Param: e.TargetHandle},
		}
	}
	// Props are opaque per-variant JSON objects. Bridge from byte-preserved
	// RawMessage into msgpack.EncodedJSON (a map[string]any) at the storage
	// boundary. Byte fidelity end to end will arrive with a future change to
	// EncodedJSON.
	props := make(map[string]msgpack.EncodedJSON, len(d.Props))
	for k, raw := range d.Props {
		if len(raw) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return Schematic{}, errors.Wrapf(err, "decode props[%q]", k)
		}
		m["variant"] = m["key"]
		props[k] = m
	}
	return Schematic{
		Key:       key,
		Name:      name,
		Snapshot:  d.Snapshot,
		Authority: control.Authority(d.Authority),
		Legend:    convertLegend(d.Legend),
		Nodes:     nodes,
		Edges:     edges,
		Props:     props,
	}, nil
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

func schematicToData(sc Schematic) (v5.Data, error) {
	nodes := make([]v0.NodeData, len(sc.Nodes))
	for i, n := range sc.Nodes {
		nodes[i] = v0.NodeData{
			Key:      n.Key,
			Position: v0.XY{X: n.Position.X, Y: n.Position.Y},
			Measured: v0.Dimensions{Width: n.Measured.Width, Height: n.Measured.Height},
		}
	}
	edges := make([]v3.EdgeData, len(sc.Edges))
	for i, e := range sc.Edges {
		edges[i] = v3.EdgeData{
			Key:          e.Key,
			Source:       e.Source.Node,
			Target:       e.Target.Node,
			SourceHandle: e.Source.Param,
			TargetHandle: e.Target.Param,
			Segments:     []v3.SegmentData{},
		}
	}
	props := make(map[string]json.RawMessage, len(sc.Props))
	for k, v := range sc.Props {
		b, err := json.Marshal(map[string]any(v))
		if err != nil {
			return v5.Data{}, errors.Wrapf(err, "encode props[%q]", k)
		}
		props[k] = b
	}
	return v5.Data{
		Nodes:     nodes,
		Edges:     edges,
		Props:     props,
		Legend:    legendToData(sc.Legend),
		Snapshot:  sc.Snapshot,
		Key:       sc.Key.String(),
		Type:      string(ontology.ResourceTypeSchematic),
		Authority: float64(sc.Authority),
	}, nil
}

func legendToData(leg Legend) v1.LegendData {
	colors := make(map[string]string, len(leg.Colors))
	for k, v := range leg.Colors {
		colors[k] = v.Hex()
	}
	return v1.LegendData{
		Visible: leg.Visible,
		Position: v1.LegendPosition{
			X: leg.Position.X,
			Y: leg.Position.Y,
			Units: map[string]string{
				"x": leg.Position.Units.X,
				"y": leg.Position.Units.Y,
			},
			Root: map[string]string{
				"x": leg.Position.Root.X,
				"y": leg.Position.Root.Y,
			},
		},
		Colors: colors,
	}
}
