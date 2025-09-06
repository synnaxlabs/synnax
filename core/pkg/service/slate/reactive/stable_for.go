// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reactive

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/arc/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/telem"
)

type stableFor struct {
	confluence.LinearTransform[spec.Value, spec.Value]
	value       spec.Value
	lastSend    spec.Value
	lastChanged telem.TimeStamp
}

func newStableFor(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.StableForType {
		return false, nil
	}
	duration := cfg.node.Config["duration"].(float64)
	dur := telem.TimeSpan(duration)
	s := &stableFor{}
	s.Transform = func(ctx context.Context, i spec.Value) (o spec.Value, shouldSend bool, err error) {
		if s.value != i {
			s.value = i
			s.lastChanged = telem.Now()
		}
		if shouldSend = telem.Since(s.lastChanged) > dur && s.lastSend != s.value; shouldSend {
			s.lastSend = s.value
		}
		return s.value, shouldSend, nil
	}
	plumber.SetSegment[spec.Value, spec.Value](cfg.pipeline, address.Address(cfg.node.Key), s)
	return true, nil
}
