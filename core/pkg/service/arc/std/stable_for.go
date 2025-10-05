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
	"github.com/synnaxlabs/synnax/pkg/service/arc/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/value"
	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/telem"
)

var symbolStableFor = ir.Symbol{
	Name: "stable_for",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Config: maps.Ordered[string, ir.Type]{
			Keys:   []string{"duration"},
			Values: []ir.Type{ir.TimeSpan{}},
		},
		Params: maps.Ordered[string, ir.Type]{
			Keys:   []string{"input"},
			Values: []ir.Type{ir.NewTypeVariable("T", nil)}, // Any type
		},
		Return: ir.NewTypeVariable("T", nil), // Pass through same type
	},
}

type stableFor struct {
	base
	duration    telem.TimeSpan
	value       uint64
	input       *value.Value
	lastSent    *uint64
	lastChanged telem.TimeStamp
	now         func() telem.TimeStamp
}

var _ stage.Node = (*stableFor)(nil)

func (s *stableFor) Load(param string, val value.Value) {
	if param == "input" {
		s.input = &val
	}
}

func (s *stableFor) Next(ctx context.Context) {
	if s.input == nil {
		return
	}

	if s.input.Value != s.value {
		s.value = s.input.Value
		s.lastChanged = s.now()
	}
	if telem.TimeSpan(s.now()-s.lastChanged) >= s.duration && (s.lastSent == nil || *s.lastSent != s.value) {
		v := s.value
		s.lastSent = &v
		s.outputHandler(ctx, "output", *s.input)
	}
}

func createStableFor(_ context.Context, cfg Config) (stage.Node, error) {
	// Handle both int and float64 for duration
	var duration telem.TimeSpan
	switch v := cfg.Node.Config["duration"].(type) {
	case float64:
		duration = telem.TimeSpan(v)
	case int:
		duration = telem.TimeSpan(v)
	case int64:
		duration = telem.TimeSpan(v)
	default:
		duration = telem.TimeSpan(0)
	}
	now := cfg.Now
	if now == nil {
		now = telem.Now
	}
	stg := &stableFor{
		base:     base{key: cfg.Node.Key},
		duration: duration,
		now:      now,
	}
	return stg, nil
}
