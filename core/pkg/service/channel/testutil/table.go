// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// Table stores channels under keys that tests choose, for records that reference
// channels without the channel service open. The channels have no time series storage
// behind them.
type Table struct {
	db    *gorp.DB
	table *gorp.Table[channel.Key, channel.Channel]
}

// OpenTable opens the table that channels are stored in on db. Close the returned
// Table to close it.
func OpenTable(ctx context.Context, db *gorp.DB) (*Table, error) {
	table, err := gorp.OpenTable(ctx, gorp.TableConfig[channel.Key, channel.Channel]{
		DB:   db,
		Name: string(resource.TypeChannel),
	})
	if err != nil {
		return nil, err
	}
	return &Table{db: db, table: table}, nil
}

// Create stores a channel under each of keys.
func (t *Table) Create(ctx context.Context, keys ...channel.Key) error {
	channels := make([]channel.Channel, len(keys))
	for i, k := range keys {
		channels[i] = channel.Channel{Leaseholder: k.Lease(), LocalKey: k.LocalKey()}
	}
	return t.table.NewCreate().Entries(&channels).Exec(ctx, t.db)
}

// Close closes the table.
func (t *Table) Close() error { return t.table.Close() }

// Delete deletes the channels under keys. The delete runs the reference checks of the
// tables that reference channels.
func (t *Table) Delete(ctx context.Context, keys ...channel.Key) error {
	return t.table.NewDelete().
		Where(gorp.MatchKeys[channel.Key, channel.Channel](keys...)).
		Exec(ctx, t.db)
}
