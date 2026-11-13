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
	"slices"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/errors"
	xpayload "github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// channelOf returns the kind and the options of f.
func channelOf(f ChannelField) xpayload.Channel {
	c := xpayload.Channel{Kind: f.Type()}
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
func newChannelField(base BaseChannelField, c xpayload.Channel) ChannelField {
	var v ChannelFieldVariant
	switch c.Kind {
	case xpayload.KindFloat:
		v = FloatChannelField{BaseChannelField: base, EnumValues: c.Float}
	case xpayload.KindInteger:
		v = IntegerChannelField{BaseChannelField: base, EnumValues: c.Integer}
	case xpayload.KindUnsigned:
		v = UnsignedChannelField{BaseChannelField: base, EnumValues: c.Unsigned}
	case xpayload.KindString:
		v = StringChannelField{BaseChannelField: base, EnumValues: c.String}
	case xpayload.KindBoolean:
		v = BooleanChannelField{BaseChannelField: base, EnumValues: c.Boolean}
	case xpayload.KindTimestamp:
		v = TimestampChannelField{BaseChannelField: base, TimeFormat: c.TimeFormat}
	case xpayload.KindJSON:
		v = JsonChannelField{BaseChannelField: base}
	case xpayload.KindUUID:
		v = UuidChannelField{BaseChannelField: base}
	case xpayload.KindBytes:
		v = BytesChannelField{BaseChannelField: base}
	default:
		panic("http: unknown channel field kind " + string(c.Kind))
	}
	return ChannelField{Variant: v}
}

func channelPath(i int) []string {
	return []string{"endpoints", strconv.Itoa(i), "channel"}
}

// ValidateWriteConfig returns an error wrapping validate.ErrValidation when cfg breaks
// its schema rules, a channel field has no kind or two entries with one stored value,
// or a body field holds an invalid value. Two NaN entries, or 0 and -0, collide.
func ValidateWriteConfig(cfg *WriteConfig) error {
	v := validate.New("http.write_config")
	v.Exec(cfg.Validate)
	for i, ep := range cfg.Endpoints {
		v.Exec(func() error {
			if ep.Channel.Variant == nil {
				return validate.PathedError(
					errors.Wrap(validate.ErrValidation, "required"),
					append(channelPath(i), "type")...,
				)
			}
			return validate.PathedError(
				channelOf(ep.Channel).Validate(),
				channelPath(i)...,
			)
		})
		for _, f := range ep.Fields {
			v.Exec(func() error {
				return errors.Wrapf(
					xpayload.ValidateWriteField(f),
					"endpoint %q: field %q",
					ep.Key,
					f.Base().Key,
				)
			})
		}
	}
	return v.Error()
}

func writeChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *WriteConfig,
	enabledOnly bool,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, ep := range cfg.Endpoints {
		if ep.Channel.Variant == nil || (enabledOnly && ep.Disabled) {
			continue
		}
		if k := ep.Channel.Base().Channel; k != 0 {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, snap, lo.Uniq(keys))
}

// ResolveWriteConfig checks the channel field of each enabled endpoint of cfg against
// its channel, narrows float entries to a float32 channel, and clears the data type. It
// skips an endpoint with no channel, or one that does not exist. It returns an error
// wrapping validate.ErrValidation when the channel does not allow the field's kind, or
// an entry does not fit the channel.
func ResolveWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *WriteConfig,
) error {
	found, err := writeChannels(ctx, snap, channels, cfg, true)
	if err != nil {
		return err
	}
	v := validate.New("http.write_config")
	for i := range cfg.Endpoints {
		ep := &cfg.Endpoints[i]
		if ep.Disabled || ep.Channel.Variant == nil {
			continue
		}
		base := ep.Channel.Base()
		ch, ok := found[base.Channel]
		if !ok {
			continue
		}
		v.Exec(func() error {
			c, err := channelOf(ep.Channel).Resolve(ch.DataType)
			if err != nil {
				return validate.PathedError(err, channelPath(i)...)
			}
			base.DataType = nil
			ep.Channel = newChannelField(base, c)
			return nil
		})
	}
	return v.Error()
}

// ReconcileWriteConfig changes the channel field of each endpoint of cfg to agree with
// its channel, as payload.Channel.Reconcile does, and clears its data type. A timestamp
// field on a channel that holds no timestamps becomes the kind of its data type when
// the channel allows it. A field with no channel agrees with its data type instead,
// which it keeps. It skips a field whose channel does not exist. warn reports each
// change.
func ReconcileWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *WriteConfig,
	warn Warn,
) error {
	found, err := writeChannels(ctx, snap, channels, cfg, false)
	if err != nil {
		return err
	}
	for i := range cfg.Endpoints {
		ep := &cfg.Endpoints[i]
		if ep.Channel.Variant == nil {
			continue
		}
		base := ep.Channel.Base()
		ch, ok := found[base.Channel]
		var dt telem.DataType
		switch {
		case ok:
			dt = ch.DataType
		case base.Channel == 0 && base.DataType != nil:
			dt = *base.DataType
		default:
			continue
		}
		c := channelOf(ep.Channel)
		var changes []string
		if ok && c.Kind == xpayload.KindTimestamp && dt != telem.TimestampT &&
			base.DataType != nil {
			if k, known := xpayload.KindOf(*base.DataType); known &&
				slices.Contains(xpayload.ChannelKinds(dt), k) {
				c = xpayload.Channel{Kind: k}
				changes = append(changes, "changed the kind from timestamp to "+string(k))
			}
		}
		c, more := c.Reconcile(dt)
		for _, change := range append(changes, more...) {
			warn(
				"changed an HTTP channel field to agree with its channel",
				zap.String("endpoint", ep.Key),
				zap.Uint32("channel", uint32(base.Channel)),
				zap.String("data_type", string(dt)),
				zap.String("change", change),
			)
		}
		if ok {
			base.DataType = nil
		}
		ep.Channel = newChannelField(base, c)
	}
	return nil
}

// NewWriteMigration returns the migration that lifts stored v4 write configs to v5,
// reconciles their channel fields with channels, and logs each change.
func NewWriteMigration(channels Channels) gorp.Migration {
	return migrate.New(
		"v62_http_typed_write_fields",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v4.WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, v4.WriteConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]WriteConfig, 0, len(olds))
			for _, old := range olds {
				warn := func(msg string, fields ...zap.Field) {
					ins.L.Warn(msg, append(fields, zap.Stringer("config", old.Key))...)
				}
				cfg, err := MigrateWriteConfig(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if err = ReconcileWriteConfig(
					ctx,
					tx,
					channels,
					&cfg,
					warn,
				); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}
