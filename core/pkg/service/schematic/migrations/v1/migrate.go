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
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
)

// Migrate transforms v0 schematic data into v1 by adding a default legend.
func Migrate(old v0.Data) (Data, error) {
	return Data{
		Nodes:         old.Nodes,
		Edges:         old.Edges,
		Props:         old.Props,
		Snapshot:      old.Snapshot,
		RemoteCreated: old.RemoteCreated,
		Legend: LegendData{
			Visible: true,
			Position: LegendPosition{
				X:     50,
				Y:     50,
				Units: map[string]string{"x": "px", "y": "px"},
			},
			Colors: map[string]string{},
		},
	}, nil
}
