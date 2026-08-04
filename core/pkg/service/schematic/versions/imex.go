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

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// lastStateVersion is the final Console state version ("6.0.0" files). A v6
// state parks the document under pendingUpload; earlier states embed it inline
// and ride the storage lift's legacy chain.
const lastStateVersion imex.Version = 6

// stateV6 is the slice of the v6 Console state the importer needs: the document
// body parked under pendingUpload when a schematic was never uploaded. The tag
// is camelCase because the file was written by the Console.
type stateV6 struct {
	PendingUpload *consoleDocument `json:"pendingUpload" msgpack:"pendingUpload"`
}

// consoleNode mirrors Node as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleNode struct {
	Key      string     `json:"key" msgpack:"key"`
	Position spatial.XY `json:"position" msgpack:"position"`
	ZIndex   int16      `json:"zIndex" msgpack:"zIndex"`
}

// consoleDocument mirrors the typed Schematic body fields as Console-written
// files serialize them: the typed export's top level, and the v6 state's
// pendingUpload. Configs values are opaque user JSON and pass through as-is.
type consoleDocument struct {
	Snapshot bool                           `json:"snapshot" msgpack:"snapshot"`
	Nodes    []consoleNode                  `json:"nodes" msgpack:"nodes"`
	Edges    []Edge                         `json:"edges" msgpack:"edges"`
	Configs  map[string]msgpack.EncodedJSON `json:"configs" msgpack:"configs"`
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

// DecodeImport materializes the envelope's body as a current-version Schematic.
// Envelopes stamped at or above Floor decode through the generated migration
// chain; older ones are Console-era files — camelCase typed exports or console
// states — and are lifted forward. An envelope newer than Latest is rejected
// with a path-scoped validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (Schematic, error) {
	if env.Version >= Floor {
		return decodeMigrate(ctx, env)
	}
	// Console-era typed exports ("6.0.0"-stamped or versionless) carry the current
	// shape with camelCase keys; console states never carry a name.
	if env.BodyNamed() {
		doc, err := imex.Decode[consoleDocument](ctx, env)
		if err != nil {
			return Schematic{}, err
		}
		return doc.schematic(), nil
	}
	if env.Version == lastStateVersion {
		st, err := imex.Decode[stateV6](ctx, env)
		if err != nil {
			return Schematic{}, err
		}
		if st.PendingUpload == nil {
			return Schematic{}, errors.Wrap(
				validate.ErrValidation, "schematic file has no document data",
			)
		}
		return st.PendingUpload.schematic(), nil
	}
	// v0-v5 console states embed the document inline: ride the storage lift, which
	// dispatches on the version string inside the body.
	snapshot, _ := env.Body()["snapshot"].(bool)
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Schematic{}, err
	}
	return v7.MigrateSchematic(ctx, v6.Schematic{
		Name: env.Name, Snapshot: snapshot, Data: body,
	})
}
