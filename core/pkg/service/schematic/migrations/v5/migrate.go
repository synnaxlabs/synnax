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
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v4"
)

// Migrate transforms v4 schematic data into v5 by adding mode and toolbar fields.
func Migrate(old v4.Data) (Data, error) {
	return Data{
		Nodes:         old.Nodes,
		Edges:         old.Edges,
		Props:         old.Props,
		Legend:        old.Legend,
		Snapshot:      old.Snapshot,
		RemoteCreated: old.RemoteCreated,
		Key:           old.Key,
		Type:          old.Type,
		Authority:     old.Authority,
		Mode:          "select",
		Toolbar: ToolbarData{
			ActiveTab:           "symbols",
			SelectedSymbolGroup: "general",
		},
	}, nil
}
