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
	"slices"

	"github.com/synnaxlabs/gorp"
)

// Tags that generated references to a channel carry. Oracle writes them for
// `@ref direction read` and `@ref direction write`.
const (
	// ReadTag is the Tag of a reference whose holder reads the channel.
	ReadTag uint8 = 1
	// WriteTag is the Tag of a reference whose holder writes the channel.
	WriteTag uint8 = 2
)

var tagDirections = map[uint8]Direction{
	ReadTag:  DirectionRead,
	WriteTag: DirectionWrite,
}

// Use is a record that reads or writes a channel.
type Use struct {
	// Holder is the record.
	Holder gorp.Target
	// Direction is how the record uses the channel.
	Direction Direction
}

// Uses returns, at each position of keys, each record that reads or writes the channel
// with that key, as seen by snap, once for each direction. A deleted channel keeps the
// uses of its weak references. It blocks and fails as gorp.Table.Dependents does.
func (s *Service) Uses(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []Key,
) ([][]Use, error) {
	deps, err := s.table.DependentsMany(ctx, snap, keys)
	if err != nil {
		return nil, err
	}
	out := make([][]Use, len(keys))
	for i, held := range deps {
		out[i] = make([]Use, 0, len(held))
		for _, d := range held {
			dir, ok := tagDirections[d.Tag]
			u := Use{Holder: d.Holder, Direction: dir}
			if ok && !slices.Contains(out[i], u) {
				out[i] = append(out[i], u)
			}
		}
	}
	return out, nil
}
