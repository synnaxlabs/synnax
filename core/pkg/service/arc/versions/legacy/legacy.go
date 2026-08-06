// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating the Console-written Arc file
// formats into the typed document parts of an Arc: the state blobs ("0.0.0".."2.0.0"
// files) and the versionless v0.56 export. Each subpackage v0..v2 owns a frozen Data
// shape and a single Migrate function that lifts the previous version's Data into its
// own, and console owns the frozen export shape; this package owns the version
// dispatch, the forward chain, the re-export of the terminal model, and the final lift
// into Document.
package legacy

import (
	"context"
	"maps"

	"github.com/synnaxlabs/arc/graph"
	graphv1 "github.com/synnaxlabs/arc/graph/versions/v1"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/console"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v2"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// LastVersion is the highest version a Console-written file carries. Envelopes stamped
// above it are server exports and route to the generated migration chain.
const LastVersion = v2.Version

// Data is the latest legacy snapshot; the migration chain terminates in it.
type Data = v2.Data

// Export is the Console's own export format. It sits outside the state chain: the
// Console wrote back the typed Arc it retrieved from the Core, with no version stamp.
type Export = console.Data

type (
	// Edge connects two node handles, nested since v1.
	Edge = v1.Edge
	// Graph is the graph-mode body: nodes, edges, and per-node props.
	Graph = v1.Graph
	// Node is a graph node and its canvas position.
	Node = v0.Node
)

// Document is the typed body a Console file lifts into. Mode is the raw wire
// string ("graph" or "text"); the importer converts it to the typed enum.
type Document struct {
	Graph graph.Graph
	Text  text.Text
	Mode  string
}

// MigrateData decodes the opaque Arc state blob, dispatches on its declared version,
// walks the per-step Migrate functions forward to v2.Data, and lifts the result into a
// Document. A nil blob and a blob without a version field both fall through to v0 and
// walk the full chain.
func MigrateData(blob msgpack.EncodedJSON) (Document, error) {
	version, err := imex.PeekVersion(blob, "arc state")
	if err != nil {
		return Document{}, err
	}
	switch version {
	case v2.Version:
		d, err := imex.DecodeBlob[v2.Data](blob, "arc state", version)
		if err != nil {
			return Document{}, err
		}
		return lift(d), nil
	case v1.Version:
		d, err := imex.DecodeBlob[v1.Data](blob, "arc state", version)
		if err != nil {
			return Document{}, err
		}
		return lift(v2.Migrate(d)), nil
	case v0.Version:
		d, err := imex.DecodeBlob[v0.Data](blob, "arc state", version)
		if err != nil {
			return Document{}, err
		}
		return lift(v2.Migrate(v1.Migrate(d))), nil
	default:
		return Document{}, errors.Newf("unknown arc state version %d", version)
	}
}

// MigrateExport lifts the Console's own export into a Document, running the Arc graph
// migration that keys the export's edges and folds per-node config into inputs.
func MigrateExport(ctx context.Context, e Export) (Document, error) {
	g, err := graphv1.MigrateGraph(ctx, e.Graph.Lift())
	if err != nil {
		return Document{}, err
	}
	return Document{Graph: g, Text: e.Text, Mode: e.Mode}, nil
}

// lift converts the latest legacy snapshot into the typed graph, text, and mode
// parts of an Arc.
func lift(d v2.Data) Document {
	nodes := make(graph.Nodes, len(d.Graph.Nodes))
	for i, n := range d.Graph.Nodes {
		nodes[i] = graph.Node{Key: n.Key, Position: n.Position}
	}
	edges := make(graph.Edges, len(d.Graph.Edges))
	for i, e := range d.Graph.Edges {
		edges[i] = graph.Edge{
			Key: e.Key,
			Edge: ir.Edge{
				Source: e.Source,
				Target: e.Target,
				Kind:   ir.EdgeKindContinuous,
			},
		}
	}
	return Document{
		Graph: graph.Graph{
			Nodes:  nodes,
			Edges:  edges,
			Inputs: liftInputs(d.Graph.Props),
		},
		Text: text.Text{Raw: d.Text.Raw},
		Mode: d.Mode,
	}
}

// liftInputs converts per-node props (function type under "key") into the typed graph
// inputs map (function type under "type"), keyed by node key.
func liftInputs(props map[string]map[string]any) map[string]msgpack.EncodedJSON {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]msgpack.EncodedJSON, len(props))
	for k, p := range props {
		entry := make(map[string]any, len(p))
		maps.Copy(entry, p)
		entry["type"] = entry["key"]
		delete(entry, "key")
		out[k] = msgpack.EncodedJSON(entry)
	}
	return out
}
