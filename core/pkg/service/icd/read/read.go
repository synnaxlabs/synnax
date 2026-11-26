// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package read checks and hashes the reads of ICD messages that task configs hold,
// against their ICDs and channels, and decodes the documents of reads into samples.
package read

import (
	"context"
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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
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
	// Framed is true when the task splits a byte stream into messages with the framing
	// of the ICD. The Driver then decodes binary and text messages as well as JSON, and
	// an edit to the framing changes the hash.
	Framed bool
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

// NewChecker returns the Checker of reads and of tags, the channels that Tag takes
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
		found, err := r.icds.NewRetrieve().
			Where(icd.MatchKeys(lo.Uniq(icdKeys)...)).
			SkipMissingKeys().
			ByKey(ctx, snap)
		if err != nil {
			return Checker{}, err
		}
		maps.Copy(c.icds, found)
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
// ICD lacks writes only that the message is absent. Hash returns query.ErrNotFound when
// an ICD of reads does not exist.
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
	icds, err := r.icds.NewRetrieve().Where(icd.MatchKeys(keys...)).ByKey(ctx, snap)
	if err != nil {
		return err
	}
	for _, rd := range reads {
		if rd.Framed {
			if err = icd.HashFraming(h, icds[rd.ICD]); err != nil {
				return err
			}
		}
		for _, rm := range rd.Messages {
			if err = icd.HashMessage(h, icds[rd.ICD], rm.Message); err != nil {
				return err
			}
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
// claims each channel of a binding. It returns a validation error at the path of the
// first problem.
func (c Checker) Read(rd Read) error {
	ic, ok := c.icds[rd.ICD]
	if !ok {
		return nil
	}
	for j, rm := range rd.Messages {
		path := slices.Concat(rd.Path, []string{"messages", strconv.Itoa(j)})
		if err := c.message(rm, ic, rd.Framed, path); err != nil {
			return err
		}
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

// Tag checks key, the channel of the tag at path, and claims it. takes reports whether
// a channel of a data type can take the values of the tag. Tag returns a validation
// error when key is an index channel or cannot take the values of the tag, or when
// another binding or tag already writes to key.
func (c Checker) Tag(
	key channel.Key,
	path []string,
	takes func(telem.DataType) bool,
) error {
	if ch, ok := c.channels[key]; ok {
		if ch.Index {
			return validate.PathedError(
				validate.Errorf("channel %s is an index channel", ch.Name),
				path...,
			)
		}
		if !takes(ch.DataType) {
			return validate.PathedError(validate.Errorf(
				"channel %s of data type %s cannot take the values of a tag",
				ch.Name, ch.DataType,
			), path...)
		}
	}
	return c.claim(key, path)
}

// ClaimIndexes claims, for each message of rd, each index channel of its bound channels
// that its index field does not bind, since the task writes its timestamps. Run it
// after Read and Tag, so that an error is at the path of the message that does not
// bind it.
func (c Checker) ClaimIndexes(rd Read) error {
	for j, rm := range rd.Messages {
		path := slices.Concat(rd.Path, []string{"messages", strconv.Itoa(j)})
		var (
			// indexes holds each index channel of the bound channels, in binding order.
			indexes []channel.Key
			// indexChannel is the channel of the index field.
			indexChannel channel.Key
		)
		for _, b := range rm.Bindings {
			if rm.Index != nil && *rm.Index == b.Field {
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
	}
	return nil
}

// ClaimIndex claims the index channel of key, the channel of the tag at path, whose
// timestamps the task writes. It returns a validation error when another binding or
// tag already writes to it.
func (c Checker) ClaimIndex(key channel.Key, path []string) error {
	if ch, ok := c.channels[key]; ok && ch.IndexKey != 0 {
		return c.claim(ch.IndexKey, path)
	}
	return nil
}

// message checks rm, at path, against its message in ic and the channels of its
// bindings, and claims each channel of a binding. A message that is not JSON passes
// only when framed is true.
func (c Checker) message(
	rm icd.ReadMessage,
	ic icd.ICD,
	framed bool,
	path []string,
) error {
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
	if _, isJSON := msg.Payload.Variant.(icd.JSONPayload); !isJSON && !framed {
		return at(
			validate.Errorf("message %s is not a JSON message", msg.Name),
			"message",
		)
	}
	fields := fieldsOf(msg.Payload)
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
	rm icd.ReadMessage,
	fields map[icd.FieldKey]field,
	message string,
) error {
	f, ok := fields[*rm.Index]
	switch {
	case !ok:
		return validate.Errorf(
			"field %s does not exist in message %s", *rm.Index, message,
		)
	case f.kind != payload.KindTimestamp:
		return validate.Errorf("the index field %s must be a timestamp field", f.name)
	case f.defaulted:
		return validate.Errorf("the index field %s cannot take a default", f.name)
	case !slices.ContainsFunc(rm.Bindings, func(b icd.ReadBinding) bool {
		return b.Field == *rm.Index
	}):
		return validate.Errorf("the index field %s has no binding", f.name)
	}
	return nil
}

// checkBinding checks b, at path, against f, the field it binds, and its channel, and
// claims the channel. isIndex is true when f is the index field of its message.
func (c Checker) checkBinding(
	b icd.ReadBinding,
	f field,
	isIndex bool,
	path []string,
) error {
	at := func(err error, segment string) error {
		return validate.PathedError(err, slices.Concat(path, []string{segment})...)
	}
	name := f.name
	if err := icd.CheckScale(f.scale); err != nil {
		return at(errors.Wrapf(err, "field %s", name), "field")
	}
	if f.unplaced {
		return at(validate.Errorf(
			"bytes field %s requires a JSON pointer", name,
		), "field")
	}
	if b.DataType != "" {
		if err := f.check(b.DataType, "data type "+string(b.DataType)); err != nil {
			return at(err, "data_type")
		}
	}
	ch, ok := c.channels[b.Channel]
	if !ok {
		return nil
	}
	target := "channel " + ch.Name + " of data type " + string(ch.DataType)
	if err := f.check(ch.DataType, target); err != nil {
		return at(err, "channel")
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

// field is the part of a message field that a read checks.
type field struct {
	name string
	kind payload.Kind
	// defaulted is true for a timestamp field with a default, which cannot be the
	// index.
	defaulted bool
	// unplaced is true for a JSON bytes field with no pointer, which a read cannot
	// find.
	unplaced bool
	scale    float64
	// scaled is true for a field with a scale or an offset, which decodes only into a
	// float channel.
	scaled bool
	// bitLength is the width of a binary field, and zero for other fields.
	bitLength uint8
}

// check returns a validation error when f cannot decode into target, a channel of data
// type dt.
func (f field) check(dt telem.DataType, target string) error {
	if !codec.Converts(f.kind, dt, f.scaled) {
		noun := "field"
		if f.scaled {
			noun = "scaled field"
		}
		return validate.Errorf(
			"%s %s of kind %s cannot read into %s", noun, f.name, f.kind, target,
		)
	}
	k, _ := payload.KindOf(dt)
	if f.bitLength == 0 || (k != payload.KindInteger && k != payload.KindUnsigned) {
		return nil
	}
	// A signed channel takes one more bit for the sign of an unsigned field.
	needed := int(f.bitLength)
	if k == payload.KindInteger && f.kind == payload.KindUnsigned {
		needed++
	}
	if needed > int(dt.Density())*8 {
		return validate.Errorf(
			"field %s of kind %d-bit %s cannot read into %s",
			f.name, f.bitLength, f.kind, target,
		)
	}
	return nil
}

// fieldsOf returns the fields of p by key. A text field decodes as a float.
func fieldsOf(p icd.Payload) map[icd.FieldKey]field {
	fields := make(map[icd.FieldKey]field)
	switch v := p.Variant.(type) {
	case icd.JSONPayload:
		for _, f := range v.Fields {
			b := f.Base()
			ts, isTimestamp := f.Variant.(icd.TimestampJSONField)
			fields[b.Key] = field{
				name:      b.Name,
				kind:      f.Type(),
				defaulted: isTimestamp && ts.Default != nil,
				unplaced:  f.Type() == payload.KindBytes && b.Pointer == "",
				scale:     b.Scale,
				scaled:    b.Scale != 1 || b.Offset != 0,
			}
		}
	case icd.BinaryPayload:
		for _, f := range v.Fields {
			b := f.Base()
			fields[b.Key] = field{
				name:      b.Name,
				kind:      payload.Kind(f.Kind()),
				scale:     b.Scale,
				scaled:    b.Scale != 1 || b.Offset != 0,
				bitLength: b.BitLength,
			}
		}
	case icd.TextPayload:
		for _, f := range v.Fields {
			b := f.Base()
			fields[b.Key] = field{name: b.Name, kind: payload.KindFloat, scale: b.Scale}
		}
	default:
		panic(errors.Newf("read: unknown payload variant %T", v))
	}
	return fields
}
