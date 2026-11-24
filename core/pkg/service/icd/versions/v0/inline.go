// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	xjson "github.com/synnaxlabs/x/json"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// InlineField is a JSON field that a read config held inline, before the config read
// the messages of an ICD.
type InlineField struct {
	// Key identifies the field within its config.
	Key string
	// Name is the name of the field.
	Name string
	// Disabled is true when the config reads no value from the field.
	Disabled bool
	// Channel is the channel that the value of the field is written to.
	Channel channel.Key
	// Pointer is the JSON Pointer to the value of the field.
	Pointer string
	// DataType is the data type of the value of the field.
	DataType telem.DataType
	// TimeFormat is the encoding of a timestamp value.
	TimeFormat *payload.TimeFormat
	// EnumValues maps text labels to values.
	EnumValues []payload.EnumEntry
}

// Channel is the part of a channel that an inline field must agree with.
type Channel struct {
	// Name is the name of the channel.
	Name string
	// DataType is the data type of the channel.
	DataType telem.DataType
	// Index is true when the channel is an index channel.
	Index bool
}

// Channels returns the channels among keys, read through snap. A key with no channel is
// absent from the result.
type Channels func(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]Channel, error)

// Of returns the channels of fields, read through snap.
func (c Channels) Of(
	ctx context.Context,
	snap gorp.Snapshot,
	fields []InlineField,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, f := range fields {
		if f.Channel != 0 {
			keys = append(keys, f.Channel)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return c(ctx, snap, lo.Uniq(keys))
}

// LiftMessage returns a JSON message that holds fields, with no name, and the read of
// that message. A field takes the kind of its channel in found, and binds no channel
// when found lacks it. A disabled field gets no binding. The field with key index
// becomes the index of the read. note receives path.fields.<i> and each change to the
// field at i, and path.index when no field has key index. A field whose pointer leads
// to the value of an earlier field, or into or around it, gets a note that names both.
func LiftMessage(
	fields []InlineField,
	index string,
	found map[channel.Key]Channel,
	path string,
	note func(path, change string),
) (Message, ReadMessage) {
	var (
		lifted = make([]JSONField, len(fields))
		read   = ReadMessage{Message: uuid.New(), Bindings: []ReadBinding{}}
		names  = make(set.Set[string])
	)
	for i, old := range fields {
		var ch *Channel
		if c, exists := found[old.Channel]; exists {
			ch = &c
		}
		isIndex := index != "" && old.Key == index
		key := uuid.New()
		field, dataType, changes := liftField(old, isIndex, ch, JSONBaseField{
			Key:     key,
			Name:    fieldName(names, old.Name, old.Pointer),
			Scale:   1,
			Pointer: old.Pointer,
		})
		at := path + ".fields." + strconv.Itoa(i)
		for _, c := range changes {
			note(at, c)
		}
		for _, prior := range lifted[:i] {
			if reads := overlap(prior.Base().Pointer, old.Pointer); reads != "" {
				note(at, fmt.Sprintf(
					"field %q reads %s field %q, so keep one of them before you edit "+
						"the ICD",
					field.Base().Name,
					reads,
					prior.Base().Name,
				))
				break
			}
		}
		lifted[i] = field
		if isIndex {
			read.Index = &key
		}
		if old.Disabled {
			continue
		}
		b := ReadBinding{Field: key}
		if ch != nil {
			b.Channel = old.Channel
		} else if dataType != "" {
			b.DataType = dataType
		}
		read.Bindings = append(read.Bindings, b)
	}
	if index != "" && read.Index == nil {
		note(path+".index", "index field "+index+" does not exist and was dropped")
	}
	return Message{
		Key:     read.Message,
		Payload: Payload{Variant: JSONPayload{Fields: lifted}},
	}, read
}

// overlap returns how the value at pointer b relates to the value at pointer a: "the
// same value as", "a value inside", or "a value that holds". It returns "" when neither
// value holds the other, or when either pointer is malformed.
func overlap(a, b string) string {
	pa, errA := xjson.ParsePointer(a)
	pb, errB := xjson.ParsePointer(b)
	switch {
	case errA != nil || errB != nil:
		return ""
	case slices.Equal(pa, pb):
		return "the same value as"
	case pb.HasPrefix(pa):
		return "a value inside"
	case pa.HasPrefix(pb):
		return "a value that holds"
	}
	return ""
}

// UniqueName returns name, or name with the lowest suffix " (n)" that taken lacks, and
// adds the result to taken.
func UniqueName(taken set.Set[string], name string) string {
	unique := name
	for n := 2; taken.Contains(unique); n++ {
		unique = name + " (" + strconv.Itoa(n) + ")"
	}
	taken.Add(unique)
	return unique
}

// fieldName returns name, or else a name made from pointer, apart from the names in
// taken, and adds it to taken. The tokens of pointer join with "_", and the whole
// document gives "value". A taken name gets the lowest free suffix "_n", from 2. Two
// names clash when they are the same as channel names. taken holds channel names.
func fieldName(taken set.Set[string], name, pointer string) string {
	base := lo.CoalesceOrEmpty(name, pointerName(pointer))
	unique := base
	for n := 2; taken.Contains(channelName(unique)); n++ {
		unique = base + "_" + strconv.Itoa(n)
	}
	taken.Add(channelName(unique))
	return unique
}

// pointerName joins the tokens of pointer with "_", or returns "value" for the whole
// document. A malformed pointer gives its text.
func pointerName(pointer string) string {
	tokens, err := xjson.ParsePointer(pointer)
	if err != nil {
		return strings.TrimPrefix(pointer, "/")
	}
	return lo.CoalesceOrEmpty(strings.Join(tokens, "_"), "value")
}

// channelName returns name as a valid channel name: each character other than a letter,
// a digit, or "_" becomes "_", and a leading digit gets a "_" before it.
func channelName(name string) string {
	escaped := strings.Map(func(r rune) rune {
		if r == '_' || 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' ||
			'0' <= r && r <= '9' {
			return r
		}
		return '_'
	}, name)
	if escaped != "" && '0' <= escaped[0] && escaped[0] <= '9' {
		return "_" + escaped
	}
	return escaped
}

// liftField returns the JSON field of base for old, the data type of a new channel for
// it, and a description of each change. The data type of ch, when old has a channel,
// picks the kind. Otherwise the stored data type picks it, and a numeric kind keeps it
// as the data type of a new channel. The returned data type is empty for any other
// field. An index field lifts to the timestamp kind, and so does a field with a time
// format, no entries, and the default or an unknown data type. An entry that the kind
// cannot hold is dropped, and a time format that is not valid becomes ISO 8601.
func liftField(
	old InlineField,
	index bool,
	ch *Channel,
	base JSONBaseField,
) (JSONField, telem.DataType, []string) {
	var changes []string
	note := func(format string, args ...any) {
		changes = append(changes, fmt.Sprintf(format, args...))
	}
	dt := old.DataType
	if dt == "" {
		dt = telem.Float64T
	}
	if ch != nil && ch.DataType != dt {
		note("data type %s became %s, the data type of its channel", dt, ch.DataType)
		dt = ch.DataType
	}
	kind, known := xpayload.KindOf(dt)
	if !known {
		kind = payload.KindFloat
		note("unknown data type %q became the float kind", dt)
	}
	switch {
	case ch == nil && kind != payload.KindTimestamp &&
		(index || (old.TimeFormat != nil && len(old.EnumValues) == 0 &&
			(dt == telem.Float64T || !known))):
		kind = payload.KindTimestamp
		note("data type %s became the timestamp kind of an index or time format", dt)
	case old.TimeFormat != nil && kind != payload.KindTimestamp:
		note(
			"time format %s was dropped, because the field is %s",
			*old.TimeFormat,
			kind,
		)
	}
	if kind == payload.KindTimestamp && old.TimeFormat != nil &&
		!old.TimeFormat.IsValid() {
		note(
			"time format %q is not valid and became %s",
			*old.TimeFormat,
			payload.TimeFormatISO8601,
		)
		old.TimeFormat = nil
	}
	field := liftRead(old, kind, base, enumFilter{seen: set.New[string](), note: note})
	numeric := kind == payload.KindFloat || kind == payload.KindInteger ||
		kind == payload.KindUnsigned
	if !known || ch != nil || !numeric {
		dt = ""
	}
	return field, dt, changes
}

// enumFilter keeps the enum entries that a lifted field can use and notes each one it
// drops.
type enumFilter struct {
	seen set.Set[string]
	note func(format string, args ...any)
}

func (f enumFilter) drop(e payload.EnumEntry, why string) {
	f.note("enum label %q was dropped: %s", e.Label, why)
}

// usable reports whether the lifted field can keep e: its label is not empty, and no
// earlier entry has it.
func (f enumFilter) usable(e payload.EnumEntry) bool {
	if e.Label == "" {
		f.drop(e, "it is empty")
		return false
	}
	if f.seen.Contains(e.Label) {
		f.drop(e, "it repeats an earlier label")
		return false
	}
	f.seen.Add(e.Label)
	return true
}

// LiftKind returns the JSON field of base with kind, and the enum values and time
// format of old that kind can hold. note receives each entry that it drops.
func LiftKind(
	old InlineField,
	kind payload.Kind,
	base JSONBaseField,
	note func(format string, args ...any),
) JSONField {
	return liftRead(old, kind, base, enumFilter{seen: set.New[string](), note: note})
}

// liftRead returns the JSON field of base with kind and the options of old.
func liftRead(
	old InlineField,
	kind payload.Kind,
	base JSONBaseField,
	f enumFilter,
) JSONField {
	if field, ok := liftEnumRead(old, kind, base, f); ok {
		return field
	}
	for _, e := range old.EnumValues {
		f.drop(e, "a "+string(kind)+" field takes no numeric entries")
	}
	switch kind {
	case payload.KindString:
		return JSONField{Variant: StringJSONField{JSONBaseField: base}}
	case payload.KindTimestamp:
		format := payload.TimeFormatISO8601
		if old.TimeFormat != nil {
			format = *old.TimeFormat
		}
		return JSONField{Variant: TimestampJSONField{
			JSONBaseField: base,
			TimeFormat:    format,
		}}
	case payload.KindJSON:
		return JSONField{Variant: JSONJSONField{JSONBaseField: base}}
	case payload.KindUUID:
		return JSONField{Variant: UuidJSONField{JSONBaseField: base}}
	case payload.KindBytes:
		return JSONField{Variant: BytesJSONField{JSONBaseField: base}}
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned,
		payload.KindBoolean:
	}
	panic("unknown read field kind " + string(kind))
}

// liftEnumRead returns the JSON field of base for a kind that takes enum entries, with
// the entries of old that f keeps, and false for any other kind.
func liftEnumRead(
	old InlineField,
	kind payload.Kind,
	base JSONBaseField,
	f enumFilter,
) (JSONField, bool) {
	switch kind {
	case payload.KindFloat:
		return JSONField{Variant: FloatJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, floatEntry),
		}}, true
	case payload.KindInteger:
		return JSONField{Variant: IntegerJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, integerEntry),
		}}, true
	case payload.KindUnsigned:
		return JSONField{Variant: UnsignedJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, unsignedEntry),
		}}, true
	case payload.KindBoolean:
		return JSONField{Variant: BooleanJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, booleanEntry),
		}}, true
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	return JSONField{}, false
}

// liftEntries returns the entries of old that f keeps, through entry. It drops each one
// whose value entry reports that kind does not hold.
func liftEntries[E any](
	old []payload.EnumEntry,
	f enumFilter,
	kind payload.Kind,
	entry func(payload.EnumEntry) (E, bool),
) []E {
	entries := make([]E, 0, len(old))
	for _, e := range old {
		lifted, ok := entry(e)
		switch {
		case !ok:
			f.drop(
				e,
				strconv.FormatFloat(e.Value, 'g', -1, 64)+" is not "+string(kind),
			)
		case f.usable(e):
			entries = append(entries, lifted)
		}
	}
	return entries
}

func floatEntry(e payload.EnumEntry) (payload.FloatEntry, bool) {
	return payload.FloatEntry{Label: e.Label, Value: payload.Float(e.Value)}, true
}

// integerEntry returns false when the value of e is not a whole number that int64
// holds.
func integerEntry(e payload.EnumEntry) (payload.IntegerEntry, bool) {
	if v := e.Value; v != math.Trunc(v) || v < math.MinInt64 || v >= 1<<63 {
		return payload.IntegerEntry{}, false
	}
	return payload.IntegerEntry{Label: e.Label, Value: payload.Integer(e.Value)}, true
}

// unsignedEntry returns false when the value of e is not a whole number that uint64
// holds.
func unsignedEntry(e payload.EnumEntry) (payload.UnsignedEntry, bool) {
	if v := e.Value; v != math.Trunc(v) || v < 0 || v >= 1<<64 {
		return payload.UnsignedEntry{}, false
	}
	return payload.UnsignedEntry{Label: e.Label, Value: payload.Unsigned(e.Value)}, true
}

func booleanEntry(e payload.EnumEntry) (payload.BooleanEntry, bool) {
	return payload.BooleanEntry{Label: e.Label, Value: e.Value != 0}, true
}
