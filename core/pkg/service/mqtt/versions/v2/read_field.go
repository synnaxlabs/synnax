// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Channel is the part of a channel that a read field must agree with.
type Channel struct {
	// Name is the name of the channel.
	Name string
	// DataType is the data type of the channel.
	DataType telem.DataType
	// Index is true when the channel is an index channel.
	Index bool
}

// Channels returns the channels among keys, read through snap. A key with no channel
// is absent from the result.
type Channels func(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]Channel, error)

// ReadOf returns the kind and the options of f.
func ReadOf(f ReadField) payload.Read {
	switch v := f.Variant.(type) {
	case FloatReadField:
		return payload.FromFloat(v.FloatRead)
	case IntegerReadField:
		return payload.FromInteger(v.IntegerRead)
	case UnsignedReadField:
		return payload.FromUnsigned(v.UnsignedRead)
	case StringReadField:
		return payload.Read{Kind: payload.KindString, String: v.StringRead}
	case BooleanReadField:
		return payload.Read{Kind: payload.KindBoolean, Boolean: v.BooleanRead}
	case TimestampReadField:
		return payload.Read{Kind: payload.KindTimestamp, Timestamp: v.TimestampRead}
	}
	return payload.Read{Kind: f.Type()}
}

// NewReadField returns the read field of base with the kind and the options r.
func NewReadField(base BaseReadField, r payload.Read) ReadField {
	switch r.Kind {
	case payload.KindFloat:
		return ReadField{
			Variant: FloatReadField{BaseReadField: base, FloatRead: r.Float()},
		}
	case payload.KindInteger:
		return ReadField{
			Variant: IntegerReadField{BaseReadField: base, IntegerRead: r.Integer()},
		}
	case payload.KindUnsigned:
		return ReadField{
			Variant: UnsignedReadField{BaseReadField: base, UnsignedRead: r.Unsigned()},
		}
	case payload.KindString:
		return ReadField{
			Variant: StringReadField{BaseReadField: base, StringRead: r.String},
		}
	case payload.KindBoolean:
		return ReadField{
			Variant: BooleanReadField{BaseReadField: base, BooleanRead: r.Boolean},
		}
	case payload.KindTimestamp:
		return ReadField{
			Variant: TimestampReadField{
				BaseReadField: base,
				TimestampRead: r.Timestamp,
			},
		}
	case payload.KindJSON:
		return ReadField{Variant: JsonReadField{BaseReadField: base}}
	case payload.KindUUID:
		return ReadField{Variant: UuidReadField{BaseReadField: base}}
	case payload.KindBytes:
		return ReadField{Variant: BytesReadField{BaseReadField: base}}
	}
	panic("unknown read field kind " + string(r.Kind))
}

// PlainFields calls fn with the path of the fields of each plain entry of cfg, and the
// entry, and returns the first error that fn returns. A change fn makes to an element
// of the entry's fields changes cfg.
func PlainFields(
	cfg *ReadConfig,
	fn func(path string, entry PlainReadEntry) error,
) error {
	for i, e := range cfg.Entries {
		plain, ok := e.Variant.(PlainReadEntry)
		if !ok {
			continue
		}
		if err := fn("entries."+strconv.Itoa(i)+".fields", plain); err != nil {
			return err
		}
	}
	return nil
}

// FieldChannels returns the channels that the fields of cfg read into, through
// channels. A field with no channel, or one that does not exist, has no entry.
func FieldChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *ReadConfig,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	if err := PlainFields(cfg, func(_ string, entry PlainReadEntry) error {
		for _, f := range entry.Fields {
			if k := f.Base().Channel; k != 0 {
				keys = append(keys, k)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, snap, lo.Uniq(keys))
}

// ReconcileReadConfig changes each field of cfg to agree with its channel, as
// payload.Read.Reconcile does, and logs each change. It skips a field with no channel,
// or one that does not exist.
func ReconcileReadConfig(
	ctx context.Context,
	tx gorp.Tx,
	channels Channels,
	ins alamos.Instrumentation,
	cfg *ReadConfig,
) error {
	found, err := FieldChannels(ctx, tx, channels, cfg)
	if err != nil {
		return err
	}
	return PlainFields(cfg, func(path string, entry PlainReadEntry) error {
		for i, f := range entry.Fields {
			base := f.Base()
			ch, ok := found[base.Channel]
			if !ok {
				continue
			}
			r, changes := ReadOf(f).Reconcile(ch.DataType, ch.Index)
			for _, change := range changes {
				ins.L.Warn(
					"changed an MQTT read field to agree with its channel",
					zap.Stringer("config", cfg.Key),
					zap.String("field", path+"."+strconv.Itoa(i)),
					zap.Stringer("channel", base.Channel),
					zap.String("channel_name", ch.Name),
					zap.String("change", change),
				)
			}
			entry.Fields[i] = NewReadField(base, r)
		}
		return nil
	})
}
