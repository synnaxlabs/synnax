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
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
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
	// Framed is true when the task frames the message on a byte stream with the framing
	// of the ICD. The Driver then encodes binary and text messages as well as JSON, and
	// an edit to the framing changes the hash.
	Framed bool
}

// Targets returns the framed write of each of targets, at targets.i of a config.
func Targets(targets []icd.WriteTarget) []Write {
	return lo.Map(targets, func(t icd.WriteTarget, i int) Write {
		return Write{
			Path:    []string{"targets", strconv.Itoa(i)},
			ICD:     t.ICD,
			Message: t.Message,
			Framed:  true,
		}
	})
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
// deploying refuses it. A framed write of a binary or text message builds no encoder,
// so only its bindings are checked. It returns the validation error of the first write
// that does not build, at the path of its message.
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
		if err := check(w, ic, dataTypes); err != nil {
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
		if w.Framed {
			if err = icd.HashFraming(h, icds[w.ICD]); err != nil {
				return err
			}
		}
		if err = icd.HashMessage(h, icds[w.ICD], w.Message.Message); err != nil {
			return err
		}
	}
	return nil
}

// check builds the encoder of w from ic, or checks only the bindings of a framed write
// of a binary or text message.
func check(w Write, ic icd.ICD, dataTypes map[channel.Key]telem.DataType) error {
	if w.Framed {
		m, ok := lo.Find(ic.Messages, func(m icd.Message) bool {
			return m.Key == w.Message.Message
		})
		if _, isJSON := m.Payload.Variant.(icd.JSONPayload); ok && !isJSON {
			return checkBindings(w.Message, m)
		}
	}
	_, err := newEncoder(w.Message, ic, dataTypes)
	return err
}

// checkBindings returns a validation error when wm binds no channels, a binding names
// no field of m, a field has two bindings, a value is not of its field's kind, or a
// field has no binding and no default.
func checkBindings(wm icd.WriteMessage, m icd.Message) error {
	if err := wm.Validate(); err != nil {
		return err
	}
	if !slices.ContainsFunc(wm.Bindings, func(b icd.WriteBinding) bool {
		return b.Type() == icd.ChannelWriteBindingType
	}) {
		return validate.Errorf("message %s binds no channels", m.Name)
	}
	fields := fieldsOf(m.Payload)
	at := make(map[icd.FieldKey]int, len(fields))
	for i, f := range fields {
		at[f.key] = i
	}
	bound := make(set.Set[icd.FieldKey], len(wm.Bindings))
	for j, b := range wm.Bindings {
		pathed := func(err error, segment string) error {
			return validate.PathedError(err, "bindings", strconv.Itoa(j), segment)
		}
		key := b.Base().Field
		i, ok := at[key]
		switch {
		case !ok:
			return pathed(validate.Errorf("field %s does not exist", key), "field")
		case bound.Contains(key):
			return pathed(validate.Errorf(
				"field %s has more than one binding", fields[i].name,
			), "field")
		}
		bound.Add(key)
		if v, isValue := b.Variant.(icd.ValueWriteBinding); isValue &&
			v.Value.Type() != fields[i].kind {
			return pathed(validate.Errorf(
				"field %s of kind %s needs a value of its kind",
				fields[i].name, fields[i].kind,
			), "value")
		}
	}
	for _, f := range fields {
		if !bound.Contains(f.key) && !f.defaulted {
			return validate.Errorf("field %s has no binding and no default", f.name)
		}
	}
	return nil
}

// field is the part of a binary or text message field that a write checks.
type field struct {
	key       icd.FieldKey
	name      string
	kind      payload.Kind
	defaulted bool
}

// fieldsOf returns the fields of p, a binary or text payload, in order. A text field
// encodes a float.
func fieldsOf(p icd.Payload) []field {
	switch v := p.Variant.(type) {
	case icd.BinaryPayload:
		return lo.Map(v.Fields, func(f icd.BinaryField, _ int) field {
			b := f.Base()
			return field{
				key:       b.Key,
				name:      b.Name,
				kind:      payload.Kind(f.Kind()),
				defaulted: binaryDefaulted(f),
			}
		})
	case icd.TextPayload:
		return lo.Map(v.Fields, func(f icd.TextField, _ int) field {
			b := f.Base()
			return field{
				key:       b.Key,
				name:      b.Name,
				kind:      payload.KindFloat,
				defaulted: b.Default != nil,
			}
		})
	default:
		panic(errors.Newf("write: payload variant %T is not binary or text", v))
	}
}

// binaryDefaulted returns true when f has a default.
func binaryDefaulted(f icd.BinaryField) bool {
	switch v := f.Variant.(type) {
	case icd.FloatBinaryField:
		return v.Default != nil
	case icd.IntegerBinaryField:
		return v.Default != nil
	case icd.UnsignedBinaryField:
		return v.Default != nil
	}
	return false
}
