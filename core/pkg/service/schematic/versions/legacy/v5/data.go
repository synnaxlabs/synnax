// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v5 holds the frozen wire format for Console schematic state at version 5.0.0.
// v5 dropped the type literal and introduced the per-schematic mode and toolbar UI
// fields.
package v5

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 5

// Data is the persisted per-schematic state at version 5.0.0. Drops the type literal
// and adds mode and toolbar (UI-only fields that the typed Schematic does not carry).
type Data struct {
	// Version is the version stamped inside the blob.
	Version imex.Version `json:"version"`
	// Editable is UI-only edit-mode state; dropped on lift.
	Editable bool `json:"editable"`
	// FitViewOnResize is UI-only viewport behavior; dropped on lift.
	FitViewOnResize bool `json:"fitViewOnResize"`
	// Snapshot marks the schematic as a range snapshot.
	Snapshot bool `json:"snapshot"`
	// RemoteCreated is UI-only sync bookkeeping; dropped on lift.
	RemoteCreated bool `json:"remoteCreated"`
	// Viewport is the editor viewport position and zoom.
	Viewport v0.Viewport `json:"viewport"`
	// Nodes are the schematic nodes.
	Nodes []v0.Node `json:"nodes"`
	// Edges are the schematic edges.
	Edges []v3.Edge `json:"edges"`
	// Props holds per-symbol configuration keyed by node key.
	Props map[string]json.RawMessage `json:"props"`
	// Control is UI-only control-mode state; dropped on lift.
	Control string `json:"control"`
	// Legend is the control legend overlay configuration.
	Legend v1.Legend `json:"legend"`
	// Key is the Console-local schematic key.
	Key string `json:"key"`
	// ViewportMode is UI-only viewport interaction mode; dropped on lift.
	ViewportMode string `json:"viewportMode"`
	// Authority is the default control authority.
	Authority float64 `json:"authority"`
	// Mode is UI-only editor mode; dropped on lift.
	Mode string `json:"mode"`
	// Toolbar is UI-only toolbar state; dropped on lift.
	Toolbar v0.ToolbarState `json:"toolbar"`
}
