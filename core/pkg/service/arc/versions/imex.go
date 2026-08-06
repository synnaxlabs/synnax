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
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// DecodeImExEnvelope materializes env's body as a current-version Arc, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Arc, error) {
	var (
		a   Arc
		err error
	)
	if env.Version > legacy.LastVersion {
		a, err = autoDecodeEnvelope(ctx, env)
	} else {
		var doc legacy.Document
		if doc, err = decodeConsole(ctx, env); err == nil {
			a.Mode, a.Graph, a.Text = Mode(doc.Mode), doc.Graph, doc.Text
		}
	}
	if err != nil {
		return Arc{}, err
	}
	// Console-era files leave the mode blank when the Arc was never switched out of the
	// graph editor.
	if a.Mode == "" {
		a.Mode = ModeGraph
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	a.Key = uuid.Nil
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	a.Name = env.Name
	return a, nil
}

// decodeConsole lifts either Console-written file shape into a Document. Both shapes
// carry the graph at the top level, so the structural guard runs ahead of the split.
func decodeConsole(ctx context.Context, env imex.Envelope) (legacy.Document, error) {
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return legacy.Document{}, err
	}
	if err = imex.RequireFields(body, "an Arc", "graph"); err != nil {
		return legacy.Document{}, err
	}
	if !isConsoleState(body) {
		// The export taken from an Arc that was not open: the typed Arc the Console
		// retrieved from the Core, spread into the file.
		e, err := imex.Decode[legacy.Export](ctx, env)
		if err != nil {
			return legacy.Document{}, err
		}
		return legacy.MigrateExport(ctx, e)
	}
	// "0.0.0".."2.0.0" Console states embed the graph inline. Nothing newer exists: the
	// shipped Console never wrote a later state format.
	return legacy.MigrateData(body)
}

// isConsoleState reports whether body is a Console editor state rather than a typed Arc
// from the Core. Every released Console state carries graph.props; neither server
// payload does. The version header cannot decide it: Arcs stored before 0.54 carried a
// "0.0.0" version, so their exports look version-stamped too.
func isConsoleState(body msgpack.EncodedJSON) bool {
	g, ok := body["graph"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = g["props"]
	return ok
}
