// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"fmt"
	"slices"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// enumMap converts enum entries into the map that maps record labels to samples.
func enumMap(entries []http.EnumEntry) xjson.EnumMap {
	if len(entries) == 0 {
		return nil
	}
	m := make(xjson.EnumMap, len(entries))
	for _, e := range entries {
		m[e.Label] = e.Value
	}
	return m
}

// reverseEnumMap converts enum entries into the map that maps samples to record
// labels.
func reverseEnumMap(entries []http.EnumEntry) xjson.ReverseEnumMap {
	if len(entries) == 0 {
		return nil
	}
	m := make(xjson.ReverseEnumMap, len(entries))
	for _, e := range entries {
		m[e.Value] = e.Label
	}
	return m
}

// recordLayout is where a write task places each part of a record.
type recordLayout struct {
	value xjson.Pointer
	// channel and timestamp are empty when the record carries no channel name or no
	// timestamp.
	channel   xjson.Pointer
	timestamp xjson.Pointer
	format    xjson.TimeFormat
	fields    []extraField
}

// extraField is a static or generated field of every record.
type extraField struct {
	pointer xjson.Pointer
	// value is the value of a static field.
	value any
	// generator is empty for a static field.
	generator http.GeneratorType
	format    xjson.TimeFormat
}

// labeledPointer is a parsed pointer named after the config entry that set it.
type labeledPointer struct {
	label, text string
	pointer     xjson.Pointer
}

// disjointPointers holds the pointers of one record, which must not overlap.
type disjointPointers []labeledPointer

// add parses text, the pointer of the config entry label. It returns an error wrapping
// validate.ErrValidation when text is malformed or empty, or when it equals or nests
// with a pointer added before, since setting both would overwrite one.
func (ps *disjointPointers) add(label, text string) (xjson.Pointer, error) {
	p, err := xjson.ParsePointer(text)
	if err != nil {
		return nil, errors.Wrap(err, label)
	}
	if len(p) == 0 {
		return nil, errors.Wrapf(validate.ErrValidation, "%s: must not be empty", label)
	}
	for _, q := range *ps {
		if hasPrefix(p, q.pointer) || hasPrefix(q.pointer, p) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"%s %q and %s %q overlap",
				q.label, q.text, label, text,
			)
		}
	}
	*ps = append(*ps, labeledPointer{label: label, text: text, pointer: p})
	return p, nil
}

// parseRecordLayout parses the pointers, time formats, and generators of rec. It
// returns an error wrapping validate.ErrValidation when a pointer fails
// disjointPointers.add, or when a time format or generator is unknown.
func parseRecordLayout(rec Record) (l recordLayout, err error) {
	var pointers disjointPointers
	if l.value, err = pointers.add(
		"record.value_pointer",
		rec.ValuePointer,
	); err != nil {
		return recordLayout{}, err
	}
	if p := rec.ChannelPointer; p != nil {
		if l.channel, err = pointers.add("record.channel_pointer", *p); err != nil {
			return recordLayout{}, err
		}
	}
	if p := rec.TimestampPointer; p != nil {
		if l.timestamp, err = pointers.add("record.timestamp_pointer", *p); err != nil {
			return recordLayout{}, err
		}
		if l.format, err = xjson.ParseTimeFormat(string(rec.TimeFormat)); err != nil {
			return recordLayout{}, errors.Wrap(err, "record.time_format")
		}
	}
	for _, f := range rec.Fields {
		var (
			field extraField
			base  http.BaseWriteField
		)
		switch v := f.Variant.(type) {
		case http.StaticWriteField:
			base, field.value = v.BaseWriteField, v.Value
		case http.GeneratedWriteField:
			base = v.BaseWriteField
			if field, err = parseGenerator(v); err != nil {
				return recordLayout{}, errors.Wrapf(err, "field %q", base.Key)
			}
		default:
			return recordLayout{}, errors.Wrapf(
				validate.ErrValidation, "unknown write field variant %T", v,
			)
		}
		label := fmt.Sprintf("field %q", base.Key)
		if field.pointer, err = pointers.add(label, base.Pointer); err != nil {
			return recordLayout{}, err
		}
		l.fields = append(l.fields, field)
	}
	return l, nil
}

// parseGenerator parses a generated field, leaving its pointer unset.
func parseGenerator(v http.GeneratedWriteField) (extraField, error) {
	field := extraField{generator: v.Generator, format: xjson.UnixNanosecond}
	switch v.Generator {
	case http.GeneratorTypeUUID:
	case http.GeneratorTypeTimestamp:
		if v.TimeFormat != nil {
			var err error
			if field.format, err = xjson.ParseTimeFormat(
				string(*v.TimeFormat),
			); err != nil {
				return extraField{}, err
			}
		}
	default:
		return extraField{}, errors.Wrapf(
			validate.ErrValidation, "unknown generator %q", v.Generator,
		)
	}
	return field, nil
}

// generate returns the value of f for a record stamped ts.
func (f extraField) generate(ts telem.TimeStamp) any {
	switch f.generator {
	case http.GeneratorTypeUUID:
		return uuid.New().String()
	case http.GeneratorTypeTimestamp:
		return xjson.FromTimeStamp(ts, f.format)
	}
	return f.value
}

func hasPrefix(p, prefix xjson.Pointer) bool {
	return len(p) >= len(prefix) && slices.Equal(p[:len(prefix)], prefix)
}
