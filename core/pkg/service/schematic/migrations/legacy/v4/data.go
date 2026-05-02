// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v4 holds the frozen wire format for console schematic state at version
// 4.0.0. v4 added the per-schematic authority field.
package v4

import (
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v3"
)

const Version = "4.0.0"

// Data is the persisted per-schematic state at version 4.0.0. Adds authority
// to v3.
type Data struct {
	Version         string                     `json:"version"`
	Editable        bool                       `json:"editable"`
	FitViewOnResize bool                       `json:"fitViewOnResize"`
	Snapshot        bool                       `json:"snapshot"`
	RemoteCreated   bool                       `json:"remoteCreated"`
	Viewport        v0.Viewport                `json:"viewport"`
	Nodes           []v0.Node                  `json:"nodes"`
	Edges           []v3.Edge                  `json:"edges"`
	Props           map[string]json.RawMessage `json:"props"`
	Control         string                     `json:"control"`
	Legend          v1.Legend                  `json:"legend"`
	Key             string                     `json:"key"`
	Type            string                     `json:"type"`
	ViewportMode    string                     `json:"viewportMode"`
	Authority       float64                    `json:"authority"`
}
