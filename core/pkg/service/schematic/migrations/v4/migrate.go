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
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v3"
)

// Migrate transforms v3 schematic data into v4 by adding a default authority of 1.
func Migrate(old v3.Data) (Data, error) {
	return Data{
		Nodes:         old.Nodes,
		Edges:         old.Edges,
		Props:         old.Props,
		Legend:        old.Legend,
		Snapshot:      old.Snapshot,
		RemoteCreated: old.RemoteCreated,
		Key:           old.Key,
		Type:          old.Type,
		Authority:     1,
	}, nil
}
