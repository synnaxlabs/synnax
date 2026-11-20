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

// Unsaved is a calculation that a request asks for and that is not stored, with the
// channel it writes.
type Unsaved struct {
	// Output is the channel that the calculation writes. It has no key when the request
	// makes it.
	Output channel.Channel
	// Expression is the expression of the calculation.
	Expression string
	// Operations are the operations of the calculation.
	Operations []Operation
}

// newAnalyzer returns an analyzer that resolves channels through snap.
func (s *Service) newAnalyzer(snap gorp.Snapshot) *channel.CalculationAnalyzer {
	return channel.NewCalculationAnalyzer(
		s.channels.NewArcSymbolResolver(snap),
		parser.Config{AllowDashedNames: !s.channels.ShouldValidateNames()},
	)
}

// Inputs returns the stored channels that the calculations of outputs and the unsaved
// calculations read, directly or through other calculations, and the channels that
// reset their operations. An unsaved calculation replaces the stored calculation of its
// output. A channel that is not stored yet is not an input. A calculation whose
// expression fails analysis reads the channels that it resolved. Inputs returns the
// error of a channel read that fails, and reads through snap.
func (s *Service) Inputs(
	ctx context.Context,
	snap gorp.Snapshot,
	outputs channel.Keys,
	unsaved []Unsaved,
) (channel.Keys, error) {
	var (
		analyzer = s.newAnalyzer(snap)
		visited  = set.New[channel.Key]()
		inputs   channel.Keys
	)
	for _, u := range unsaved {
		visited.Add(u.Output.Key())
	}
	var stored channel.Keys
	for _, k := range outputs {
		if !visited.Contains(k) {
			visited.Add(k)
			stored = append(stored, k)
		}
	}
	level, err := s.pendingOf(ctx, snap, stored)
	if err != nil {
		return nil, err
	}
	level = append(slices.Clone(unsaved), level...)
	for len(level) > 0 {
		var next channel.Keys
		for _, u := range level {
			reads, err := calculationReads(ctx, analyzer, u)
			if err != nil {
				return nil, err
			}
			for _, k := range reads {
				// A channel of the request that is not stored yet has no local key.
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

// pendingOf returns the stored calculations that write the channels with keys.
func (s *Service) pendingOf(
	ctx context.Context,
	snap gorp.Snapshot,
	keys channel.Keys,
) ([]Unsaved, error) {
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
	level := make([]Unsaved, len(channels))
	for i, ch := range channels {
		c := calcs[ch.Key()]
		level[i] = Unsaved{
			Output:     ch,
			Expression: c.Expression,
			Operations: c.Operations,
		}
	}
	return level, nil
}

// calculationReads returns the channels that u reads, and the channels that reset its
// operations. When its expression fails analysis, it returns the channels the analysis
// resolved.
func calculationReads(
	ctx context.Context,
	analyzer *channel.CalculationAnalyzer,
	u Unsaved,
) (channel.Keys, error) {
	result, err := analyzer.AnalyzeExpression(
		ctx,
		u.Output,
		u.Expression,
		HasDerivative(u.Operations),
	)
	if _, ok := errors.AsType[*diagnostics.Diagnostics](err); err != nil && !ok {
		return nil, err
	}
	return append(result.Deps, resets(u.Operations)...), nil
}
