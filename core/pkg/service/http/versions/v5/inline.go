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
// through snap with channels, and binds no channel when that channel does not exist. A
// disabled field gets no binding, and a fill value becomes the default of its field.
// note receives the path and each change of a field. ResolveReadConfig returns an error
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
	names := make(set.Set[string])
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
		msg, read := icdv0.LiftMessage(
			lo.Map(e.Fields, toInline), e.Index, found, paths[i], note,
		)
		msg.Name = icdv0.UniqueName(
			names,
			strings.TrimSpace(string(ep.Method)+" "+ep.Path),
		)
		lifted.Messages[i] = msg
		ep.ICD = new(lifted.Key)
		ep.Messages = []icdv0.ReadMessage{read}
	}
	ic, err := icdv1.MigrateICD(ctx, lifted)
	if err != nil {
		return icdv1.ICD{}, false, err
	}
	for i, e := range staged {
		fields := ic.Messages[i].Payload.Variant.(icdv1.JSONPayload).Fields
		for j, f := range e.Fields {
			if f.FillValue == nil {
				continue
			}
			if err = liftFill(&fields[j], f, *f.FillValue); err != nil {
				note(
					paths[i]+".fields."+strconv.Itoa(j),
					"fill value was dropped: "+err.Error(),
				)
			}
		}
	}
	cfg.Inline = nil
	return ic, true, nil
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

// liftFill sets the default of field to fill, read as a response value of old: through
// the enum labels of old and the time format of field. It returns an error when fill
// does not convert to the kind of field.
func liftFill(field *icdv1.JSONField, old ReadField, fill any) error {
	var (
		f   xjson.Field
		set func(sample []byte) icdv1.JSONFieldVariant
	)
	enums := lo.Map(old.EnumValues, func(e payload.EnumEntry, _ int) xjson.Enum {
		return xjson.NewEnum(e.Label, e.Value)
	})
	switch v := field.Variant.(type) {
	case icdv1.FloatJSONField:
		f = xjson.Field{DataType: telem.Float64T, Enums: enums}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(payload.Float(math.Float64frombits(fixed(s))))
			return v
		}
	case icdv1.IntegerJSONField:
		f = xjson.Field{DataType: telem.Int64T, Enums: enums}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(payload.Integer(int64(fixed(s))))
			return v
		}
	case icdv1.UnsignedJSONField:
		f = xjson.Field{DataType: telem.Uint64T, Enums: enums}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(payload.Unsigned(fixed(s)))
			return v
		}
	case icdv1.BooleanJSONField:
		f = xjson.Field{DataType: telem.BooleanT, Enums: enums}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(s[0] != 0)
			return v
		}
	case icdv1.StringJSONField:
		f = xjson.Field{DataType: telem.StringT}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(string(variable(s)))
			return v
		}
	case icdv1.TimestampJSONField:
		format, err := xjson.ParseTimeFormat(string(v.TimeFormat))
		if err != nil {
			return err
		}
		f = xjson.Field{DataType: telem.TimestampT, TimeFormat: format}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = &icdv1.TimestampDefault{
				Variant: icdv1.ConstantTimestampDefault{
					Value: telem.TimeStamp(fixed(s)),
				},
			}
			return v
		}
	case icdv1.JSONJSONField:
		f = xjson.Field{DataType: telem.JSONT}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(string(variable(s)))
			return v
		}
	case icdv1.UuidJSONField:
		f = xjson.Field{DataType: telem.UUIDT}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = &icdv1.UUIDDefault{
				Variant: icdv1.ConstantUUIDDefault{Value: uuid.UUID(s)},
			}
			return v
		}
	case icdv1.BytesJSONField:
		f = xjson.Field{DataType: telem.BytesT}
		set = func(s []byte) icdv1.JSONFieldVariant {
			v.Default = new(base64.StdEncoding.EncodeToString(variable(s)))
			return v
		}
	default:
		panic("no default for JSON field " + field.Base().Name)
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
	field.Variant = set(sample)
	return nil
}

// fixed returns the 8-byte sample s as a uint64.
func fixed(s []byte) uint64 { return telem.ByteOrder.Uint64(s) }

// variable returns the variable-density sample s without its length.
func variable(s []byte) []byte {
	b, _ := telem.UnmarshalVariableSample(s)
	return b
}
