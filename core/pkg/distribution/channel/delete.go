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

	"github.com/samber/lo"
)

// Delete deletes the storage channels for the provided keys. Free channels have no
// storage registration and are skipped. It does not touch channel metadata.
func (s *Service) Delete(ctx context.Context, keys Keys) error {
	stored := Keys(lo.Reject(keys, func(key Key, _ int) bool { return key.Free() }))
	return s.cfg.TS.DeleteChannels(stored.Storage())
}
