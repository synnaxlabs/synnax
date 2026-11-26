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
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
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

// check builds the encoder of w from ic, or checks a framed write of a binary or text
// message with the rules of the Driver's encoder.
func check(w Write, ic icd.ICD, dataTypes map[channel.Key]telem.DataType) error {
	if w.Framed {
		m, ok := lo.Find(ic.Messages, func(m icd.Message) bool {
			return m.Key == w.Message.Message
		})
		if _, isJSON := m.Payload.Variant.(icd.JSONPayload); ok && !isJSON {
			return checkBindings(w.Message, m, dataTypes)
		}
	}
	_, err := newEncoder(w.Message, ic, dataTypes)
	return err
}

// checkBindings returns a validation error when wm cannot send m, a binary or text
// message. dataTypes holds the data type of each channel that the lookup finds. A field
// that holds the identifier takes no binding. A field that a multiplexor with a fixed
// value never selects takes no channel and needs no default.
func checkBindings(
	wm icd.WriteMessage,
	m icd.Message,
	dataTypes map[channel.Key]telem.DataType,
) error {
	if err := wm.Validate(); err != nil {
		return err
	}
	if !slices.ContainsFunc(wm.Bindings, func(b icd.WriteBinding) bool {
		return b.Type() == icd.ChannelWriteBindingType
	}) {
		return validate.Errorf("message %s binds no channels", m.Name)
	}
	p := m.Payload
	fields := fieldsOf(p)
	at := fieldIndexes(fields)
	bound := make(set.Set[icd.FieldKey], len(wm.Bindings))
	channels := make(map[icd.FieldKey]int, len(wm.Bindings))
	for j, b := range wm.Bindings {
		pathed := func(err error, segments ...string) error {
			path := slices.Concat([]string{"bindings", strconv.Itoa(j)}, segments)
			return validate.PathedError(err, path...)
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
		case fields[i].identifier:
			return pathed(validate.Errorf(
				"field %s holds the identifier, so it takes no binding", fields[i].name,
			), "field")
		}
		bound.Add(key)
		switch v := b.Variant.(type) {
		case icd.ChannelWriteBinding:
			channels[key] = j
			if err := checkChannel(fields[i], v, dataTypes); err != nil {
				return pathed(err)
			}
		case icd.ValueWriteBinding:
			var err error
			if p, err = withFieldValue(p, i, v.Value); err != nil {
				return pathed(err, "value")
			}
		default:
			panic(errors.Newf("write: unknown write binding variant %T", v))
		}
	}
	return checkSelected(fieldsOf(p), at, channels)
}

// checkChannel returns a validation error when f has a zero scale, or when the data
// type of b, from the channel that dataTypes holds or else from b, cannot write f.
func checkChannel(
	f field,
	b icd.ChannelWriteBinding,
	dataTypes map[channel.Key]telem.DataType,
) error {
	if err := icd.CheckScale(f.scale); err != nil {
		return errors.Wrapf(err, "field %s", f.name)
	}
	dt, found := dataTypes[b.Channel]
	if !found && b.DataType != nil {
		dt, found = *b.DataType, true
	}
	if !found {
		return nil
	}
	return errors.Wrapf(codec.CheckWrite(f.kind, f.scaled, dt), "field %s", f.name)
}

// checkSelected returns a validation error when a channel sets a field that a fixed
// multiplexor never selects, or when a field that can be selected has no channel and no
// default. channels holds the index of the binding of each field that a channel sets.
func checkSelected(
	fields []field,
	at map[icd.FieldKey]int,
	channels map[icd.FieldKey]int,
) error {
	for _, f := range fields {
		mux, never := neverSelected(f, fields, at, channels)
		j, channeled := channels[f.key]
		switch {
		case never && channeled:
			return validate.PathedError(validate.Errorf(
				"field %s takes a channel, but multiplexor %s never selects it",
				f.name, mux,
			), "bindings", strconv.Itoa(j))
		case !never && !channeled && !f.defaulted:
			return validate.Errorf("field %s has no binding and no default", f.name)
		}
	}
	return nil
}

// neverSelected returns the name of a multiplexor of f, or of a multiplexor above it,
// that no channel sets and whose fixed value never selects f.
func neverSelected(
	f field,
	fields []field,
	at map[icd.FieldKey]int,
	channels map[icd.FieldKey]int,
) (string, bool) {
	for cur := f; cur.multiplexor != nil; {
		mux := fields[at[*cur.multiplexor]]
		if _, channeled := channels[mux.key]; !channeled && mux.fixed != nil &&
			!slices.ContainsFunc(cur.multiplexValues, func(v int32) bool {
				return int64(v) == *mux.fixed
			}) {
			return mux.name, true
		}
		cur = mux
	}
	return "", false
}

// field is the part of a binary or text message field that a write checks.
type field struct {
	key       icd.FieldKey
	name      string
	kind      payload.Kind
	scale     float64
	scaled    bool
	defaulted bool
	// identifier is true when the field holds the identifier of its message.
	identifier bool
	// fixed is the raw value that an integer field sends when no channel sets it.
	fixed           *int64
	multiplexor     *icd.FieldKey
	multiplexValues []int32
}

// fieldIndexes returns the index of each of fields by key.
func fieldIndexes(fields []field) map[icd.FieldKey]int {
	at := make(map[icd.FieldKey]int, len(fields))
	for i, f := range fields {
		at[f.key] = i
	}
	return at
}

// fieldsOf returns the fields of p, a binary or text payload, in order. A text field
// encodes a float.
func fieldsOf(p icd.Payload) []field {
	switch v := p.Variant.(type) {
	case icd.BinaryPayload:
		var id *icd.FieldIdentifier
		if v.Identifier != nil {
			if fi, ok := v.Identifier.Variant.(icd.FieldIdentifier); ok {
				id = &fi
			}
		}
		return lo.Map(v.Fields, func(f icd.BinaryField, _ int) field {
			b := f.Base()
			out := field{
				key:             b.Key,
				name:            b.Name,
				kind:            payload.Kind(f.Kind()),
				scale:           b.Scale,
				scaled:          b.Scale != 1 || b.Offset != 0,
				multiplexor:     b.Multiplexor,
				multiplexValues: b.MultiplexValues,
			}
			switch d := f.Variant.(type) {
			case icd.FloatBinaryField:
				out.defaulted = d.Default != nil
			case icd.IntegerBinaryField:
				if d.Default != nil {
					out.fixed = new(int64(*d.Default))
				}
			case icd.UnsignedBinaryField:
				if d.Default != nil {
					out.fixed = new(int64(*d.Default))
				}
			}
			if id != nil && id.Field == b.Key {
				out.identifier, out.fixed = true, new(int64(id.Value))
			}
			out.defaulted = out.defaulted || out.fixed != nil
			return out
		})
	case icd.TextPayload:
		return lo.Map(v.Fields, func(f icd.TextField, _ int) field {
			b := f.Base()
			return field{
				key:       b.Key,
				name:      b.Name,
				kind:      payload.KindFloat,
				scale:     b.Scale,
				scaled:    b.Scale != 1 || b.Offset != 0,
				defaulted: b.Default != nil,
			}
		})
	default:
		panic(errors.Newf("write: payload variant %T is not binary or text", v))
	}
}

// withFieldValue returns p, a binary or text payload, with v as the default of field i.
// It returns a validation error when v is not of the kind of the field, or when the
// field cannot hold v.
func withFieldValue(p icd.Payload, i int, v icd.WriteValue) (icd.Payload, error) {
	var (
		name, kind string
		ok         bool
		err        error
	)
	switch pv := p.Variant.(type) {
	case icd.BinaryPayload:
		pv.Fields = slices.Clone(pv.Fields)
		f := &pv.Fields[i]
		name, kind = f.Base().Name, string(f.Kind())
		if *f, ok = binaryWithValue(*f, v); ok {
			err = icd.CheckBinaryDefault(*f)
		}
		p.Variant = pv
	case icd.TextPayload:
		pv.Fields = slices.Clone(pv.Fields)
		f := &pv.Fields[i]
		name, kind = f.Base().Name, string(payload.KindFloat)
		if *f, ok = textWithValue(*f, v); ok {
			err = icd.CheckTextDefault(*f)
		}
		p.Variant = pv
	default:
		panic(errors.Newf("write: payload variant %T is not binary or text", pv))
	}
	if !ok {
		return p, validate.Errorf(
			"field %s of kind %s needs a value of its kind",
			name,
			kind,
		)
	}
	return p, errors.Wrapf(err, "field %s", name)
}

// binaryWithValue returns f with v as its default, and false when v is not of the kind
// of f.
func binaryWithValue(f icd.BinaryField, v icd.WriteValue) (icd.BinaryField, bool) {
	switch x := v.Variant.(type) {
	case icd.FloatWriteValue:
		fv, ok := f.Variant.(icd.FloatBinaryField)
		if ok {
			fv.Default = &x.Value
			f.Variant = fv
		}
		return f, ok
	case icd.IntegerWriteValue:
		fv, ok := f.Variant.(icd.IntegerBinaryField)
		if ok {
			fv.Default = &x.Value
			f.Variant = fv
		}
		return f, ok
	case icd.UnsignedWriteValue:
		fv, ok := f.Variant.(icd.UnsignedBinaryField)
		if ok {
			fv.Default = &x.Value
			f.Variant = fv
		}
		return f, ok
	}
	return f, false
}

// textWithValue returns f with v as its default, and false when v is not a float.
func textWithValue(f icd.TextField, v icd.WriteValue) (icd.TextField, bool) {
	x, ok := v.Variant.(icd.FloatWriteValue)
	if !ok {
		return f, false
	}
	switch fv := f.Variant.(type) {
	case icd.DelimitedTextField:
		fv.Default = &x.Value
		f.Variant = fv
	case icd.TaggedTextField:
		fv.Default = &x.Value
		f.Variant = fv
	default:
		panic(errors.Newf("write: unknown text field variant %T", fv))
	}
	return f, true
}
