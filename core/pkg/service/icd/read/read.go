// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package read checks and hashes the reads of ICD messages that task configs hold,
// against their ICDs and channels.
package read

import (
	"context"
	"encoding/json/v2"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/validate"
)

// Read is a read of messages of an ICD, at a path of a task config.
type Read struct {
	// Path is the path of the read in its config.
	Path []string
	// ICD is the key of the ICD that holds the messages.
	ICD icd.Key
	// Messages are the messages to read and the channel of each field.
	Messages []icd.ReadMessage
}

// Resolver checks and hashes reads against the ICDs and the channels that they name.
type Resolver struct {
	icds     *icd.Service
	channels icdv0.Channels
}

// NewResolver returns a Resolver that retrieves ICDs from icds and channels with
// channels.
func NewResolver(icds *icd.Service, channels icdv0.Channels) Resolver {
	return Resolver{icds: icds, channels: channels}
}

// NewChecker returns the Checker of reads and of tags, the channels that Claim takes
// beyond the bindings of reads. It holds each ICD from icds or else retrieved through
// snap, and the channels of the bindings and the tags. The Checker skips an ICD or a
// channel that does not exist, which the reference check of the write reports.
func (r Resolver) NewChecker(
	ctx context.Context,
	snap gorp.Snapshot,
	reads []Read,
	tags []channel.Key,
	icds map[icd.Key]icd.ICD,
) (Checker, error) {
	c := Checker{
		icds:    make(map[icd.Key]icd.ICD, len(icds)),
		written: make(map[channel.Key]string),
	}
	maps.Copy(c.icds, icds)
	var (
		icdKeys     []icd.Key
		channelKeys = slices.Clone(tags)
	)
	for _, rd := range reads {
		if _, ok := c.icds[rd.ICD]; !ok {
			icdKeys = append(icdKeys, rd.ICD)
		}
		for _, m := range rd.Messages {
			for _, b := range m.Bindings {
				if b.Channel != 0 {
					channelKeys = append(channelKeys, b.Channel)
				}
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
			return Checker{}, err
		}
		for _, ic := range found {
			c.icds[ic.Key] = ic
		}
	}
	if len(channelKeys) > 0 {
		var err error
		if c.channels, err = r.channels(ctx, snap, lo.Uniq(channelKeys)); err != nil {
			return Checker{}, err
		}
	}
	return c, nil
}

// Hash writes to h the payload of each ICD message that reads read, retrieved through
// snap, so an edit to the format of one changes the hash. A read of a message that its
// ICD lacks writes only that the message is absent. Hash returns query.ErrNotFound
// when an ICD of reads does not exist.
func (r Resolver) Hash(
	ctx context.Context,
	snap gorp.Snapshot,
	h *content.Hash,
	reads []Read,
) error {
	if len(reads) == 0 {
		return nil
	}
	keys := lo.Uniq(lo.Map(reads, func(rd Read, _ int) icd.Key { return rd.ICD }))
	var found []icd.ICD
	if err := r.icds.NewRetrieve().
		Where(icd.MatchKeys(keys...)).
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return err
	}
	icds := lo.KeyBy(found, func(ic icd.ICD) icd.Key { return ic.Key })
	for _, rd := range reads {
		if err := hash(h, icds[rd.ICD], rd.Messages); err != nil {
			return err
		}
	}
	return nil
}

// Checker checks the reads of one task config. It passes a read of an ICD or a binding
// to a channel that it does not hold, which the reference check of the write reports.
// It is not safe for concurrent use.
type Checker struct {
	icds     map[icd.Key]icd.ICD
	channels map[channel.Key]icdv0.Channel
	// written holds the path of the binding or tag that writes to each channel.
	written map[channel.Key]string
}

// Read checks each message of rd against its ICD and the channels of its bindings, and
// claims each channel that the message writes: the channel of each binding, and each
// index channel of the bound channels, whose timestamps the task writes when no index
// field binds it. It returns a validation error at the path of the first problem.
func (c Checker) Read(rd Read) error {
	ic, ok := c.icds[rd.ICD]
	if !ok {
		return nil
	}
	for j, rm := range rd.Messages {
		path := slices.Concat(rd.Path, []string{"messages", strconv.Itoa(j)})
		if err := c.message(rm, ic, path); err != nil {
			return err
		}
	}
	return nil
}

// Claim takes key, the channel of the tag at path, and the index channel of key when
// the Checker holds key. It returns a validation error when another binding or tag of
// the config already writes to either.
func (c Checker) Claim(key channel.Key, path []string) error {
	if err := c.claim(key, path); err != nil {
		return err
	}
	if ch, ok := c.channels[key]; ok && ch.IndexKey != 0 && ch.IndexKey != key {
		return c.claim(ch.IndexKey, path)
	}
	return nil
}

// claim takes key for the binding or tag at path. It returns a validation error when
// another binding or tag of the config already writes to key.
func (c Checker) claim(key channel.Key, path []string) error {
	if other, ok := c.written[key]; ok {
		return validate.PathedError(
			validate.Errorf("channel %d is also written by %s", key, other),
			path...,
		)
	}
	c.written[key] = strings.Join(path, ".")
	return nil
}

// message checks rm, at path, against its message in ic and the channels of its
// bindings, and claims each channel that it writes.
func (c Checker) message(rm icd.ReadMessage, ic icd.ICD, path []string) error {
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
	var (
		// indexes holds each index channel of the bound channels, in binding order.
		indexes []channel.Key
		// indexChannel is the channel of the index field.
		indexChannel channel.Key
	)
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
		if isIndex {
			indexChannel = b.Channel
		}
		ch, ok := c.channels[b.Channel]
		if ok && ch.IndexKey != 0 && !slices.Contains(indexes, ch.IndexKey) {
			indexes = append(indexes, ch.IndexKey)
		}
	}
	for _, key := range indexes {
		if key == indexChannel {
			continue
		}
		if err := c.claim(key, path); err != nil {
			return err
		}
	}
	return nil
}

// checkIndex checks the index field of rm, which has one, against fields, the fields
// of the message named message.
func checkIndex(
	rm icd.ReadMessage,
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
	case !slices.ContainsFunc(rm.Bindings, func(b icd.ReadBinding) bool {
		return b.Field == *rm.Index
	}):
		return validate.Errorf("the index field %s has no binding", f.Base().Name)
	}
	return nil
}

// checkBinding checks b, at path, against f, the field it binds, and its channel, and
// claims the channel. isIndex is true when f is the index field of its message.
func (c Checker) checkBinding(
	b icd.ReadBinding,
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
			"field %s cannot write to index channel %s, because only the index field "+
				"can. Bind field %s to a data channel",
			name, ch.Name, name,
		), "channel")
	}
	return c.claim(b.Channel, slices.Concat(path, []string{"channel"}))
}

// hash writes to h the payload of the message of ic that each of rms reads.
func hash(h *content.Hash, ic icd.ICD, rms []icd.ReadMessage) error {
	for _, rm := range rms {
		m, ok := lo.Find(ic.Messages, func(m icd.Message) bool {
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
	return nil
}
