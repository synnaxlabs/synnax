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
	"context"
	"fmt"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

func fieldPath(rec, fld int, rest ...string) []string {
	path := []string{"records", strconv.Itoa(rec), "fields", strconv.Itoa(fld)}
	return append(path, rest...)
}

// validateReadConfig checks the rules of c that need no channel: each field's labels
// are unique, and the data type of a new channel matches the field's kind and holds
// each entry and the fill.
func validateReadConfig(c *ReadConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	for i, rec := range c.Records {
		for j, fld := range rec.Fields {
			if fld.Variant == nil {
				return validate.PathedError(errors.Wrap(
					validate.ErrValidation, "type: must be set",
				), fieldPath(i, j)...)
			}
			opts := optionsOf(fld)
			labels := make(set.Set[string], len(opts.entries))
			for _, e := range opts.entries {
				if labels.Contains(e.label) {
					return validate.PathedError(errors.Wrapf(
						validate.ErrValidation, "label %q repeats", e.label,
					), fieldPath(i, j, "enum_values")...)
				}
				labels.Add(e.label)
			}
			if dt := dataTypeOf(fld); dt != nil {
				if kind, _ := payload.KindOf(*dt); kind != fld.Type() {
					return validate.PathedError(errors.Wrapf(
						validate.ErrValidation,
						"%s is not a data type of kind %s", *dt, fld.Type(),
					), fieldPath(i, j, "data_type")...)
				}
				for k, e := range opts.entries {
					if _, err := convert(e.value, *dt); err != nil {
						return validate.PathedError(
							err,
							fieldPath(i, j, "enum_values", strconv.Itoa(k), "value")...,
						)
					}
				}
				if opts.fill != nil {
					if _, err := convert(opts.fill, *dt); err != nil {
						return validate.PathedError(err, fieldPath(i, j, "fill")...)
					}
				}
			}
		}
	}
	return nil
}

// channelsOf returns the channels that the fields of c write to, by key. A key with no
// channel has no entry, since an import can hold keys of another Core.
func channelsOf(
	ctx context.Context,
	snap gorp.Snapshot,
	svc *channel.Service,
	c *ReadConfig,
) (map[channel.Key]channel.Channel, error) {
	var keys channel.Keys
	for _, rec := range c.Records {
		for _, fld := range rec.Fields {
			if k := fld.Base().Channel; k != 0 {
				keys = append(keys, k)
			}
		}
	}
	var channels []channel.Channel
	if len(keys) > 0 {
		if err := svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&channels).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
	}
	byKey := make(map[channel.Key]channel.Channel, len(channels))
	for _, ch := range channels {
		byKey[ch.Key()] = ch
	}
	return byKey, nil
}

// resolveReadConfig checks each enabled field of c against its channel: the kind
// matches, each entry and the fill fit, and an index channel takes no fill. It clears
// the data type of each field whose channel exists.
func resolveReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	svc *channel.Service,
	c *ReadConfig,
) error {
	channels, err := channelsOf(ctx, snap, svc, c)
	if err != nil {
		return err
	}
	for i, rec := range c.Records {
		for j := range rec.Fields {
			fld := &rec.Fields[j]
			ch, ok := channels[fld.Base().Channel]
			if !ok {
				continue
			}
			setDataType(fld, nil)
			if fld.Base().Disabled {
				continue
			}
			if _, err := newColumn(*fld, ch); err != nil {
				return validate.PathedError(err, fieldPath(i, j)...)
			}
		}
	}
	return nil
}

// reconcileReadConfig changes each field of c to match its channel, and keeps the first
// entry of each repeated label. It logs each change through ins.
func reconcileReadConfig(
	ctx context.Context,
	tx gorp.Tx,
	svc *channel.Service,
	ins alamos.Instrumentation,
	c *ReadConfig,
) error {
	channels, err := channelsOf(ctx, tx, svc, c)
	if err != nil {
		return err
	}
	for _, rec := range c.Records {
		for j := range rec.Fields {
			fld := &rec.Fields[j]
			changes := dedupeLabels(fld)
			if ch, ok := channels[fld.Base().Channel]; ok {
				changes = append(changes, reconcileField(fld, ch)...)
			}
			if len(changes) > 0 {
				ins.L.Warn(
					"changed a Kafka read field to match its channel",
					zap.Stringer("config", c.Key),
					zap.String("field", fld.Base().Key),
					zap.Uint32("channel", uint32(fld.Base().Channel)),
					zap.Strings("changes", changes),
				)
			}
		}
	}
	return nil
}

// reconcileField changes fld to the kind of ch and clears its data type. Between
// numeric kinds, it keeps each entry and the fill that fits ch, and an integer must fit
// a float channel exactly. On other kinds, it drops them. It drops a fill on an index
// channel. It returns the changes it made.
func reconcileField(fld *ReadField, ch channel.Channel) []string {
	var changes []string
	if dataTypeOf(*fld) != nil {
		setDataType(fld, nil)
		changes = append(changes, "cleared data_type")
	}
	kind, _ := payload.KindOf(ch.DataType)
	same := fld.Type() == kind
	if !same {
		changes = append(changes, fmt.Sprintf(
			"%s kind became %s for channel %s (%s)",
			fld.Type(), kind, ch.Name, ch.DataType,
		))
	}
	if same && !isNumeric(kind) {
		if v, ok := fld.Variant.(TimestampReadField); ok && ch.IsIndex &&
			v.Fill != nil {
			v.Fill = nil
			fld.Variant = v
			changes = append(changes, "dropped fill")
		}
		return changes
	}
	var (
		opts    = optionsOf(*fld)
		numeric = isNumeric(fld.Type()) && isNumeric(kind)
		fits    = func(v any) bool {
			if !numeric || (kind == payload.KindFloat && !exactFloat(v, ch.DataType)) {
				return false
			}
			_, err := convert(v, ch.DataType)
			return err == nil
		}
		entries []entry
		fill    any
	)
	for _, e := range opts.entries {
		if fits(e.value) {
			entries = append(entries, e)
		} else {
			changes = append(changes, "dropped entry "+strconv.Quote(e.label))
		}
	}
	if opts.fill != nil {
		if fits(opts.fill) {
			fill = opts.fill
		} else {
			changes = append(changes, "dropped fill")
		}
	}
	fld.Variant = newVariant(fld.Base(), kind, entries, fill)
	return changes
}

// exactFloat returns true when v keeps its value as a sample of dt, a float type. An
// integer must be exact. A float64 always fits, as a float channel takes its nearest
// value.
func exactFloat(v any, dt telem.DataType) bool {
	f := toFloat(v)
	switch v := v.(type) {
	case int64:
		if f >= 0x1p63 || int64(f) != v {
			return false
		}
	case uint64:
		if f >= 0x1p64 || uint64(f) != v {
			return false
		}
	default:
		return true
	}
	return dt != telem.Float32T || float64(float32(f)) == f
}

func isNumeric(k payload.Kind) bool {
	return k == payload.KindFloat || k == payload.KindInteger ||
		k == payload.KindUnsigned
}

// newVariant returns a field of kind with base, entries, and fill. The values of
// entries and fill must be numbers that the kind holds.
func newVariant(
	base BaseReadField,
	kind payload.Kind,
	entries []entry,
	fill any,
) ReadFieldVariant {
	switch kind {
	case payload.KindFloat:
		v := FloatReadField{BaseReadField: base, EnumValues: []payload.FloatEntry{}}
		for _, e := range entries {
			v.EnumValues = append(v.EnumValues, payload.FloatEntry{
				Label: e.label, Value: payload.Float(toFloat(e.value)),
			})
		}
		if fill != nil {
			v.Fill = new(payload.Float(toFloat(fill)))
		}
		return v
	case payload.KindInteger:
		v := IntegerReadField{BaseReadField: base, EnumValues: []payload.IntegerEntry{}}
		for _, e := range entries {
			v.EnumValues = append(v.EnumValues, payload.IntegerEntry{
				Label: e.label, Value: payload.Integer(toInteger(e.value)),
			})
		}
		if fill != nil {
			v.Fill = new(payload.Integer(toInteger(fill)))
		}
		return v
	case payload.KindUnsigned:
		v := UnsignedReadField{
			BaseReadField: base,
			EnumValues:    []payload.UnsignedEntry{},
		}
		for _, e := range entries {
			v.EnumValues = append(v.EnumValues, payload.UnsignedEntry{
				Label: e.label, Value: payload.Unsigned(toUnsigned(e.value)),
			})
		}
		if fill != nil {
			v.Fill = new(payload.Unsigned(toUnsigned(fill)))
		}
		return v
	case payload.KindString:
		return StringReadField{
			BaseReadField: base,
			EnumValues:    []payload.StringEntry{},
		}
	case payload.KindBoolean:
		return BooleanReadField{
			BaseReadField: base,
			EnumValues:    []payload.BooleanEntry{},
		}
	case payload.KindTimestamp:
		return TimestampReadField{
			BaseReadField: base,
			TimeFormat:    payload.TimeFormatISO8601,
		}
	case payload.KindJSON:
		return JsonReadField{BaseReadField: base}
	case payload.KindUUID:
		return UuidReadField{BaseReadField: base}
	case payload.KindBytes:
		return BytesReadField{BaseReadField: base}
	}
	panic(errors.Newf("kafka: unknown read field kind %q", kind))
}

func toFloat(v any) float64 {
	switch v := v.(type) {
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	}
	return v.(float64)
}

func toInteger(v any) int64 {
	switch v := v.(type) {
	case float64:
		return int64(v)
	case uint64:
		return int64(v)
	}
	return v.(int64)
}

func toUnsigned(v any) uint64 {
	switch v := v.(type) {
	case float64:
		return uint64(v)
	case int64:
		return uint64(v)
	}
	return v.(uint64)
}

// dedupeLabels keeps the first entry of each label of fld. It returns the changes it
// made.
func dedupeLabels(fld *ReadField) []string {
	var changes []string
	switch v := fld.Variant.(type) {
	case FloatReadField:
		v.EnumValues, changes = dedupe(
			v.EnumValues,
			func(e payload.FloatEntry) string { return e.Label },
		)
		fld.Variant = v
	case IntegerReadField:
		v.EnumValues, changes = dedupe(
			v.EnumValues,
			func(e payload.IntegerEntry) string {
				return e.Label
			},
		)
		fld.Variant = v
	case UnsignedReadField:
		v.EnumValues, changes = dedupe(
			v.EnumValues,
			func(e payload.UnsignedEntry) string { return e.Label },
		)
		fld.Variant = v
	case StringReadField:
		v.EnumValues, changes = dedupe(
			v.EnumValues,
			func(e payload.StringEntry) string {
				return e.Label
			},
		)
		fld.Variant = v
	case BooleanReadField:
		v.EnumValues, changes = dedupe(
			v.EnumValues,
			func(e payload.BooleanEntry) string {
				return e.Label
			},
		)
		fld.Variant = v
	}
	return changes
}

func dedupe[E any](entries []E, label func(E) string) ([]E, []string) {
	var (
		seen    = make(set.Set[string], len(entries))
		kept    = entries[:0:0]
		changes []string
	)
	for _, e := range entries {
		l := label(e)
		if seen.Contains(l) {
			changes = append(changes, "dropped repeated label "+strconv.Quote(l))
			continue
		}
		seen.Add(l)
		kept = append(kept, e)
	}
	return kept, changes
}

// dataTypeOf returns the data type that fld gives a new channel, or nil when it gives
// none.
func dataTypeOf(fld ReadField) *telem.DataType {
	switch v := fld.Variant.(type) {
	case FloatReadField:
		return v.DataType
	case IntegerReadField:
		return v.DataType
	case UnsignedReadField:
		return v.DataType
	}
	return nil
}

// setDataType sets the data type that fld gives a new channel. A field of a kind with
// no data type is unchanged.
func setDataType(fld *ReadField, dt *telem.DataType) {
	switch v := fld.Variant.(type) {
	case FloatReadField:
		v.DataType = dt
		fld.Variant = v
	case IntegerReadField:
		v.DataType = dt
		fld.Variant = v
	case UnsignedReadField:
		v.DataType = dt
		fld.Variant = v
	}
}
