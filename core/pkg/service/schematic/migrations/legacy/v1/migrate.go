// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Lift decodes the opaque schematic data blob as a v1.Data, recursing into
// the previous version's Lift when the blob announces an older format and
// running Migrate on the result. Each version owns its own slice of the
// chain; the top-level migration only ever calls Lift on the latest
// shipped wire format.
func Lift(blob msgpack.EncodedJSON) (Data, error) {
	var peek struct {
		Version string `json:"version"`
	}
	if blob != nil {
		if err := blob.Unmarshal(&peek); err != nil {
			return Data{}, errors.Wrap(err, "peek schematic data version")
		}
	}
	if peek.Version == Version {
		var d Data
		if err := blob.Unmarshal(&d); err != nil {
			return Data{}, errors.Wrap(err, "decode v1 schematic data")
		}
		return d, nil
	}
	prior, err := v0.Lift(blob)
	if err != nil {
		return Data{}, err
	}
	return Migrate(prior)
}

// ZeroLegend is the default legend used when a v0 payload is migrated forward.
// Mirrors the console's ZERO_LEGEND_STATE at v1.
var ZeroLegend = Legend{
	Visible: true,
	Position: LegendPosition{
		X:     50,
		Y:     50,
		Units: &LegendUnits{X: "px", Y: "px"},
	},
	Colors: map[string]string{},
}

// Migrate transforms v0 schematic data into v1 by attaching the default legend.
func Migrate(old v0.Data) (Data, error) {
	return Data{
		Version:         Version,
		Editable:        old.Editable,
		FitViewOnResize: old.FitViewOnResize,
		Snapshot:        old.Snapshot,
		RemoteCreated:   old.RemoteCreated,
		Viewport:        old.Viewport,
		Nodes:           old.Nodes,
		Edges:           old.Edges,
		Props:           old.Props,
		Control:         old.Control,
		Legend:          ZeroLegend,
	}, nil
}
