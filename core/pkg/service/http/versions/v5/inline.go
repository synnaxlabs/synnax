// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"math"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// ResolveReadConfig moves the staged fields of cfg into a new ICD named name, with one
// JSON message for each staged endpoint, named after its method and path, and reads
// each endpoint from its message. It returns the ICD, which the caller must create, and
// false when cfg holds no inline fields. A field takes the kind of its channel, read
// through snap with channels. A field binds no channel when it is disabled, when its
// channel does not exist, or when an earlier field writes that channel. The index of a
// read is the field that writes an index channel, and it takes no fill value. Any other
// fill value becomes the default of its field, and an optional float field with no fill
// value takes the default NaN. An endpoint whose fields are all disabled becomes
// disabled. note receives the path and each change. ResolveReadConfig returns an error
// when a staged endpoint is not an endpoint of cfg.
func ResolveReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels icdv0.Channels,
	name string,
	cfg *ReadConfig,
	note func(path, change string),
) (icdv1.ICD, bool, error) {
	if cfg.Inline == nil {
		return icdv1.ICD{}, false, nil
	}
	staged := cfg.Inline.Endpoints
	found, err := channels.Of(
		ctx,
		snap,
		lo.FlatMap(staged, func(e InlineEndpoint, _ int) []icdv0.InlineField {
			return lo.Map(e.Fields, toInline)
		}),
	)
	if err != nil {
		return icdv1.ICD{}, false, err
	}
	lifted := icdv0.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icdv0.Message, len(staged)),
	}
	paths := make([]string, len(staged))
	indexes := make([]string, len(staged))
	names := make(set.Set[string])
	claimed := make(map[channel.Key]string)
	for i, e := range staged {
		at := slices.IndexFunc(cfg.Endpoints, func(ep ReadEndpoint) bool {
			return ep.Key == e.Key
		})
		if at < 0 {
			return icdv1.ICD{}, false, errors.Newf(
				"staged endpoint %s is not an endpoint of the config",
				e.Key,
			)
		}
		ep := &cfg.Endpoints[at]
		paths[i] = "endpoints." + strconv.Itoa(at)
		fields, index := readFields(e, found, claimed, paths[i], note)
		indexes[i] = index
		msg, read := icdv0.LiftMessage(fields, index, found, paths[i], note)
		msg.Name = icdv0.UniqueName(
			names,
			strings.TrimSpace(string(ep.Method)+" "+ep.Path),
		)
		lifted.Messages[i] = msg
		ep.ICD = new(lifted.Key)
		ep.Messages = []icdv0.ReadMessage{read}
		if !ep.Disabled && len(e.Fields) > 0 &&
			lo.EveryBy(e.Fields, func(f ReadField) bool { return f.Disabled }) {
			ep.Disabled = true
			note(paths[i], "was disabled, because each of its fields is disabled")
		}
	}
	ic, err := icdv1.MigrateICD(ctx, lifted)
	if err != nil {
		return icdv1.ICD{}, false, err
	}
	liftFills(ic, staged, paths, indexes, note)
	cfg.Inline = nil
	return ic, true, nil
}

// readFields returns the fields of e to lift and the key of its index field: the field
// that writes an index channel in found, preferring the field that e names. A field
// binds no channel when claimed holds an earlier field that writes it. readFields adds
// each field it binds to claimed, and note receives each change at path and each other
// field that writes an index channel.
func readFields(
	e InlineEndpoint,
	found map[channel.Key]icdv0.Channel,
	claimed map[channel.Key]string,
	path string,
	note func(path, change string),
) ([]icdv0.InlineField, string) {
	fields := lo.Map(e.Fields, toInline)
	index := ""
	var indexFields []int
	for j := range fields {
		f := &fields[j]
		ch, exists := found[f.Channel]
		if f.Disabled || !exists {
			continue
		}
		at := path + ".fields." + strconv.Itoa(j)
		if other, ok := claimed[f.Channel]; ok {
			f.Disabled = true
			note(at, "binds no channel, because "+other+" also writes channel "+
				strconv.FormatUint(uint64(f.Channel), 10))
			continue
		}
		claimed[f.Channel] = at
		if !ch.Index {
			continue
		}
		indexFields = append(indexFields, j)
		if index == "" || f.Key == e.Index {
			index = f.Key
		}
	}
	for _, j := range indexFields {
		if key := fields[j].Key; key != index {
			note(path+".fields."+strconv.Itoa(j), "field "+key+" and index field "+
				index+" both write index channels. Split the endpoint into two, each "+
				"with one index field")
		}
	}
	switch index {
	case e.Index:
	case "":
		note(path+".index", "index field "+e.Index+
			" was dropped, because it writes no index channel")
	default:
		note(path+".index", "field "+index+
			" became the index, because it writes an index channel")
	}
	return fields, index
}

// liftFills turns the fill value of each field of staged into the default of its field
// in ic, whose messages hold the endpoints of staged in order, at paths. It drops the
// fill value of the field that indexes names for each endpoint, and lifts each optional
// field with no fill value through liftOptional. note receives the path of each fill
// value that it drops.
func liftFills(
	ic icdv1.ICD,
	staged []InlineEndpoint,
	paths []string,
	indexes []string,
	note func(path, change string),
) {
	for i, e := range staged {
		p, ok := ic.Messages[i].Payload.Variant.(icdv1.JSONPayload)
		if !ok {
			panic("lifted message " + ic.Messages[i].Name + " is not a JSON message")
		}
		for j, f := range e.Fields {
			at := paths[i] + ".fields." + strconv.Itoa(j)
			switch {
			case f.FillValue == nil:
				if f.Optional {
					liftOptional(&p.Fields[j], at, note)
				}
			case indexes[i] != "" && f.Key == indexes[i]:
				note(at, "fill value was dropped, because the index takes no default")
			default:
				if err := liftFill(&p.Fields[j], *f.FillValue); err != nil {
					note(at, "fill value was dropped: "+err.Error())
				}
			}
		}
	}
}

// liftOptional gives field, an optional field with no fill value, the default NaN, as a
// read before ICDs wrote NaN for its missing value. note receives path when field is
// not a float field, as only a float holds NaN.
func liftOptional(field *icdv1.JSONField, path string, note func(path, change string)) {
	v, ok := field.Variant.(icdv1.FloatJSONField)
	if !ok {
		note(path, "has no default, because it is optional with no fill value and is "+
			"not a float. A read skips each document that has no value for it")
		return
	}
	v.Default = new(payload.Float(math.NaN()))
	field.Variant = v
}

func toInline(f ReadField, _ int) icdv0.InlineField {
	return icdv0.InlineField{
		Key:        f.Key,
		Name:       f.Name,
		Disabled:   f.Disabled,
		Channel:    f.Channel,
		Pointer:    f.Pointer,
		DataType:   f.DataType,
		TimeFormat: f.TimeFormat,
		EnumValues: f.EnumValues,
	}
}

// liftFill sets the default of field to fill, read as a response value through the enum
// labels and the time format of field. It returns an error when fill does not convert
// to the kind of field.
func liftFill(field *icdv1.JSONField, fill any) error {
	f, err := fillField(*field)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(fill)
	if err != nil {
		return err
	}
	d, err := xjson.NewDecoder(f)
	if err != nil {
		return err
	}
	sample, err := d.Append(nil, raw)
	if err != nil {
		return err
	}
	field.Variant = withDefault(*field, sample)
	return nil
}

// fillField returns the decoder field that reads a fill value into the kind of field.
func fillField(field icdv1.JSONField) (xjson.Field, error) {
	switch v := field.Variant.(type) {
	case icdv1.FloatJSONField:
		return xjson.Field{
			DataType: telem.Float64T,
			Enums: lo.Map(v.EnumValues, func(e payload.FloatEntry, _ int) xjson.Enum {
				return xjson.NewEnum(e.Label, float64(e.Value))
			}),
		}, nil
	case icdv1.IntegerJSONField:
		return xjson.Field{
			DataType: telem.Int64T,
			Enums: lo.Map(v.EnumValues, func(e payload.IntegerEntry, _ int) xjson.Enum {
				return xjson.NewEnum(e.Label, int64(e.Value))
			}),
		}, nil
	case icdv1.UnsignedJSONField:
		return xjson.Field{
			DataType: telem.Uint64T,
			Enums: lo.Map(
				v.EnumValues,
				func(e payload.UnsignedEntry, _ int) xjson.Enum {
					return xjson.NewEnum(e.Label, uint64(e.Value))
				},
			),
		}, nil
	case icdv1.BooleanJSONField:
		return xjson.Field{
			DataType: telem.BooleanT,
			Enums: lo.Map(v.EnumValues, func(e payload.BooleanEntry, _ int) xjson.Enum {
				return xjson.NewEnum(e.Label, boolSample(e.Value))
			}),
		}, nil
	case icdv1.StringJSONField:
		return xjson.Field{DataType: telem.StringT}, nil
	case icdv1.TimestampJSONField:
		format, err := xjson.ParseTimeFormat(string(v.TimeFormat))
		return xjson.Field{DataType: telem.TimestampT, TimeFormat: format}, err
	case icdv1.JSONJSONField:
		return xjson.Field{DataType: telem.JSONT}, nil
	case icdv1.UuidJSONField:
		return xjson.Field{DataType: telem.UUIDT}, nil
	case icdv1.BytesJSONField:
		return xjson.Field{DataType: telem.BytesT}, nil
	default:
		panic("no default for JSON field " + field.Base().Name)
	}
}

// withDefault returns the variant of field with sample, as fillField reads it, for its
// default.
func withDefault(field icdv1.JSONField, sample []byte) icdv1.JSONFieldVariant {
	switch v := field.Variant.(type) {
	case icdv1.FloatJSONField:
		v.Default = new(payload.Float(math.Float64frombits(fixed(sample))))
		return v
	case icdv1.IntegerJSONField:
		v.Default = new(payload.Integer(int64(fixed(sample))))
		return v
	case icdv1.UnsignedJSONField:
		v.Default = new(payload.Unsigned(fixed(sample)))
		return v
	case icdv1.BooleanJSONField:
		v.Default = new(sample[0] != 0)
		return v
	case icdv1.StringJSONField:
		v.Default = new(string(variable(sample)))
		return v
	case icdv1.TimestampJSONField:
		v.Default = &icdv1.TimestampDefault{
			Variant: icdv1.ConstantTimestampDefault{
				Value: telem.TimeStamp(fixed(sample)),
			},
		}
		return v
	case icdv1.JSONJSONField:
		v.Default = new(string(variable(sample)))
		return v
	case icdv1.UuidJSONField:
		v.Default = &icdv1.UUIDDefault{
			Variant: icdv1.ConstantUUIDDefault{Value: uuid.UUID(sample)},
		}
		return v
	case icdv1.BytesJSONField:
		v.Default = new(base64.StdEncoding.EncodeToString(variable(sample)))
		return v
	default:
		panic("no default for JSON field " + field.Base().Name)
	}
}

// boolSample returns v as a numeric sample, 1 for true.
func boolSample(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

// fixed returns the 8-byte sample s as a uint64.
func fixed(s []byte) uint64 { return telem.ByteOrder.Uint64(s) }

// variable returns the variable-density sample s without its length.
func variable(s []byte) []byte {
	b, _ := telem.UnmarshalVariableSample(s)
	return b
}
