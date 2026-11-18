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
	"encoding/json/v2"
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
	"github.com/synnaxlabs/x/content"
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

// hash writes to h the payload of each ICD message that an enabled plain entry of cfg
// reads, so an edit to the format of one changes the config hash of the task.
func (r readResolver) hash(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg *ReadConfig,
	h *content.Hash,
) error {
	icdKeys, _ := referencedKeys(cfg, nil)
	if len(icdKeys) == 0 {
		return nil
	}
	var found []icd.ICD
	if err := r.icds.NewRetrieve().
		Where(icd.MatchKeys(lo.Uniq(icdKeys)...)).
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return err
	}
	icds := lo.KeyBy(found, func(ic icd.ICD) icd.Key { return ic.Key })
	for _, e := range cfg.Entries {
		entry, ok := e.Variant.(PlainReadEntry)
		if !ok || entry.Disabled {
			continue
		}
		for _, rm := range entry.Messages {
			m, ok := lo.Find(icds[entry.ICD].Messages, func(m icd.Message) bool {
				return m.Key == rm.Message
			})
			h.Bool(ok)
			if !ok {
				continue
			}
			b, err := json.Marshal(m.Payload, json.Deterministic(true))
			if err != nil {
				return err
			}
			h.Bytes(b)
		}
	}
	return nil
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
	icdKeys, channelKeys := referencedKeys(cfg, icds)
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
	c := readChecker{icds: icds, written: make(map[channel.Key]string)}
	if len(channelKeys) > 0 {
		var err error
		if c.channels, err = r.channels(ctx, snap, lo.Uniq(channelKeys)); err != nil {
			return err
		}
	}
	for i, e := range cfg.Entries {
		if err := c.checkEntry(e, []string{"entries", strconv.Itoa(i)}); err != nil {
			return err
		}
	}
	return nil
}

// referencedKeys returns the ICDs that the enabled entries of cfg name and icds lacks,
// and the channels that the enabled entries name.
func referencedKeys(
	cfg *ReadConfig,
	icds map[icd.Key]icd.ICD,
) (icdKeys []icd.Key, channelKeys []channel.Key) {
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
	return icdKeys, channelKeys
}

// readChecker checks the entries of a read config against their ICDs and channels.
type readChecker struct {
	icds     map[icd.Key]icd.ICD
	channels map[channel.Key]v2.Channel
	// written holds the path of the binding or tag that writes to each channel.
	written map[channel.Key]string
}

// claim takes key for the binding or tag at path.
func (c readChecker) claim(key channel.Key, path []string) error {
	if other, ok := c.written[key]; ok {
		return validate.PathedError(
			validate.Errorf("channel %d is also written by %s", key, other),
			path...,
		)
	}
	c.written[key] = strings.Join(path, ".")
	return nil
}

// checkEntry checks e, at path, when it is enabled. It skips a plain entry whose ICD
// does not exist.
func (c readChecker) checkEntry(e ReadEntry, path []string) error {
	switch entry := e.Variant.(type) {
	case PlainReadEntry:
		ic, ok := c.icds[entry.ICD]
		if entry.Disabled || !ok {
			return nil
		}
		for j, rm := range entry.Messages {
			messagePath := slices.Concat(path, []string{"messages", strconv.Itoa(j)})
			if err := c.checkMessage(rm, ic, messagePath); err != nil {
				return err
			}
		}
	case SparkplugReadEntry:
		if !entry.Disabled && entry.Channel != 0 {
			return c.claim(entry.Channel, slices.Concat(path, []string{"channel"}))
		}
	}
	return nil
}

// checkMessage checks rm, at path, against its message in ic and the channels of its
// bindings, and claims each channel of a binding.
func (c readChecker) checkMessage(rm ReadMessage, ic icd.ICD, path []string) error {
	at := func(err error, segment string) error {
		return validate.PathedError(err, slices.Concat(path, []string{segment})...)
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
		if err := checkIndex(rm, fields, msg.Name); err != nil {
			return at(err, "index")
		}
	}
	for j, b := range rm.Bindings {
		f, ok := fields[b.Field]
		bindingPath := slices.Concat(path, []string{"bindings", strconv.Itoa(j)})
		if !ok {
			return validate.PathedError(validate.Errorf(
				"field %s does not exist in message %s", b.Field, msg.Name,
			), slices.Concat(bindingPath, []string{"field"})...)
		}
		isIndex := rm.Index != nil && *rm.Index == b.Field
		if err := c.checkBinding(b, f, isIndex, bindingPath); err != nil {
			return err
		}
	}
	return nil
}

// checkIndex checks the index field of rm, which has one, against fields, the fields
// of the message named message.
func checkIndex(
	rm ReadMessage,
	fields map[icd.FieldKey]icd.JSONField,
	message string,
) error {
	f, ok := fields[*rm.Index]
	timestamp, isTimestamp := f.Variant.(icd.TimestampJSONField)
	switch {
	case !ok:
		return validate.Errorf(
			"field %s does not exist in message %s", *rm.Index, message,
		)
	case !isTimestamp:
		return validate.Errorf(
			"the index field %s must be a timestamp field", f.Base().Name,
		)
	case timestamp.Default != nil:
		return validate.Errorf(
			"the index field %s cannot take a default", f.Base().Name,
		)
	case !slices.ContainsFunc(rm.Bindings, func(b ReadBinding) bool {
		return b.Field == *rm.Index
	}):
		return validate.Errorf("the index field %s has no binding", f.Base().Name)
	}
	return nil
}

// checkBinding checks b, at path, against f, the field it binds, and its channel, and
// claims the channel. isIndex is true when f is the index field of its message.
func (c readChecker) checkBinding(
	b ReadBinding,
	f icd.JSONField,
	isIndex bool,
	path []string,
) error {
	at := func(err error, segment string) error {
		return validate.PathedError(err, slices.Concat(path, []string{segment})...)
	}
	kind, name := f.Type(), f.Base().Name
	if kind == payload.KindBytes && f.Base().Pointer == "" {
		return at(validate.Errorf(
			"bytes field %s requires a JSON pointer", name,
		), "field")
	}
	if b.DataType != "" && !codec.Converts(kind, b.DataType) {
		return at(validate.Errorf(
			"field %s of kind %s cannot read into data type %s",
			name, kind, b.DataType,
		), "data_type")
	}
	ch, ok := c.channels[b.Channel]
	if !ok {
		return nil
	}
	if !codec.Converts(kind, ch.DataType) {
		return at(validate.Errorf(
			"field %s of kind %s cannot read into channel %s of data type %s",
			name, kind, ch.Name, ch.DataType,
		), "channel")
	}
	if isIndex && !ch.Index {
		return at(validate.Errorf(
			"the index field %s must write to an index channel", name,
		), "channel")
	}
	if !isIndex && ch.Index {
		return at(validate.Errorf(
			"only the index field can write to index channel %s", ch.Name,
		), "channel")
	}
	return c.claim(b.Channel, slices.Concat(path, []string{"channel"}))
}
