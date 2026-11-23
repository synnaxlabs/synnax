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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// fieldsOf returns the channel actions that set the expression and operations of the
// output of c to those of c, or to none when c no longer writes it.
func fieldsOf(c Calculation, writes bool) []channel.Action {
	expression, operations := "", []channel.Operation{}
	if writes {
		expression, operations = c.Expression, ToChannelOperations(c.Operations)
	}
	return []channel.Action{
		channel.NewSetAction(channel.SetPayload{
			Path:  []string{"expression"},
			Value: expression,
		}),
		channel.NewSetAction(channel.SetPayload{
			Path:  []string{"operations"},
			Value: operations,
		}),
	}
}

// announce attaches acts for output through cw as one batch, which goes only to a
// subject who reads c.
func (s *Service) announce(
	tx gorp.Tx,
	cw channel.Writer,
	output channel.Key,
	c Calculation,
	acts ...channel.Action,
) {
	seq := cw.Announce(output, acts...)
	s.outputChecks.Add(tx, batchCheck[channel.Key]{
		id:    batchID[channel.Key]{key: output, seq: seq},
		check: s.readable(c),
	})
}

// AnnounceOutputs announces through w, in tx, the calculation fields of each channel
// among keys that a calculation writes.
func (s *Service) AnnounceOutputs(
	ctx context.Context,
	tx gorp.Tx,
	w channel.Writer,
	keys ...channel.Key,
) error {
	calcs, err := s.Outputs(ctx, tx, keys...)
	if err != nil {
		return err
	}
	for key, c := range calcs {
		s.announce(tx, w, key, c, fieldsOf(c, true)...)
	}
	return nil
}

// ChannelBatchChecks returns the lookup of the check of each channel action batch that
// c holds, by the key of its channel and its number. A batch that announces the
// calculation fields of an output goes only to a subject who reads the calculation, and
// any other batch has no check of its own.
func (s *Service) ChannelBatchChecks(
	c gorp.Commit,
) func(key channel.Key, seq uint64) (streamCheck, bool) {
	attached := s.outputChecks.Of(c)
	if len(attached) == 0 {
		return noChannelCheck
	}
	byBatch := checksOf(attached)
	return func(
		key channel.Key,
		seq uint64,
	) (streamCheck, bool) {
		return byBatch[batchID[channel.Key]{key: key, seq: seq}], true
	}
}

func noChannelCheck(
	channel.Key,
	uint64,
) (streamCheck, bool) {
	return nil, true
}
