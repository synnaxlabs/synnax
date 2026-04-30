// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"github.com/google/uuid"

	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Lift decodes the opaque schematic data blob as a v2.Data, recursing into
// v1.Lift on older blobs and running Migrate on the result.
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
			return Data{}, errors.Wrap(err, "decode v2 schematic data")
		}
		return d, nil
	}
	prior, err := v1.Lift(blob)
	if err != nil {
		return Data{}, err
	}
	return Migrate(prior)
}

// Migrate transforms v1 schematic data into v2 by adding the per-schematic
// key, the literal "schematic" type, and the default viewport mode. The
// generated key is a fresh UUID; the storage migration overwrites it with
// the gorp entry's key when it lifts the blob into the typed schematic.
func Migrate(old v1.Data) (Data, error) {
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
		Key:             uuid.New().String(),
		Type:            "schematic",
		ViewportMode:    "select",
	}, nil
}
