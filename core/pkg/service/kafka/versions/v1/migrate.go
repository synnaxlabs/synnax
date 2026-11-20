// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"math"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// liftField returns the JSON field of base for old, the data type of a new channel for
// it, and a description of each change. The data type of ch, when old has a channel,
// picks the kind. Otherwise the stored data type picks it, and a numeric kind keeps it
// as the data type of a new channel. The returned data type is empty for any other
// field. An index field lifts to the timestamp kind, and so does a field with a time
// format, no entries, and the default or an unknown data type. An entry that the kind
// cannot hold is dropped.
func liftField(
	old ReadField,
	index bool,
	ch *Channel,
	base icd.JSONBaseField,
) (icd.JSONField, telem.DataType, []string) {
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
	kind, known := payload.KindOf(dt)
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

// liftRead returns the JSON field of base with kind and the options of old.
func liftRead(
	old ReadField,
	kind payload.Kind,
	base icd.JSONBaseField,
	f enumFilter,
) icd.JSONField {
	if field, ok := liftEnumRead(old, kind, base, f); ok {
		return field
	}
	for _, e := range old.EnumValues {
		f.drop(e, "a "+string(kind)+" field takes no numeric entries")
	}
	switch kind {
	case payload.KindString:
		return icd.JSONField{Variant: icd.StringJSONField{JSONBaseField: base}}
	case payload.KindTimestamp:
		format := payload.TimeFormatISO8601
		if old.TimeFormat != nil {
			format = *old.TimeFormat
		}
		return icd.JSONField{Variant: icd.TimestampJSONField{
			JSONBaseField: base,
			TimeFormat:    format,
		}}
	case payload.KindJSON:
		return icd.JSONField{Variant: icd.JSONJSONField{JSONBaseField: base}}
	case payload.KindUUID:
		return icd.JSONField{Variant: icd.UuidJSONField{JSONBaseField: base}}
	case payload.KindBytes:
		return icd.JSONField{Variant: icd.BytesJSONField{JSONBaseField: base}}
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned,
		payload.KindBoolean:
	}
	panic("unknown read field kind " + string(kind))
}

// liftEnumRead returns the JSON field of base for a kind that takes enum entries, with
// the entries of old that f keeps, and false for any other kind.
func liftEnumRead(
	old ReadField,
	kind payload.Kind,
	base icd.JSONBaseField,
	f enumFilter,
) (icd.JSONField, bool) {
	switch kind {
	case payload.KindFloat:
		return icd.JSONField{Variant: icd.FloatJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, floatEntry),
		}}, true
	case payload.KindInteger:
		return icd.JSONField{Variant: icd.IntegerJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, integerEntry),
		}}, true
	case payload.KindUnsigned:
		return icd.JSONField{Variant: icd.UnsignedJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, unsignedEntry),
		}}, true
	case payload.KindBoolean:
		return icd.JSONField{Variant: icd.BooleanJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, booleanEntry),
		}}, true
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	return icd.JSONField{}, false
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

// MigrateReadRecord lifts a v0 read record to v1. The record has no ICD until
// MigrateReadConfig stages it.
func MigrateReadRecord(ctx context.Context, old v0.ReadRecord) (ReadRecord, error) {
	return autoMigrateReadRecord(ctx, old)
}

// MigrateReadConfig lifts a v0 read config to v1. It moves each record into Inline, for
// ResolveReadConfig to add back with an ICD.
func MigrateReadConfig(ctx context.Context, old v0.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil || len(old.Records) == 0 {
		return cfg, err
	}
	staged := make([]InlineRecord, len(old.Records))
	for i, r := range old.Records {
		staged[i] = InlineRecord{
			Key:       r.Key,
			RecordKey: r.RecordKey,
			Index:     r.Index,
			Fields:    r.Fields,
		}
	}
	cfg.Records, cfg.Inline = []ReadRecord{}, &InlineRead{Records: staged}
	return cfg, nil
}

// CreateICD creates ic through tx.
type CreateICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

// NoteLift returns the note for ResolveReadConfig that logs each change to a field of
// the config with key through ins.
func NoteLift(ins alamos.Instrumentation, key uuid.UUID) func(path, change string) {
	return func(path, change string) {
		ins.L.Warn(
			"changed a Kafka read field to v1",
			zap.Stringer("config", key),
			zap.String("field", path),
			zap.String("change", change),
		)
	}
}

// StagedRecordsKVPrefix is the KV key prefix under which ReadMigration stages the
// records of each config. The rest of the key is the config key as a string, and the
// value is the InlineRead that holds the records, as JSON.
const StagedRecordsKVPrefix = "sy_kafka_read_staged_records/"

// StagedRecordsKVKey returns the staging KV key of the records of the config with the
// given key.
func StagedRecordsKVKey(key uuid.UUID) []byte {
	return []byte(StagedRecordsKVPrefix + key.String())
}

// ReadMigration lifts each stored v0 read config to v1. The v1 shape stores no inline
// records, so ReadMigration stages the records of each config under
// StagedRecordsKVPrefix.
var ReadMigration = migrate.New(
	"v63_kafka_read_stage",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var olds []v0.ReadConfig
		if err := gorp.NewRetrieve[uuid.UUID, v0.ReadConfig]().
			Entries(&olds).
			Exec(ctx, tx); err != nil || len(olds) == 0 {
			return err
		}
		news := make([]ReadConfig, len(olds))
		for i, old := range olds {
			cfg, err := MigrateReadConfig(ctx, old)
			if err != nil {
				return errors.Wrapf(err, "config %s", old.Key)
			}
			if cfg.Inline != nil {
				b, err := json.Marshal(cfg.Inline)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if err = tx.Set(ctx, StagedRecordsKVKey(cfg.Key), b); err != nil {
					return err
				}
				cfg.Inline = nil
			}
			news[i] = cfg
		}
		return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
	},
)
