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
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
)

// ClearUnissuedChannelLinks returns the migration that removes, from every stored
// schematic and table, each channel key that the Core never issued. An import in 0.62
// kept the raw keys of another Core, which a later channel could take. It must run
// after the channel service opens.
func ClearUnissuedChannelLinks(
	sch *schematic.Service,
	tbl *table.Service,
) gorp.Migration {
	return migrate.New(
		"v63_clear_unissued_channel_links",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			if err := sch.DropDangling(ctx, tx); err != nil {
				return err
			}
			return tbl.DropDangling(ctx, tx)
		},
	)
}
