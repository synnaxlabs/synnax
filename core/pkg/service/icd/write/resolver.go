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
	"encoding/json/v2"
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
// returns the validation error of the first write that does not build, at the path of
// its message.
func (r Resolver) Check(
	ctx context.Context,
	snap gorp.Snapshot,
	writes []Write,
	icds map[icd.Key]icd.ICD,
) error {
	if len(writes) == 0 {
		return nil
	}
	icds, err := r.retrieve(ctx, snap, writes, icds)
	if err != nil {
		return err
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
		if !ok {
			continue
		}
		if _, err = newEncoder(w.Message, ic, dataTypes); err != nil {
			path := slices.Concat(w.Path, []string{"message"})
			return validate.PathedError(err, path...)
		}
	}
	return nil
}

// Hash writes to h the payload of each ICD message that writes send, retrieved through
// snap, so an edit to the format of one changes the hash. A write of a message that its
// ICD lacks, or of an ICD that does not exist, writes only that the message is absent.
func (r Resolver) Hash(
	ctx context.Context,
	snap gorp.Snapshot,
	h *content.Hash,
	writes []Write,
) error {
	if len(writes) == 0 {
		return nil
	}
	icds, err := r.retrieve(ctx, snap, writes, nil)
	if err != nil {
		return err
	}
	for _, w := range writes {
		if err = hash(h, icds[w.ICD], w.Message); err != nil {
			return err
		}
	}
	return nil
}

// retrieve returns icds with the ICD of each of writes that icds lacks, read through
// snap. An ICD that does not exist is absent from the result.
func (r Resolver) retrieve(
	ctx context.Context,
	snap gorp.Snapshot,
	writes []Write,
	icds map[icd.Key]icd.ICD,
) (map[icd.Key]icd.ICD, error) {
	missing := lo.Uniq(lo.FilterMap(writes, func(w Write, _ int) (icd.Key, bool) {
		_, ok := icds[w.ICD]
		return w.ICD, !ok
	}))
	if len(missing) == 0 {
		return icds, nil
	}
	var found []icd.ICD
	if err := r.icds.NewRetrieve().
		Where(icd.MatchKeys(missing...)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.Assign(
		icds,
		lo.KeyBy(found, func(ic icd.ICD) icd.Key { return ic.Key }),
	), nil
}

// hash writes to h the payload of the message of ic that wm sends. When ic lacks the
// message, hash writes only that it is absent.
func hash(h *content.Hash, ic icd.ICD, wm icd.WriteMessage) error {
	m, ok := lo.Find(ic.Messages, func(m icd.Message) bool {
		return m.Key == wm.Message
	})
	h.Bool(ok)
	if !ok {
		return nil
	}
	b, err := json.Marshal(m.Payload, json.Deterministic(true))
	if err != nil {
		return err
	}
	h.Bytes(b)
	return nil
}
