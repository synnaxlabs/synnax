// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"slices"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// HasDerivative reports whether ops holds a derivative.
func HasDerivative(ops []Operation) bool {
	return slices.ContainsFunc(ops, func(op Operation) bool {
		_, ok := op.Variant.(DerivativeOperation)
		return ok
	})
}

// resets returns the reset channel of each windowed operation of ops that has one.
func resets(ops []Operation) []channel.Key {
	var keys []channel.Key
	for _, op := range ops {
		var w Window
		switch v := op.Variant.(type) {
		case MinOperation:
			w = v.Window
		case MaxOperation:
			w = v.Window
		case AvgOperation:
			w = v.Window
		}
		if w.Reset != nil {
			keys = append(keys, *w.Reset)
		}
	}
	return keys
}

// pending is a calculation that Inputs has yet to analyze.
type pending struct {
	output     channel.Channel
	expression string
	operations []Operation
}

// newAnalyzer returns an analyzer that resolves channels through snap.
func (s *Service) newAnalyzer(snap gorp.Snapshot) *channel.CalculationAnalyzer {
	return channel.NewCalculationAnalyzer(
		s.channels.NewArcSymbolResolver(snap),
		parser.Config{AllowDashedNames: !s.channels.ShouldValidateNames()},
	)
}

// Inputs returns the stored channels that the calculations among channels read,
// directly or through other calculations, and the channels that reset their
// operations. A channel among channels is a calculation when it has an expression,
// which replaces its stored calculation, or when a calculation writes it. A channel
// that is not stored yet is not an input. A calculation whose expression fails analysis
// reads nothing. Inputs returns the error of a channel read that fails, and reads
// through snap.
func (s *Service) Inputs(
	ctx context.Context,
	snap gorp.Snapshot,
	channels []channel.Channel,
) (channel.Keys, error) {
	stored, err := s.Outputs(ctx, snap, channel.KeysFromChannels(channels)...)
	if err != nil {
		return nil, err
	}
	var (
		analyzer = s.newAnalyzer(snap)
		visited  = set.New[channel.Key]()
		level    []pending
		inputs   channel.Keys
	)
	for _, ch := range channels {
		if ch.Expression != "" {
			level = append(level, pending{
				output:     ch,
				expression: ch.Expression,
				operations: fromChannelOperations(ch.Operations),
			})
			visited.Add(ch.Key())
		} else if c, ok := stored[ch.Key()]; ok {
			level = append(level, pending{ch, c.Expression, c.Operations})
			visited.Add(ch.Key())
		}
	}
	for len(level) > 0 {
		var next channel.Keys
		for _, p := range level {
			reads, err := calculationReads(ctx, analyzer, p)
			if err != nil {
				return nil, err
			}
			for _, k := range reads {
				// A channel among channels that is not stored yet has no local key.
				if k.LocalKey() == 0 {
					continue
				}
				inputs = append(inputs, k)
				if !visited.Contains(k) {
					visited.Add(k)
					next = append(next, k)
				}
			}
		}
		if level, err = s.pendingOf(ctx, snap, next); err != nil {
			return nil, err
		}
	}
	return inputs.Unique(), nil
}

// pendingOf returns the calculations that write the channels with keys.
func (s *Service) pendingOf(
	ctx context.Context,
	snap gorp.Snapshot,
	keys channel.Keys,
) ([]pending, error) {
	calcs, err := s.Outputs(ctx, snap, keys...)
	if err != nil || len(calcs) == 0 {
		return nil, err
	}
	outputs := make(channel.Keys, 0, len(calcs))
	for k := range calcs {
		outputs = append(outputs, k)
	}
	var channels []channel.Channel
	if err = s.channels.NewRetrieve().
		Where(channel.MatchKeys(outputs...)).
		Entries(&channels).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	level := make([]pending, len(channels))
	for i, ch := range channels {
		c := calcs[ch.Key()]
		level[i] = pending{ch, c.Expression, c.Operations}
	}
	return level, nil
}

// calculationReads returns the channels that p reads, and the channels that reset its
// operations. It returns none when the expression of p fails analysis.
func calculationReads(
	ctx context.Context,
	analyzer *channel.CalculationAnalyzer,
	p pending,
) (channel.Keys, error) {
	result, err := analyzer.AnalyzeExpression(
		ctx,
		p.output,
		p.expression,
		HasDerivative(p.operations),
	)
	if _, ok := errors.AsType[*diagnostics.Diagnostics](err); ok {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return append(result.Deps, resets(p.operations)...), nil
}
