// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v3"
)

// Version is the numeric version for console schematic state v5.0.0.
// Computed via legacyToNumeric: 5*5 + 0*2 + 0 = 25.
const Version = 25

type ToolbarData struct {
	ActiveTab           string `json:"activeTab"`
	SelectedSymbolGroup string `json:"selectedSymbolGroup"`
}

// Data holds schematic content at version 5.0.0. Adds mode and toolbar fields
// (UI-only, not used for server content).
type Data struct {
	Nodes         []v0.NodeData              `json:"nodes"`
	Edges         []v3.EdgeData              `json:"edges"`
	Props         map[string]json.RawMessage `json:"props"`
	Legend        v1.LegendData              `json:"legend"`
	Snapshot      bool                       `json:"snapshot"`
	RemoteCreated bool                       `json:"remote_created"`
	Key           string                     `json:"key"`
	Type          string                     `json:"type"`
	Authority     float64                    `json:"authority"`
	Mode          string                     `json:"mode"`
	Toolbar       ToolbarData                `json:"toolbar"`
}
