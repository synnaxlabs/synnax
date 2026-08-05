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
	legacyv6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v6"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// schematicFromConsole lifts the frozen Console v6 export into the current Schematic.
func schematicFromConsole(d legacyv6.Data) Schematic {
	nodes := make([]Node, len(d.Nodes))
	for i, n := range d.Nodes {
		nodes[i] = Node{Key: n.Key, Position: n.Position, ZIndex: n.ZIndex}
	}
	edges := make([]Edge, len(d.Edges))
	for i, e := range d.Edges {
		edges[i] = Edge{
			Key:    e.Key,
			Source: Handle{Node: e.Source.Node, Param: e.Source.Param},
			Target: Handle{Node: e.Target.Node, Param: e.Target.Param},
		}
	}
	return Schematic{
		Snapshot: d.Snapshot, Nodes: nodes, Edges: edges, Configs: d.Configs,
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
		var doc legacyv6.Data
		if doc, err = imex.Decode[legacyv6.Data](ctx, env); err == nil {
			sch = schematicFromConsole(doc)
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
