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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// channelOf returns the kind and the options of wc.
func channelOf(wc WriteChannel) payload.Channel {
	c := payload.Channel{Kind: wc.Type()}
	switch v := wc.Variant.(type) {
	case FloatWriteChannel:
		c.Float = v.EnumValues
	case IntegerWriteChannel:
		c.Integer = v.EnumValues
	case UnsignedWriteChannel:
		c.Unsigned = v.EnumValues
	case StringWriteChannel:
		c.String = v.EnumValues
	case BooleanWriteChannel:
		c.Boolean = v.EnumValues
	case TimestampWriteChannel:
		c.TimeFormat = v.TimeFormat
	}
	return c
}

// newWriteChannelConfig returns the write channel of base with the kind and the options
// of c.
func newWriteChannelConfig(base BaseWriteChannel, c payload.Channel) WriteChannel {
	var v WriteChannelVariant
	switch c.Kind {
	case payload.KindFloat:
		v = FloatWriteChannel{BaseWriteChannel: base, EnumValues: c.Float}
	case payload.KindInteger:
		v = IntegerWriteChannel{BaseWriteChannel: base, EnumValues: c.Integer}
	case payload.KindUnsigned:
		v = UnsignedWriteChannel{BaseWriteChannel: base, EnumValues: c.Unsigned}
	case payload.KindString:
		v = StringWriteChannel{BaseWriteChannel: base, EnumValues: c.String}
	case payload.KindBoolean:
		v = BooleanWriteChannel{BaseWriteChannel: base, EnumValues: c.Boolean}
	case payload.KindTimestamp:
		v = TimestampWriteChannel{BaseWriteChannel: base, TimeFormat: c.TimeFormat}
	case payload.KindJSON:
		v = JsonWriteChannel{BaseWriteChannel: base}
	case payload.KindUUID:
		v = UuidWriteChannel{BaseWriteChannel: base}
	case payload.KindBytes:
		v = BytesWriteChannel{BaseWriteChannel: base}
	default:
		panic("kafka: unknown write channel kind " + string(c.Kind))
	}
	return WriteChannel{Variant: v}
}

// validateWriteConfig checks cfg without reading its Synnax channels. The values of
// each channel's entries must differ, so two NaN entries, or 0 and -0, collide. An
// empty value pointer writes each sample as the raw record value, so every enabled
// channel must be bytes, and the record may place nothing else.
func validateWriteConfig(cfg *WriteConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	raw := cfg.Record.ValuePointer == ""
	if rec := cfg.Record; raw && (lo.FromPtr(rec.ChannelPointer) != "" ||
		lo.FromPtr(rec.TimestampPointer) != "" || len(rec.Fields) > 0) {
		return errors.Wrap(
			validate.ErrValidation,
			"record.value_pointer: an empty pointer must be the only record field",
		)
	}
	for _, wc := range cfg.Channels {
		b := wc.Base()
		if raw && !b.Disabled && wc.Type() != payload.KindBytes {
			return errors.Wrapf(
				validate.ErrValidation,
				"channel %q: an empty record.value_pointer requires the bytes kind",
				b.Key,
			)
		}
		if err := channelOf(wc).Validate(); err != nil {
			return errors.Wrapf(err, "channel %q", b.Key)
		}
	}
	return nil
}

// readChannels returns the channels among keys, read through snap. A key with no
// channel is absent from the result.
func readChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	svc *channel.Service,
	keys channel.Keys,
) (map[channel.Key]channel.Channel, error) {
	var channels []channel.Channel
	if err := svc.NewRetrieve().
		Where(func(channel.Retrieve) gorp.Filter[channel.Key, channel.Channel] {
			return gorp.MatchKeysFunc[channel.Key, channel.Channel](
				func(gorp.Context) ([]channel.Key, error) { return keys, nil },
			)
		}).
		Entries(&channels).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.KeyBy(channels, channel.Channel.Key), nil
}

// resolveWriteConfig checks each enabled channel of cfg against its Synnax channel,
// read through snap, and narrows float entries to a float32 channel. It skips channel
// 0 and a channel that does not exist.
func resolveWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	svc *channel.Service,
	cfg *WriteConfig,
) error {
	channels, err := readChannels(ctx, snap, svc, lo.FilterMap(
		cfg.Channels,
		func(wc WriteChannel, _ int) (channel.Key, bool) {
			b := wc.Base()
			return b.Channel, !b.Disabled && b.Channel != 0
		},
	))
	if err != nil {
		return err
	}
	for i, wc := range cfg.Channels {
		b := wc.Base()
		ch, ok := channels[b.Channel]
		if b.Disabled || !ok {
			continue
		}
		c, err := channelOf(wc).Resolve(ch.DataType)
		if err != nil {
			return errors.Wrapf(err, "channel %q", b.Key)
		}
		cfg.Channels[i] = newWriteChannelConfig(b, c)
	}
	return nil
}

// reconcileWriteConfig changes each channel of cfg to agree with its Synnax channel,
// read through tx, as payload.Channel.Reconcile does, and logs each change.
func reconcileWriteConfig(
	ctx context.Context,
	tx gorp.Tx,
	svc *channel.Service,
	ins alamos.Instrumentation,
	cfg *WriteConfig,
) error {
	channels, err := readChannels(ctx, tx, svc, lo.FilterMap(
		cfg.Channels,
		func(wc WriteChannel, _ int) (channel.Key, bool) {
			return wc.Base().Channel, wc.Base().Channel != 0
		},
	))
	if err != nil {
		return err
	}
	for i, wc := range cfg.Channels {
		b := wc.Base()
		ch, ok := channels[b.Channel]
		if !ok {
			continue
		}
		c, changes := channelOf(wc).Reconcile(ch.DataType)
		for _, change := range changes {
			ins.L.Warn(
				"changed a Kafka write channel to agree with its channel",
				zap.Stringer("config", cfg.Key),
				zap.String("field", b.Key),
				zap.Stringer("channel", ch.Key()),
				zap.String("channel_name", ch.Name),
				zap.String("data_type", string(ch.DataType)),
				zap.String("change", change),
			)
		}
		cfg.Channels[i] = newWriteChannelConfig(b, c)
	}
	return nil
}

// newWriteMigration returns the migration that lifts stored write configs to v1 and
// reconciles their channels against the channels in svc.
func newWriteMigration(svc *channel.Service) gorp.Migration {
	return migrate.New(
		"v62_kafka_typed_write_channels",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v0.WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, v0.WriteConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]WriteConfig, 0, len(olds))
			for _, old := range olds {
				cfg, err := v1.MigrateWriteConfig(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				if err = reconcileWriteConfig(ctx, tx, svc, ins, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}
