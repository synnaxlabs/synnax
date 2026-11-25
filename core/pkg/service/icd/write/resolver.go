// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Write is a write of one ICD message, at a path of a task config.
type Write struct {
	// Path is the path of the write in its config.
	Path []string
	// ICD is the key of the ICD that holds the message.
	ICD icd.Key
	// Message is the message to send and the binding of each field.
	Message icd.WriteMessage
}

// Resolver checks and hashes writes against the ICDs and the channels that they name.
type Resolver struct {
	icds     *icd.Service
	channels icdv0.Channels
}

// NewResolver returns a Resolver that retrieves ICDs from icds and channels with
// channels.
func NewResolver(icds *icd.Service, channels icdv0.Channels) Resolver {
	return Resolver{icds: icds, channels: channels}
}

// Check builds the encoder of each of writes from its ICD, from icds or else retrieved
// through snap, and the data types of the channels that the lookup finds. It skips a
// write whose ICD does not exist, which the reference check of the write reports. It
// also skips a write with no bindings of a JSON message with no fields, because only
// deploying refuses it. It returns the validation error of the first write that does
// not build, at the path of its message.
func (r Resolver) Check(
	ctx context.Context,
	snap gorp.Snapshot,
	writes []Write,
	icds map[icd.Key]icd.ICD,
) error {
	if len(writes) == 0 {
		return nil
	}
	missing := lo.Uniq(lo.FilterMap(writes, func(w Write, _ int) (icd.Key, bool) {
		_, ok := icds[w.ICD]
		return w.ICD, !ok
	}))
	if len(missing) > 0 {
		found, err := r.icds.NewRetrieve().
			Where(icd.MatchKeys(missing...)).
			SkipMissingKeys().
			ByKey(ctx, snap)
		if err != nil {
			return err
		}
		icds = lo.Assign(icds, found)
	}
	var keys []channel.Key
	for _, w := range writes {
		for _, b := range ChannelBindings(w.Message) {
			if b.Channel != 0 {
				keys = append(keys, b.Channel)
			}
		}
	}
	var channels map[channel.Key]icdv0.Channel
	if len(keys) > 0 {
		var err error
		if channels, err = r.channels(ctx, snap, lo.Uniq(keys)); err != nil {
			return err
		}
	}
	dataTypes := lo.MapValues(
		channels,
		func(ch icdv0.Channel, _ channel.Key) telem.DataType { return ch.DataType },
	)
	for _, w := range writes {
		ic, ok := icds[w.ICD]
		if !ok || unfilled(ic, w.Message) {
			continue
		}
		if _, err := newEncoder(w.Message, ic, dataTypes); err != nil {
			path := slices.Concat(w.Path, []string{"message"})
			return validate.PathedError(err, path...)
		}
	}
	return nil
}

// unfilled reports whether wm has no bindings and sends a JSON message of ic with no
// fields.
func unfilled(ic icd.ICD, wm icd.WriteMessage) bool {
	if len(wm.Bindings) > 0 {
		return false
	}
	m, ok := lo.Find(ic.Messages, func(m icd.Message) bool {
		return m.Key == wm.Message
	})
	if !ok {
		return false
	}
	jp, ok := m.Payload.Variant.(icd.JSONPayload)
	return ok && len(jp.Fields) == 0
}

// Hash writes to h the payload of each ICD message that writes send, retrieved through
// snap, so an edit to the format of one changes the hash. A write of a message that its
// ICD lacks writes only that the message is absent. Hash returns query.ErrNotFound when
// an ICD of writes does not exist.
func (r Resolver) Hash(
	ctx context.Context,
	snap gorp.Snapshot,
	h *content.Hash,
	writes []Write,
) error {
	if len(writes) == 0 {
		return nil
	}
	keys := lo.Uniq(lo.Map(writes, func(w Write, _ int) icd.Key { return w.ICD }))
	icds, err := r.icds.NewRetrieve().Where(icd.MatchKeys(keys...)).ByKey(ctx, snap)
	if err != nil {
		return err
	}
	for _, w := range writes {
		if err = icd.HashMessage(h, icds[w.ICD], w.Message.Message); err != nil {
			return err
		}
	}
	return nil
}
