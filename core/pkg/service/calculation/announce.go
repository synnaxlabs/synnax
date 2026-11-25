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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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

// announce attaches through cw a set of whether c writes its output, in a batch that
// goes to every subject who reads the output, then the fields of c as announceFields
// does.
func (s *Service) announce(tx gorp.Tx, cw channel.Writer, c Calculation, writes bool) {
	cw.Announce(c.Output, channel.NewSetAction(channel.SetPayload{
		Path:  []string{"calculated"},
		Value: writes,
	}))
	s.announceFields(tx, cw, c, writes)
}

// announceFields attaches through cw a set of the fields of c for its output, or of no
// fields when writes is false, as one batch that goes only to a subject who reads c.
func (s *Service) announceFields(
	tx gorp.Tx,
	cw channel.Writer,
	c Calculation,
	writes bool,
) {
	seq := cw.Announce(c.Output, fieldsOf(c, writes)...)
	s.outputChecks.Add(tx, batchCheck[channel.Key]{
		id:    batchID[channel.Key]{key: c.Output, seq: seq},
		check: s.Readable(c),
	})
}

// AnnounceOutputs announces through w, in tx, that a calculation writes each channel
// among keys that one writes, and the fields of that calculation.
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
	for _, c := range calcs {
		s.announce(tx, w, c, true)
	}
	return nil
}

// AnnounceRenamed attaches, in tx, a set of its reads to each calculation whose output
// is among keys. The set changes nothing, and tells each client to read again the name
// that the calculation takes from its output.
func (s *Service) AnnounceRenamed(
	ctx context.Context,
	tx gorp.Tx,
	keys ...channel.Key,
) error {
	calcs, err := s.Outputs(ctx, tx, keys...)
	if err != nil {
		return err
	}
	w := s.NewWriter(tx)
	for _, c := range calcs {
		w.attach(c, NewSetAction(SetPayload{Path: []string{"reads"}, Value: c.Reads}))
	}
	return nil
}

// ChannelBatchChecks returns the lookup of the check of each channel action batch that
// c holds, by the key of its channel and its number. A batch that announces the
// expression and operations of an output goes only to a subject who reads the
// calculation, and any other batch has no check of its own.
func (s *Service) ChannelBatchChecks(
	c gorp.Commit,
) func(key channel.Key, seq uint64) (func(allowed func(resource.ID) bool) bool, bool) {
	attached := s.outputChecks.Of(c)
	if len(attached) == 0 {
		return noChannelCheck
	}
	byBatch := checksOf(attached)
	return func(
		key channel.Key,
		seq uint64,
	) (func(allowed func(resource.ID) bool) bool, bool) {
		return byBatch[batchID[channel.Key]{key: key, seq: seq}], true
	}
}

func noChannelCheck(
	channel.Key,
	uint64,
) (func(allowed func(resource.ID) bool) bool, bool) {
	return nil, true
}
