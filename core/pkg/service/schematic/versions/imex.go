// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
)

// consoleNode mirrors Node as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleNode struct {
	// Key is the node's unique key.
	Key string `json:"key" msgpack:"key"`
	// Position is the node position on the canvas.
	Position spatial.XY `json:"position" msgpack:"position"`
	// ZIndex is the node stacking order.
	ZIndex int16 `json:"zIndex" msgpack:"zIndex"`
}

// consoleDocument mirrors the typed Schematic body fields as Console-written
// files serialize them at the typed export's top level. Configs values are
// opaque user JSON and pass through as-is.
type consoleDocument struct {
	// Snapshot marks the schematic as a range snapshot.
	Snapshot bool `json:"snapshot" msgpack:"snapshot"`
	// Nodes are the schematic nodes.
	Nodes []consoleNode `json:"nodes" msgpack:"nodes"`
	// Edges are the schematic edges.
	Edges []Edge `json:"edges" msgpack:"edges"`
	// Configs holds per-symbol configuration keyed by node key.
	Configs map[string]msgpack.EncodedJSON `json:"configs" msgpack:"configs"`
}

// schematic lifts the Console document into the current Schematic shape.
func (d consoleDocument) schematic() Schematic {
	nodes := make([]Node, len(d.Nodes))
	for i, n := range d.Nodes {
		nodes[i] = Node{Key: n.Key, Position: n.Position, ZIndex: n.ZIndex}
	}
	return Schematic{
		Snapshot: d.Snapshot, Nodes: nodes, Edges: d.Edges, Configs: d.Configs,
	}
}

// DecodeImExEnvelope materializes env's body as a current-version Schematic, keyless
// and named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Schematic, error) {
	var (
		sch Schematic
		err error
	)
	switch {
	case env.Version >= Floor:
		sch, err = decodeMigrate(ctx, env)
	case env.BodyNamed():
		// Console-era typed exports ("6.0.0"-stamped or versionless) carry the current
		// shape with camelCase keys; Console states never carry a name.
		var doc consoleDocument
		if doc, err = imex.Decode[consoleDocument](ctx, env); err == nil {
			sch = doc.schematic()
		}
	default:
		// Console states embed the document inline: ride the storage lift, which
		// dispatches on the version stamped inside the body.
		snapshot, _ := env.Body()["snapshot"].(bool)
		var body msgpack.EncodedJSON
		if body, err = imex.Decode[msgpack.EncodedJSON](ctx, env); err == nil {
			sch, err = v7.MigrateSchematic(ctx, v0.Schematic{
				Name: env.Name, Snapshot: snapshot, Data: body,
			})
		}
	}
	if err != nil {
		return Schematic{}, err
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	sch.Key = uuid.Nil
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	sch.Name = env.Name
	return sch, nil
}
