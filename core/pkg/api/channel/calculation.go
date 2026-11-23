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
	"strconv"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// createChannels creates channels in tx. The channel at each position of unsavedAt is
// the output of the unsaved calculation at the same position of unsaved, made through
// calculation.Writer.CreateOutput. When retrieveIfExists is true, a channel whose name
// or key is stored is found as channel.RetrieveIfNameExists finds it, and no
// calculation is made for it. It returns the stored channels in the order of channels,
// then the index channels of the new outputs. The error of a failed channel has its
// position in channels as the first segment of its path.
func (s *Service) createChannels(
	ctx context.Context,
	tx gorp.Tx,
	channels []channel.Channel,
	unsaved []calculation.Unsaved,
	unsavedAt []int,
	retrieveIfExists bool,
) ([]channel.Channel, error) {
	var (
		stored  = slices.Clone(channels)
		plainAt []int
		opts    []channel.CreateOption
		byName  = map[string]channel.Channel{}
		byKey   = map[channel.Key]channel.Channel{}
	)
	for i := range channels {
		if !slices.Contains(unsavedAt, i) {
			plainAt = append(plainAt, i)
		}
	}
	if retrieveIfExists {
		opts = append(opts, channel.RetrieveIfNameExists())
		found, err := s.storedChannels(ctx, tx, channels)
		if err != nil {
			return nil, err
		}
		for _, ch := range found {
			byName[ch.Name], byKey[ch.Key()] = ch, ch
		}
	}
	if err := s.createPlain(ctx, tx, stored, plainAt, opts); err != nil {
		return nil, err
	}
	var (
		w       = s.calculations.NewWriter(tx)
		indexes []channel.Channel
	)
	for k, u := range unsaved {
		i := unsavedAt[k]
		if ch, ok := storedOf(byName, byKey, u.Output); ok && retrieveIfExists {
			stored[i] = ch
			continue
		}
		if err := s.checkKeyFree(ctx, tx, u.Output); err != nil {
			return nil, validate.PathedError(err, strconv.Itoa(i))
		}
		c := calculation.Calculation{Expression: u.Expression, Operations: u.Operations}
		created, err := w.CreateOutput(ctx, &c, u.Output.Name, u.Output.Group)
		if err != nil {
			return nil, validate.PathedError(err, strconv.Itoa(i))
		}
		stored[i] = created[0]
		indexes = append(indexes, created[1:]...)
	}
	return slices.Concat(stored, indexes), nil
}

// createPlain creates the channels at plainAt of channels, which no calculation
// writes, and puts each created channel back at its position.
func (s *Service) createPlain(
	ctx context.Context,
	tx gorp.Tx,
	channels []channel.Channel,
	plainAt []int,
	opts []channel.CreateOption,
) error {
	plain := make([]channel.Channel, len(plainAt))
	for j, i := range plainAt {
		plain[j] = channels[i]
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &plain, opts...); err != nil {
		if b, ok := errors.AsType[channel.BatchError](err); ok {
			return validate.PathedError(b.Err, strconv.Itoa(plainAt[b.At]))
		}
		return err
	}
	for j, i := range plainAt {
		channels[i] = plain[j]
	}
	return nil
}

// storedChannels returns the stored channels that have the name or key of a channel in
// channels.
func (s *Service) storedChannels(
	ctx context.Context,
	tx gorp.Tx,
	channels []channel.Channel,
) ([]channel.Channel, error) {
	var byName []channel.Channel
	if err := s.internal.NewRetrieve().
		Where(channel.MatchNames(channel.Names(channels)...)).
		Entries(&byName).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	var keys []channel.Key
	for _, ch := range channels {
		if ch.LocalKey != 0 {
			keys = append(keys, ch.Key())
		}
	}
	if len(keys) == 0 {
		return byName, nil
	}
	var byKey []channel.Channel
	if err := s.internal.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&byKey).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return slices.Concat(byName, byKey), nil
}

// checkKeyFree returns an error wrapping validate.ErrValidation when ch names the key
// of a stored channel, which the old channel API refuses to create again.
func (s *Service) checkKeyFree(
	ctx context.Context,
	tx gorp.Tx,
	ch channel.Channel,
) error {
	if ch.LocalKey == 0 {
		return nil
	}
	exists, err := s.internal.NewRetrieve().
		Where(channel.MatchKeys(ch.Key())).
		Exists(ctx, tx)
	if err != nil || !exists {
		return err
	}
	return errors.Wrapf(validate.ErrValidation, "channel %s already exists", ch.Key())
}

// storedOf returns the stored channel that has the name or key of ch.
func storedOf(
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

// calculationFields are the members of a channel that the old channel API edits on its
// calculation.
var calculationFields = set.New("expression", "operations")

// calculationDocument is the part of a calculated channel that the old channel API
// edits on its calculation.
type calculationDocument struct {
	Expression string              `json:"expression"`
	Operations []channel.Operation `json:"operations"`
}

// applyChannel applies acts to the channel with key in tx. An action on the expression
// or the operations edits the calculation that writes the channel, and every other
// action edits the channel. It returns the actions the Core computed, such as a set of
// a data type that the new expression changed. It returns an error wrapping
// validate.ErrValidation for an edit of the expression or the operations of a channel
// that no calculation writes.
func (s *Service) applyChannel(
	ctx context.Context,
	tx gorp.Tx,
	dispatchKey string,
	key channel.Key,
	acts []channel.Action,
) ([]channel.Action, error) {
	var calcActs, chanActs []channel.Action
	for _, a := range acts {
		if editsCalculation(a) {
			calcActs = append(calcActs, a)
		} else {
			chanActs = append(chanActs, a)
		}
	}
	var computed []channel.Action
	if len(chanActs) > 0 {
		var err error
		if computed, err = s.internal.NewWriter(tx).
			WithDispatchKey(dispatchKey).
			Apply(ctx, key, chanActs...); err != nil {
			return nil, err
		}
	}
	if len(calcActs) == 0 {
		return computed, nil
	}
	set, err := s.applyCalculation(ctx, tx, dispatchKey, key, calcActs)
	if err != nil || set == nil {
		return computed, err
	}
	return slices.Concat(computed, []channel.Action{*set}), nil
}

// applyCalculation applies acts on the expression or the operations to the calculation
// that writes the channel with key. It returns a set of the data type of the channel
// when the acts changed it.
func (s *Service) applyCalculation(
	ctx context.Context,
	tx gorp.Tx,
	dispatchKey string,
	key channel.Key,
	acts []channel.Action,
) (*channel.Action, error) {
	calcs, err := s.calculations.Outputs(ctx, tx, key)
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
	doc, ops, err := edited(c, acts)
	if err != nil {
		return nil, err
	}
	before, err := s.channelOf(ctx, tx, key)
	if err != nil {
		return nil, err
	}
	w := s.calculations.NewWriter(tx).WithDispatchKey(dispatchKey)
	// The reads that ApplyTo adds belong to the calculation, which the old API does not
	// show.
	next, _, err := w.ApplyTo(
		ctx,
		c,
		calculation.NewSetAction(calculation.SetPayload{
			Path:  []string{"expression"},
			Value: doc.Expression,
		}),
		calculation.NewSetAction(calculation.SetPayload{
			Path:  []string{"operations"},
			Value: ops,
		}),
	)
	if err != nil {
		return nil, err
	}
	// A subject whom the edit lets read the calculation holds none of its fields yet,
	// so the announce carries all of them.
	w.Announce(next)
	after, err := s.channelOf(ctx, tx, key)
	if err != nil || after.DataType == before.DataType {
		return nil, err
	}
	set := channel.NewSetAction(channel.SetPayload{
		Path:  []string{"data_type"},
		Value: after.DataType,
	})
	return &set, nil
}

// edited returns the document of c after acts, and its operations as the calculation
// holds them.
func edited(
	c calculation.Calculation,
	acts []channel.Action,
) (calculationDocument, []calculation.Operation, error) {
	doc := calculationDocument{
		Expression: c.Expression,
		Operations: calculation.ToChannelOperations(c.Operations),
	}
	var err error
	for _, a := range acts {
		if doc, err = editDocument(doc, a); err != nil {
			return doc, nil, err
		}
	}
	ops, err := calculation.FromChannelOperations(doc.Operations)
	if err != nil {
		return doc, nil, validate.PathedError(err, "operations")
	}
	return doc, ops, nil
}

func (s *Service) channelOf(
	ctx context.Context,
	tx gorp.Tx,
	key channel.Key,
) (channel.Channel, error) {
	var ch channel.Channel
	err := s.internal.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, tx)
	return ch, err
}

// editsCalculation reports whether a edits the expression or the operations of a
// channel, which applyChannel sends to the calculation of the channel.
func editsCalculation(a channel.Action) bool {
	path := actionPath(a)
	return len(path) > 0 && calculationFields.Contains(path[0])
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

// editDocument applies a path action of the old channel API to doc.
func editDocument(
	doc calculationDocument,
	a channel.Action,
) (calculationDocument, error) {
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
