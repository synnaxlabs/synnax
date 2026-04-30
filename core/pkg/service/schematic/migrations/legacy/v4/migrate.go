// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Lift decodes the opaque schematic data blob as a v4.Data, recursing into
// v3.Lift on older blobs and running Migrate on the result.
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
			return Data{}, errors.Wrap(err, "decode v4 schematic data")
		}
		return d, nil
	}
	prior, err := v3.Lift(blob)
	if err != nil {
		return Data{}, err
	}
	return Migrate(prior)
}

// Migrate transforms v3 schematic data into v4 by adding the default
// authority of 1. Mirrors the console's v3 -> v4 step.
func Migrate(old v3.Data) (Data, error) {
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
		Legend:          old.Legend,
		Key:             old.Key,
		Type:            old.Type,
		ViewportMode:    old.ViewportMode,
		Authority:       1,
	}, nil
}
