// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/validate"
)

// readChannels returns the v2.Channels that reads channels from svc.
func readChannels(svc *channel.Service) v2.Channels {
	return func(
		ctx context.Context,
		snap gorp.Snapshot,
		keys []channel.Key,
	) (map[channel.Key]v2.Channel, error) {
		var found []channel.Channel
		if err := svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		return lo.SliceToMap(found, func(c channel.Channel) (channel.Key, v2.Channel) {
			return c.Key(), v2.Channel{
				Name:     c.Name,
				DataType: c.DataType,
				Index:    c.IsIndex,
			}
		}), nil
	}
}

// createICD returns the v2.CreateICD that creates an ICD with svc.
func createICD(svc *icd.Service) v2.CreateICD {
	return func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error {
		return svc.NewWriter(tx).Create(ctx, &ic)
	}
}

// readResolver resolves read configs against the ICDs and channels they read.
type readResolver struct {
	icds     *icd.Service
	channels v2.Channels
	create   v2.CreateICD
	ins      alamos.Instrumentation
}

// resolve moves the inline entries of cfg into a new ICD named name, then checks each
// enabled entry of cfg against its ICD and its channels, read through snap. It returns
// the write that creates the ICD, or nil when cfg holds no inline entries.
func (r readResolver) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	name string,
	cfg *ReadConfig,
) (func(context.Context, gorp.Tx) error, error) {
	staged, ok, err := v2.ResolveReadConfig(
		ctx, snap, r.channels, name, cfg, v2.NoteLift(r.ins, cfg.Key),
	)
	if err != nil {
		return nil, err
	}
	icds := make(map[icd.Key]icd.ICD)
	if ok {
		icds[staged.Key] = staged
	}
	if err = r.check(ctx, snap, cfg, icds); err != nil || !ok {
		return nil, err
	}
	return func(ctx context.Context, tx gorp.Tx) error {
		return r.create(ctx, tx, staged)
	}, nil
}

// check checks each enabled entry of cfg against its ICD, from icds or else read
// through snap, and its channels. It skips an ICD or a channel that does not exist,
// which the reference check of the write reports.
func (r readResolver) check(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) error {
	var (
		icdKeys     []icd.Key
		channelKeys []channel.Key
	)
	for _, e := range cfg.Entries {
		switch entry := e.Variant.(type) {
		case PlainReadEntry:
			if entry.Disabled {
				continue
			}
			if _, ok := icds[entry.ICD]; !ok {
				icdKeys = append(icdKeys, entry.ICD)
			}
			for _, m := range entry.Messages {
				for _, b := range m.Bindings {
					if b.Channel != 0 {
						channelKeys = append(channelKeys, b.Channel)
					}
				}
			}
		case SparkplugReadEntry:
			if !entry.Disabled && entry.Channel != 0 {
				channelKeys = append(channelKeys, entry.Channel)
			}
		}
	}
	if len(icdKeys) > 0 {
		var found []icd.ICD
		if err := r.icds.NewRetrieve().
			Where(icd.MatchKeys(lo.Uniq(icdKeys)...)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return err
		}
		for _, ic := range found {
			icds[ic.Key] = ic
		}
	}
	var channels map[channel.Key]v2.Channel
	if len(channelKeys) > 0 {
		var err error
		if channels, err = r.channels(ctx, snap, lo.Uniq(channelKeys)); err != nil {
			return err
		}
	}
	// written holds the path of the binding or tag that writes to each channel.
	written := make(map[channel.Key]string)
	claim := func(key channel.Key, path []string) error {
		if other, ok := written[key]; ok {
			return validate.PathedError(
				validate.Errorf("channel %d is also written by %s", key, other),
				path...,
			)
		}
		written[key] = strings.Join(path, ".")
		return nil
	}
	for i, e := range cfg.Entries {
		path := []string{"entries", strconv.Itoa(i)}
		switch entry := e.Variant.(type) {
		case PlainReadEntry:
			ic, ok := icds[entry.ICD]
			if entry.Disabled || !ok {
				continue
			}
			for j, rm := range entry.Messages {
				if err := checkReadMessage(
					rm,
					ic,
					channels,
					claim,
					slices.Concat(path, []string{"messages", strconv.Itoa(j)}),
				); err != nil {
					return err
				}
			}
		case SparkplugReadEntry:
			if entry.Disabled || entry.Channel == 0 {
				continue
			}
			if err := claim(
				entry.Channel, slices.Concat(path, []string{"channel"}),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkReadMessage checks rm, at path, against its message in ic and the channels of
// its bindings, and claims each channel of a binding.
func checkReadMessage(
	rm ReadMessage,
	ic icd.ICD,
	channels map[channel.Key]v2.Channel,
	claim func(key channel.Key, path []string) error,
	path []string,
) error {
	at := func(err error, segments ...string) error {
		return validate.PathedError(err, slices.Concat(path, segments)...)
	}
	i := slices.IndexFunc(ic.Messages, func(m icd.Message) bool {
		return m.Key == rm.Message
	})
	if i < 0 {
		return at(validate.Errorf(
			"message %s does not exist in ICD %s", rm.Message, ic.Name,
		), "message")
	}
	msg := ic.Messages[i]
	jp, ok := msg.Payload.Variant.(icd.JSONPayload)
	if !ok {
		return at(
			validate.Errorf("message %s is not a JSON message", msg.Name),
			"message",
		)
	}
	fields := lo.SliceToMap(
		jp.Fields,
		func(f icd.JSONField) (icd.FieldKey, icd.JSONField) {
			return f.Base().Key, f
		},
	)
	if rm.Index != nil {
		f, ok := fields[*rm.Index]
		switch {
		case !ok:
			return at(validate.Errorf(
				"field %s does not exist in message %s", *rm.Index, msg.Name,
			), "index")
		case f.Type() != payload.KindTimestamp:
			return at(validate.Errorf(
				"the index field %s must be a timestamp field", f.Base().Name,
			), "index")
		case f.Variant.(icd.TimestampJSONField).Default != nil:
			return at(validate.Errorf(
				"the index field %s cannot take a default", f.Base().Name,
			), "index")
		case !slices.ContainsFunc(rm.Bindings, func(b ReadBinding) bool {
			return b.Field == *rm.Index
		}):
			return at(validate.Errorf(
				"the index field %s has no binding", f.Base().Name,
			), "index")
		}
	}
	for j, b := range rm.Bindings {
		index := strconv.Itoa(j)
		f, ok := fields[b.Field]
		if !ok {
			return at(validate.Errorf(
				"field %s does not exist in message %s", b.Field, msg.Name,
			), "bindings", index, "field")
		}
		kind, name := f.Type(), f.Base().Name
		if kind == payload.KindBytes && f.Base().Pointer == "" {
			return at(validate.Errorf(
				"bytes field %s requires a JSON pointer", name,
			), "bindings", index, "field")
		}
		if b.DataType != "" && !codec.Converts(kind, b.DataType) {
			return at(validate.Errorf(
				"field %s of kind %s cannot read into data type %s",
				name, kind, b.DataType,
			), "bindings", index, "data_type")
		}
		ch, ok := channels[b.Channel]
		if !ok {
			continue
		}
		channelPath := slices.Concat(path, []string{"bindings", index, "channel"})
		if !codec.Converts(kind, ch.DataType) {
			return validate.PathedError(validate.Errorf(
				"field %s of kind %s cannot read into channel %s of data type %s",
				name, kind, ch.Name, ch.DataType,
			), channelPath...)
		}
		isIndex := rm.Index != nil && *rm.Index == b.Field
		if isIndex && !ch.Index {
			return validate.PathedError(validate.Errorf(
				"the index field %s must write to an index channel", name,
			), channelPath...)
		}
		if !isIndex && ch.Index {
			return validate.PathedError(validate.Errorf(
				"only the index field can write to index channel %s", ch.Name,
			), channelPath...)
		}
		if err := claim(b.Channel, channelPath); err != nil {
			return err
		}
	}
	return nil
}
