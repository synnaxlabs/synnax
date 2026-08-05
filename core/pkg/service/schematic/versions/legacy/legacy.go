// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque schematic data blob
// through the chain of historical wire formats up to the latest legacy snapshot,
// v5.Data. Each subpackage v0..v5 owns a frozen Data shape and a single Migrate
// function that lifts the previous version's Data into its own; this package owns the
// version dispatch, the forward chain, and the re-export of the terminal model, so
// callers never reach past it into a version sub-package.
package legacy

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v6"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// LastVersion is the highest version a Console-written file carries. Envelopes stamped
// above it are server exports and route to the generated migration chain. v6 sits
// outside the state chain: it is the Console's own export format, not a stored state.
const LastVersion = v6.Version

// DataVersion is the version Data carries once the chain has run. It trails
// LastVersion because v6 is an export format rather than a chain step.
const DataVersion = v5.Version

// Data is the latest legacy snapshot; the migration chain terminates in it.
type Data = v5.Data

// Export is the Console's own export format, stamped at LastVersion. It sits outside
// the state chain: the Console wrote back the typed schematic it retrieved from the
// Core rather than a stored state.
type Export = v6.Data

type (
	// Edge connects two nodes, carrying the routing segments added at v3.
	Edge = v3.Edge
	// Node is a schematic node with its canvas position and stacking order.
	Node = v0.Node
	// Segment is one leg of an edge's routed path.
	Segment = v3.Segment
	// XY is a position on the canvas.
	XY = v0.XY
)

// MigrateData decodes the opaque schematic data blob, dispatches on its declared
// version, and walks the per-step Migrate functions forward to Data. A nil blob and a
// blob without a version field both fall through to v0 and walk the full chain. Orphan
// edges (empty source or target — persisted by ReactFlow after partial-drop
// interactions) are filtered after the chain runs since the condition is uniform across
// every legacy version.
func MigrateData(blob msgpack.EncodedJSON) (Data, error) {
	version, err := imex.PeekVersion(blob, "schematic data")
	if err != nil {
		return Data{}, err
	}
	var d Data
	switch version {
	case v5.Version:
		d, err = imex.DecodeBlob[v5.Data](blob, "schematic data", version)
	case v4.Version:
		var old v4.Data
		old, err = imex.DecodeBlob[v4.Data](blob, "schematic data", version)
		if err != nil {
			break
		}
		d = v5.Migrate(old)
	case v3.Version:
		var old v3.Data
		old, err = imex.DecodeBlob[v3.Data](blob, "schematic data", version)
		if err != nil {
			break
		}
		d = v5.Migrate(v4.Migrate(old))
	case v2.Version:
		var old v2.Data
		old, err = imex.DecodeBlob[v2.Data](blob, "schematic data", version)
		if err != nil {
			break
		}
		d = v5.Migrate(v4.Migrate(v3.Migrate(old)))
	case v1.Version:
		var old v1.Data
		old, err = imex.DecodeBlob[v1.Data](blob, "schematic data", version)
		if err != nil {
			break
		}
		d = v5.Migrate(
			v4.Migrate(v3.Migrate(v2.Migrate(old))),
		)
	case v0.Version:
		var old v0.Data
		old, err = imex.DecodeBlob[v0.Data](blob, "schematic data", version)
		if err != nil {
			break
		}
		d = v5.Migrate(
			v4.Migrate(
				v3.Migrate(v2.Migrate(v1.Migrate(old))),
			),
		)
	default:
		return Data{}, errors.Newf("unknown schematic data version %d", version)
	}
	if err != nil {
		return Data{}, err
	}
	d.Edges = slices.DeleteFunc(d.Edges, func(e v3.Edge) bool {
		return e.Source == "" || e.Target == ""
	})
	return d, nil
}
