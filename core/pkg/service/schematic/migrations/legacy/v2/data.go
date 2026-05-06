// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v2 holds the frozen wire format for console schematic state at version
// 2.0.0. v2 added per-schematic key, type literal, and viewportMode fields.
package v2

import (
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
)

const Version = "2.0.0"

// Data is the persisted per-schematic state at version 2.0.0.
type Data struct {
	Version         string                     `json:"version"`
	Editable        bool                       `json:"editable"`
	FitViewOnResize bool                       `json:"fitViewOnResize"`
	Snapshot        bool                       `json:"snapshot"`
	RemoteCreated   bool                       `json:"remoteCreated"`
	Viewport        v0.Viewport                `json:"viewport"`
	Nodes           []v0.Node                  `json:"nodes"`
	Edges           []v0.Edge                  `json:"edges"`
	Props           map[string]json.RawMessage `json:"props"`
	Control         string                     `json:"control"`
	Legend          v1.Legend                  `json:"legend"`
	Key             string                     `json:"key"`
	Type            string                     `json:"type"`
	ViewportMode    string                     `json:"viewportMode"`
}
