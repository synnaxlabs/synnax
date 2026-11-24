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
	"encoding/json/v2"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// DecodeImExEnvelope materializes env's body as a current-version Schematic, keyless
// and named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Schematic, error) {
	var (
		sch Schematic
		err error
	)
	if env.Version > legacy.LastVersion {
		sch, err = autoDecodeEnvelope(ctx, env)
	} else {
		var s7 v7.Schematic
		if s7, err = decodeConsole(ctx, env); err == nil {
			sch, err = liftFirst(ctx, s7)
		}
	}
	if err != nil {
		return Schematic{}, err
	}
	if sch, err = positiveScales(sch); err != nil {
		return Schematic{}, err
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	sch.Key = uuid.Nil()
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	sch.Name = env.Name
	return sch, nil
}

// decodeConsole decodes env, a file that the Console wrote, as the first exported
// shape.
func decodeConsole(ctx context.Context, env imex.Envelope) (v7.Schematic, error) {
	body, err := env.Decode[msgpack.EncodedJSON]()
	if err != nil {
		return v7.Schematic{}, err
	}
	if env.Version == legacy.LastVersion {
		// The v0.56 Console export: the typed schematic it retrieved from the Core,
		// written back out in camelCase under the Console's own version stamp.
		if err = imex.RequireFields(
			body,
			"a schematic",
			"nodes",
			"configs",
		); err != nil {
			return v7.Schematic{}, err
		}
		var doc legacy.Export
		if doc, err = env.Decode[legacy.Export](); err != nil {
			return v7.Schematic{}, err
		}
		return v7.SchematicFromConsole(doc), nil
	}
	// Console states embed the document inline: ride the storage lift, which dispatches
	// on the version stamped inside the body.
	if err = imex.RequireFields(body, "a schematic", "nodes", "props"); err != nil {
		return v7.Schematic{}, err
	}
	return v7.MigrateSchematic(ctx, v0.Schematic{Name: env.Name, Data: body})
}

// positiveScales sets every scale of sch that is not positive to 1, as the
// v59_positive_scale migration does for stored schematics. A Core before that migration
// could export a zero scale, which sizes a symbol to nothing.
func positiveScales(sch Schematic) (Schematic, error) {
	for i := range sch.Nodes {
		cfg := sch.Nodes[i].Config
		b, err := json.Marshal(cfg)
		if err != nil {
			return Schematic{}, err
		}
		var fields map[string]any
		if err = json.Unmarshal(b, &fields); err != nil {
			return Schematic{}, err
		}
		scale, ok := fields["scale"].(float64)
		if !ok || scale > 0 {
			continue
		}
		fields["scale"] = 1.0
		if b, err = json.Marshal(fields); err != nil {
			return Schematic{}, err
		}
		if err = json.Unmarshal(b, &cfg); err != nil {
			return Schematic{}, err
		}
		sch.Nodes[i].Config = cfg
	}
	return sch, nil
}
