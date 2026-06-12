// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// channelRename returns an lsp.OnRename callback that propagates a rename
// targeting a channel symbol to the underlying channel service. The Synnax
// channel resolver populates KindChannel symbols with the channel key in
// Symbol.ID, so the callback can rename by key without a name lookup. Renames
// of any other symbol kind are no-ops.
func channelRename(channels *channel.Service) lsp.OnRename {
	return func(ctx context.Context, sym *symbol.Symbol, _, newName string) error {
		if sym.Kind != symbol.KindChannel {
			return nil
		}
		return channels.NewWriter(nil).Rename(ctx, channel.Key(sym.ID), newName, false)
	}
}
