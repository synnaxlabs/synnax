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

	"github.com/synnaxlabs/gorp"
)

// Tombstones returns the tombstones of s, as snap sees them.
func (s *Service) Tombstones(
	ctx context.Context,
	snap gorp.Snapshot,
) ([]Tombstone, error) {
	var out []Tombstone
	err := s.tombstones.table.NewRetrieve().Entries(&out).Exec(ctx, snap)
	return out, err
}
