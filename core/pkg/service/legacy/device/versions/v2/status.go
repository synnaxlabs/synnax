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
	"context"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
)

// isDefault returns whether s is a default status that the Core wrote for a device no
// Driver has reached. Only the Core writes these messages. Before "Device status
// unknown", the default message was the device name followed by " state unknown".
func isDefault(s Status) bool {
	return s.Variant == status.VariantWarning &&
		!s.Details.Connected &&
		(s.Message == "Device status unknown" ||
			strings.HasSuffix(s.Message, " state unknown"))
}

// NewUnknownStatusMigration gives each default device status the disabled variant,
// which the Core now writes for a device that no Driver has reached.
func NewUnknownStatusMigration(statuses *status.Service) gorp.Migration {
	return migrate.New(
		"v63_device_unknown_status_disabled",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var devices []Device
			if err := gorp.NewRetrieve[Key, Device]().
				Entries(&devices).
				Exec(ctx, tx); err != nil {
				return err
			}
			if len(devices) == 0 {
				return nil
			}
			var held []Status
			if err := statuses.NewRetrieve[StatusDetails]().
				Where(status.MatchOwners[StatusDetails](
					lo.Map(devices, func(d Device, _ int) resource.ID {
						return d.ResourceID()
					})...,
				)).
				Entries(&held).
				Exec(ctx, tx); err != nil {
				return err
			}
			w := statuses.NewMigrationWriter(tx)
			for _, s := range held {
				if !isDefault(s) {
					continue
				}
				s.Variant = status.VariantDisabled
				if err := w.Set(ctx, s.ToAny()); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
