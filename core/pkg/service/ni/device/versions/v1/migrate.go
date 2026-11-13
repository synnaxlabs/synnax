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
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v0"
)

// Migration lifts stored NI devices from v0 to v1 and drops their DAQmx alias. A
// simulated device moves its serial, which held its NI-MAX resource name, to its
// resource name and gets an empty serial. A physical device keeps its serial and gets
// an empty resource name until the scanner writes its identity. Until then, the device
// fails validation, so a configuration update to it fails.
var Migration = migrate.NewEntry("v62_ni_device_identity", migrateDevice)

func migrateDevice(ctx context.Context, old v0.Device) (Device, error) {
	d, err := autoMigrateDevice(ctx, old)
	if err != nil || !old.Simulated {
		return d, err
	}
	d.ResourceName = old.Serial
	d.Serial = ""
	return d, nil
}
