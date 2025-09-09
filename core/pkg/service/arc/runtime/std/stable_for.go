// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/x/telem"
)

type stableFor struct {
	base
	duration    telem.TimeSpan
	value       uint64
	lastSent    uint64
	lastChanged telem.TimeStamp
	now         func() telem.TimeStamp
}

func (s *stableFor) Next(ctx context.Context, value stage.Value) {
	if value.Value != s.lastSent {
		s.value = value.Value
		s.lastChanged = s.now()
	}
	if telem.Since(s.lastChanged) > s.duration && s.lastSent != s.value {
		s.lastSent = s.value
		s.outputHandler(ctx, value)
	}
}

func createStableFor(_ context.Context, node ir.Node) (stage.Stage, error) {
	duration := telem.TimeSpan(node.Config["duration"].(int))
	stg := &stableFor{}
	stg.duration = duration
	return stg, nil
}
