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
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ChannelOperations returns the channel form of ops, which the old channel API sends.
func ChannelOperations(ops []Operation) []channel.Operation {
	out := make([]channel.Operation, len(ops))
	for i, op := range ops {
		var (
			typ channel.OperationType
			w   Window
		)
		switch v := op.Variant.(type) {
		case MinOperation:
			typ, w = channel.OperationTypeMin, v.Window
		case MaxOperation:
			typ, w = channel.OperationTypeMax, v.Window
		case AvgOperation:
			typ, w = channel.OperationTypeAvg, v.Window
		case DerivativeOperation:
			typ = channel.OperationTypeDerivative
		}
		out[i] = channel.Operation{Type: typ, Duration: w.Duration}
		if w.Reset != nil {
			out[i].ResetChannel = *w.Reset
		}
	}
	return out
}

// Requested is a channel that the old channel API creates. A non-empty Expression makes
// the channel the output of a new calculation.
type Requested struct {
	// Channel is the channel to create.
	Channel channel.Channel
	// Expression is the expression of the calculation that writes Channel.
	Expression string
	// Operations are the operations of that calculation.
	Operations []channel.Operation
}

// CreateChannels creates channels as the old channel API does. A requested channel with
// an expression becomes the output of a new calculation through CreateOutput, and every
// other channel goes to the channel service. When retrieveIfExists is true, a channel
// whose name or key is stored is found as channel.RetrieveIfNameExists finds it, and
// CreateChannels writes no calculation for it. It returns the stored channels in the
// order of requested, then the index channels of the new outputs. A failed channel's
// error holds its position in requested.
func (w Writer) CreateChannels(
	ctx context.Context,
	requested []Requested,
	retrieveIfExists bool,
) ([]channel.Channel, error) {
	var (
		channels = make([]channel.Channel, len(requested))
		plain    []channel.Channel
		plainAt  []int
		opts     []channel.CreateOption
		stored   = map[string]channel.Channel{}
		keys     = map[channel.Key]channel.Channel{}
	)
	for i, r := range requested {
		channels[i] = r.Channel
		if r.Expression == "" {
			plain, plainAt = append(plain, r.Channel), append(plainAt, i)
		}
	}
	if retrieveIfExists {
		opts = append(opts, channel.RetrieveIfNameExists())
		found, err := w.storedChannels(ctx, channels)
		if err != nil {
			return nil, err
		}
		for _, ch := range found {
			stored[ch.Name], keys[ch.Key()] = ch, ch
		}
	}
	if len(plain) > 0 {
		if err := w.channels.NewWriter(w.tx).
			WithDispatchKey(w.dispatchKey).
			CreateMany(ctx, &plain, opts...); err != nil {
			return nil, reindex(err, func(j int) []string {
				return []string{strconv.Itoa(plainAt[j])}
			})
		}
		for j, i := range plainAt {
			channels[i] = plain[j]
		}
	}
	var indexes []channel.Channel
	for i, r := range requested {
		if r.Expression == "" {
			continue
		}
		if s, ok := found(stored, keys, r.Channel); ok && retrieveIfExists {
			channels[i] = s
			continue
		}
		c := Calculation{
			Expression: r.Expression,
			Operations: FromChannelOperations(r.Operations),
		}
		created, err := w.CreateOutput(ctx, &c, r.Channel.Name, r.Channel.Group)
		if err != nil {
			return nil, validate.PathedError(err, strconv.Itoa(i))
		}
		channels[i] = created[0]
		indexes = append(indexes, created[1:]...)
	}
	return append(channels, indexes...), nil
}

// found returns the stored channel that has the name or key of ch.
func found(
	byName map[string]channel.Channel,
	byKey map[channel.Key]channel.Channel,
	ch channel.Channel,
) (channel.Channel, bool) {
	if ch.LocalKey != 0 {
		if s, ok := byKey[ch.Key()]; ok {
			return s, true
		}
	}
	s, ok := byName[ch.Name]
	return s, ok
}

// legacyFields are the members of a channel that the old channel API edits on its
// calculation.
var legacyFields = set.New("expression", "operations")

// legacyDocument is the part of a calculated channel that the old channel API edits on
// its calculation.
type legacyDocument struct {
	Expression string              `json:"expression"`
	Operations []channel.Operation `json:"operations"`
}

// ApplyChannel applies actions of the old channel API to the channel with key. An
// action on the expression or the operations edits the calculation that writes the
// channel, and every other action edits the channel. It returns the actions the Core
// computed, such as a set of a data type that the new expression changed. It returns
// an error wrapping validate.ErrValidation for an edit of the expression or the
// operations of a channel that no calculation writes.
func (w Writer) ApplyChannel(
	ctx context.Context,
	key channel.Key,
	acts ...channel.Action,
) ([]channel.Action, error) {
	var calcActs, chanActs []channel.Action
	for _, a := range acts {
		if path := actionPath(a); len(path) > 0 && legacyFields.Contains(path[0]) {
			calcActs = append(calcActs, a)
		} else {
			chanActs = append(chanActs, a)
		}
	}
	var computed []channel.Action
	if len(chanActs) > 0 {
		var err error
		if computed, err = w.channels.NewWriter(w.tx).
			WithDispatchKey(w.dispatchKey).
			Apply(ctx, key, chanActs...); err != nil {
			return nil, err
		}
	}
	if len(calcActs) == 0 {
		return computed, nil
	}
	set, err := w.applyLegacy(ctx, key, calcActs)
	if err != nil || set == nil {
		return computed, err
	}
	return slices.Concat(computed, []channel.Action{*set}), nil
}

// applyLegacy applies actions of the old channel API on the expression or the
// operations to the calculation that writes the channel with key. It returns a set of
// the data type of the channel when the actions changed it.
func (w Writer) applyLegacy(
	ctx context.Context,
	key channel.Key,
	acts []channel.Action,
) (*channel.Action, error) {
	calcs, err := w.svc.Outputs(ctx, w.tx, key)
	if err != nil {
		return nil, err
	}
	c, ok := calcs[key]
	if !ok {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"only a calculated channel can change its expression or operations",
		)
	}
	doc := legacyDocument{
		Expression: c.Expression,
		Operations: ChannelOperations(c.Operations),
	}
	for _, a := range acts {
		if doc, err = editLegacy(doc, a); err != nil {
			return nil, err
		}
	}
	before, err := w.outputOf(ctx, key)
	if err != nil {
		return nil, err
	}
	if err = w.Apply(
		ctx,
		c.Key,
		NewSetAction(SetPayload{Path: []string{"expression"}, Value: doc.Expression}),
		NewSetAction(SetPayload{
			Path:  []string{"operations"},
			Value: FromChannelOperations(doc.Operations),
		}),
	); err != nil {
		return nil, err
	}
	after, err := w.outputOf(ctx, key)
	if err != nil {
		return nil, err
	}
	if after.DataType == before.DataType {
		return nil, nil
	}
	set := channel.NewSetAction(channel.SetPayload{
		Path:  []string{"data_type"},
		Value: after.DataType,
	})
	return &set, nil
}

func (w Writer) outputOf(
	ctx context.Context,
	key channel.Key,
) (channel.Channel, error) {
	var ch channel.Channel
	err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, w.tx)
	return ch, err
}

// actionPath returns the path that a path action of the old channel API edits.
func actionPath(a channel.Action) []string {
	switch {
	case a.Set != nil:
		return a.Set.Path
	case a.Remove != nil:
		return a.Remove.Path
	case a.Insert != nil:
		return a.Insert.Path
	case a.Move != nil:
		return a.Move.Path
	default:
		return nil
	}
}

// editLegacy applies a path action of the old channel API to doc.
func editLegacy(doc legacyDocument, a channel.Action) (legacyDocument, error) {
	var apply func(map[string]any) error
	switch {
	case a.Set != nil:
		apply = patch.Set(*a.Set).Apply
	case a.Remove != nil:
		apply = patch.Remove(*a.Remove).Apply
	case a.Insert != nil:
		apply = patch.Insert(*a.Insert).Apply
	case a.Move != nil:
		apply = patch.Move(*a.Move).Apply
	}
	return patch.Edit(doc, apply)
}
