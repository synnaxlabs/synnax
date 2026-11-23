// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package read checks the reads of ICD messages that task configs hold against their
// ICDs and channels.
package read

import (
	"encoding/json/v2"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Channel is what a check needs of a channel.
type Channel struct {
	// Name is the name of the channel.
	Name string
	// DataType is the data type of the channel.
	DataType telem.DataType
	// Index is true when the channel is an index channel.
	Index bool
}

// Checker checks the reads of one task config. It passes a binding to a channel it
// does not hold, which the reference check of the write reports. It is not safe for
// concurrent use.
type Checker struct {
	channels map[channel.Key]Channel
	// written holds the path of the binding or tag that writes to each channel.
	written map[channel.Key]string
}

// NewChecker returns a Checker of bindings to channels.
func NewChecker(channels map[channel.Key]Channel) Checker {
	return Checker{channels: channels, written: make(map[channel.Key]string)}
}

// Claim takes key for the binding or tag at path. It returns a validation error when
// another binding or tag of the config already writes to key.
func (c Checker) Claim(key channel.Key, path []string) error {
	if other, ok := c.written[key]; ok {
		return validate.PathedError(
			validate.Errorf("channel %d is also written by %s", key, other),
			path...,
		)
	}
	c.written[key] = strings.Join(path, ".")
	return nil
}

// Message checks rm, at path, against its message in ic and the channels of its
// bindings, and claims each channel of a binding. It returns a validation error at the
// path of the first problem.
func (c Checker) Message(rm icd.ReadMessage, ic icd.ICD, path []string) error {
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
			"only the index field can write to index channel %s", ch.Name,
		), "channel")
	}
	return c.Claim(b.Channel, slices.Concat(path, []string{"channel"}))
}

// Hash writes to h the payload of the message of ic that each of rms reads, so an edit
// to the format of one changes the hash. A read of a message that ic lacks writes only
// that the message is absent.
func Hash(h *content.Hash, ic icd.ICD, rms []icd.ReadMessage) error {
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
