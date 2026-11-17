// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/storage/ts"
)

// Rename renames each storage channel keyed in renames to its corresponding new name.
// Free channels have no storage registration and are skipped.
func (s *Service) Rename(ctx context.Context, renames map[Key]string) error {
	stored := make(map[ts.ChannelKey]string, len(renames))
	for key, name := range renames {
		if !key.Free() {
			stored[key.StorageKey()] = name
		}
	}
	return s.cfg.TS.RenameChannels(ctx, stored)
}
