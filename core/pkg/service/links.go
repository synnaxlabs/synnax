// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/errors"
)

const clearUnissuedKey = "v63_clear_unissued_channel_links"

// clearUnissuedChannelLinks returns the migration that removes, from every stored
// schematic, table, and Arc graph node input, each channel key that the Core had not
// issued at its first open. An import in 0.62 kept the raw keys of another Core, which
// a channel created at start could take. issuedAtFirstOpen answers for a channel key
// as of that open.
func clearUnissuedChannelLinks(
	sch *schematic.Service,
	tbl *table.Service,
	arcs *arc.Service,
	issuedAtFirstOpen func(channel.Key) bool,
) gorp.Migration {
	issued := func(target gorp.Target) (bool, error) {
		if target.Table != string(resource.TypeChannel) {
			return false, errors.Newf("unexpected weak link to %s", target.Table)
		}
		k, err := channel.ParseKey(target.Key)
		if err != nil {
			return false, err
		}
		return issuedAtFirstOpen(k), nil
	}
	schematics := sch.DanglingMigration(clearUnissuedKey, issued)
	tables := tbl.DanglingMigration(clearUnissuedKey, issued)
	automations := arcs.DanglingMigration(clearUnissuedKey, issued)
	return migrate.New(
		clearUnissuedKey,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			for _, m := range []gorp.Migration{schematics, tables, automations} {
				if err := m.Run(ctx, tx, ins); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
