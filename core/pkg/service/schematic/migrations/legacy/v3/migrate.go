// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Lift decodes the opaque schematic data blob as a v3.Data, recursing into
// v2.Lift on older blobs and running Migrate on the result.
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
			return Data{}, errors.Wrap(err, "decode v3 schematic data")
		}
		return d, nil
	}
	prior, err := v2.Lift(blob)
	if err != nil {
		return Data{}, err
	}
	return Migrate(prior)
}

// Migrate transforms v2 schematic data into v3 by attaching an empty segments
// slice to every edge. Mirrors the console's v2 -> v3 step.
func Migrate(old v2.Data) (Data, error) {
	edges := make([]Edge, len(old.Edges))
	for i, e := range old.Edges {
		edges[i] = upgradeEdge(e)
	}
	return Data{
		Version:         Version,
		Editable:        old.Editable,
		FitViewOnResize: old.FitViewOnResize,
		Snapshot:        old.Snapshot,
		RemoteCreated:   old.RemoteCreated,
		Viewport:        old.Viewport,
		Nodes:           old.Nodes,
		Edges:           edges,
		Props:           old.Props,
		Control:         old.Control,
		Legend:          old.Legend,
		Key:             old.Key,
		Type:            old.Type,
		ViewportMode:    old.ViewportMode,
	}, nil
}

func upgradeEdge(e v0.Edge) Edge {
	return Edge{
		Key:          e.Key,
		Source:       e.Source,
		Target:       e.Target,
		SourceHandle: e.SourceHandle,
		TargetHandle: e.TargetHandle,
		Segments:     []Segment{},
	}
}
