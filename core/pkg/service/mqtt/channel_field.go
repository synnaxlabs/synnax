// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// channelOf returns the kind and the options of f.
func channelOf(f ChannelField) payload.Channel {
	c := payload.Channel{Kind: f.Type()}
	switch v := f.Variant.(type) {
	case FloatChannelField:
		c.Float = v.EnumValues
	case IntegerChannelField:
		c.Integer = v.EnumValues
	case UnsignedChannelField:
		c.Unsigned = v.EnumValues
	case StringChannelField:
		c.String = v.EnumValues
	case BooleanChannelField:
		c.Boolean = v.EnumValues
	case TimestampChannelField:
		c.TimeFormat = v.TimeFormat
	}
	return c
}

// newChannelField returns the channel field of base with the kind and the options of c.
func newChannelField(base BaseChannelField, c payload.Channel) ChannelField {
	var v ChannelFieldVariant
	switch c.Kind {
	case payload.KindFloat:
		v = FloatChannelField{BaseChannelField: base, EnumValues: c.Float}
	case payload.KindInteger:
		v = IntegerChannelField{BaseChannelField: base, EnumValues: c.Integer}
	case payload.KindUnsigned:
		v = UnsignedChannelField{BaseChannelField: base, EnumValues: c.Unsigned}
	case payload.KindString:
		v = StringChannelField{BaseChannelField: base, EnumValues: c.String}
	case payload.KindBoolean:
		v = BooleanChannelField{BaseChannelField: base, EnumValues: c.Boolean}
	case payload.KindTimestamp:
		v = TimestampChannelField{BaseChannelField: base, TimeFormat: c.TimeFormat}
	case payload.KindJSON:
		v = JsonChannelField{BaseChannelField: base}
	case payload.KindUUID:
		v = UuidChannelField{BaseChannelField: base}
	case payload.KindBytes:
		v = BytesChannelField{BaseChannelField: base}
	default:
		panic("mqtt: unknown channel field kind " + string(c.Kind))
	}
	return ChannelField{Variant: v}
}

// plainTargets calls fn with the path and each plain target of cfg. fn may change the
// target.
func plainTargets(
	cfg *WriteConfig,
	fn func(path string, t *PlainWriteTarget) error,
) error {
	for i, t := range cfg.Targets {
		plain, ok := t.Variant.(PlainWriteTarget)
		if !ok {
			continue
		}
		if err := fn("targets."+strconv.Itoa(i)+".channel", &plain); err != nil {
			return err
		}
		cfg.Targets[i].Variant = plain
	}
	return nil
}

// validateWriteConfig checks the rules of cfg that need no channel: the entries of a
// channel field have unique values, so two NaN entries, or 0 and -0, collide, and each
// additional field holds a valid value.
func validateWriteConfig(cfg *WriteConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	return plainTargets(cfg, func(path string, t *PlainWriteTarget) error {
		if t.Channel.Variant == nil {
			return errors.Wrapf(validate.ErrValidation, "%s.type: required", path)
		}
		for _, f := range t.Fields {
			if err := payload.ValidateWriteField(f); err != nil {
				return errors.Wrapf(err, "field %q", f.Base().Key)
			}
		}
		return errors.Wrap(channelOf(t.Channel).Validate(), path)
	})
}

func targetChannels(
	ctx context.Context,
	svc *channel.Service,
	snap gorp.Snapshot,
	cfg *WriteConfig,
	enabledOnly bool,
) (map[channel.Key]channel.Channel, error) {
	var keys []channel.Key
	if err := plainTargets(cfg, func(_ string, t *PlainWriteTarget) error {
		if !enabledOnly || !t.Disabled {
			keys = append(keys, t.Channel.Base().Channel)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return channelsOf(ctx, svc, snap, keys)
}

// channelsOf returns the channels among keys, read through snap. A key with no channel
// is absent from the result.
func channelsOf(
	ctx context.Context,
	svc *channel.Service,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]channel.Channel, error) {
	keys = lo.Uniq(lo.Filter(keys, func(k channel.Key, _ int) bool { return k != 0 }))
	if len(keys) == 0 {
		return nil, nil
	}
	var found []channel.Channel
	if err := svc.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.KeyBy(found, channel.Channel.Key), nil
}

// resolveWriteConfig checks the channel field of each enabled plain target of cfg
// against its channel, narrows float entries to a float32 channel, and clears the data
// type. It skips a target with no channel, or one that does not exist.
func resolveWriteConfig(
	ctx context.Context,
	svc *channel.Service,
	snap gorp.Snapshot,
	cfg *WriteConfig,
) error {
	channels, err := targetChannels(ctx, svc, snap, cfg, true)
	if err != nil {
		return err
	}
	return plainTargets(cfg, func(path string, t *PlainWriteTarget) error {
		base := t.Channel.Base()
		ch, ok := channels[base.Channel]
		if t.Disabled || !ok {
			return nil
		}
		c, err := channelOf(t.Channel).Resolve(ch.DataType)
		if err != nil {
			return errors.Wrap(err, path)
		}
		base.DataType = nil
		t.Channel = newChannelField(base, c)
		return nil
	})
}

// reconcileWriteConfig changes the channel field of each plain target of cfg to agree
// with its channel, as payload.Channel.Reconcile does, clears its data type, and logs
// each change. A timestamp field on a channel that holds no timestamps becomes the
// kind of its data type, and then the kind the channel allows. A field with no channel
// agrees with its data type instead, which it keeps, and one with no data type takes
// the data type of its kind. It skips a field whose channel does not exist.
func reconcileWriteConfig(
	ctx context.Context,
	svc *channel.Service,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	cfg *WriteConfig,
) error {
	channels, err := targetChannels(ctx, svc, tx, cfg, false)
	if err != nil {
		return err
	}
	return plainTargets(cfg, func(path string, t *PlainWriteTarget) error {
		if t.Channel.Variant == nil {
			return errors.Wrapf(validate.ErrValidation, "%s.type: required", path)
		}
		base := t.Channel.Base()
		ch, ok := channels[base.Channel]
		var dt telem.DataType
		switch {
		case ok:
			dt = ch.DataType
		case base.Channel == 0 && base.DataType != nil &&
			len(payload.ChannelKinds(*base.DataType)) > 0:
			dt = *base.DataType
		case base.Channel == 0:
			dt = newChannelDataTypes[t.Channel.Type()]
			base.DataType = &dt
			ins.L.Warn(
				"gave an MQTT channel field with no channel the data type of its kind",
				zap.Stringer("config", cfg.Key),
				zap.String("field", path),
				zap.String("data_type", string(dt)),
			)
		default:
			return nil
		}
		c := channelOf(t.Channel)
		var changes []string
		if c.Kind == payload.KindTimestamp && dt != telem.TimestampT &&
			base.DataType != nil {
			if k, known := payload.KindOf(*base.DataType); known &&
				k != payload.KindTimestamp {
				c = payload.Channel{Kind: k}
				changes = append(
					changes,
					"changed the kind from timestamp to "+string(k),
				)
			}
		}
		c, more := c.Reconcile(dt)
		changes = append(changes, more...)
		for _, change := range changes {
			ins.L.Warn(
				"changed an MQTT channel field to agree with its channel",
				zap.Stringer("config", cfg.Key),
				zap.String("field", path),
				zap.Stringer("channel", base.Channel),
				zap.String("data_type", string(dt)),
				zap.String("change", change),
			)
		}
		if ok {
			base.DataType = nil
		}
		t.Channel = newChannelField(base, c)
		return nil
	})
}

// newChannelDataTypes holds the data type of a new channel that each kind writes.
var newChannelDataTypes = map[payload.Kind]telem.DataType{
	payload.KindFloat:     telem.Float64T,
	payload.KindInteger:   telem.Int64T,
	payload.KindUnsigned:  telem.Uint64T,
	payload.KindString:    telem.StringT,
	payload.KindBoolean:   telem.Uint8T,
	payload.KindTimestamp: telem.TimestampT,
	payload.KindJSON:      telem.JSONT,
	payload.KindUUID:      telem.UUIDT,
	payload.KindBytes:     telem.BytesT,
}

// newWriteMigration returns the migration that lifts stored write configs to v2 and
// reconciles their channel fields against the channels in svc.
func newWriteMigration(svc *channel.Service) gorp.Migration {
	return migrate.New(
		"v62_mqtt_typed_channel_fields",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v1.WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.WriteConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]WriteConfig, 0, len(olds))
			for _, old := range olds {
				cfg, err := v2.MigrateWriteConfig(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				if err = reconcileWriteConfig(ctx, svc, tx, ins, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}
