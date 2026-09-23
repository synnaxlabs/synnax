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
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// publication is one message to publish.
type publication struct {
	topic    string
	payload  []byte
	qos      byte
	retained bool
}

// target turns the samples of one command channel into messages.
type target interface {
	// publication returns the message for sample i of series.
	publication(series telem.Series, i int) (publication, error)
	// String names the target in an error.
	String() string
}

// extraField is a static or generated field of a payload.
type extraField struct {
	// static is the JSON text of a fixed value. It is decoded for each payload, so
	// that payloads never share a mutable value. Empty for a generated field.
	static    []byte
	pointer   xjson.Pointer
	generator GeneratorType
	format    xjson.TimeFormat
}

// writeTarget is one plain write target, ready to convert samples into payloads.
type writeTarget struct {
	enums    xjson.ReverseEnumMap
	topic    string
	pointer  xjson.Pointer
	extras   []extraField
	dataType telem.DataType
	format   xjson.TimeFormat
	jsonType xjson.Type
	qos      byte
	retained bool
}

var _ target = writeTarget{}

// String implements target.
func (t writeTarget) String() string { return "target " + t.topic }

// publication implements target.
func (t writeTarget) publication(series telem.Series, i int) (publication, error) {
	payload, err := t.payload(series, i)
	return publication{
		topic: t.topic, payload: payload, qos: t.qos, retained: t.retained,
	}, err
}

func parseTimeFormat(format *TimeFormat) (xjson.TimeFormat, error) {
	if format == nil {
		return xjson.ISO8601, nil
	}
	return xjson.ParseTimeFormat(string(*format))
}

// newWriteTarget validates target against ch, its command channel.
func newWriteTarget(target PlainWriteTarget, ch channel.Channel) (writeTarget, error) {
	path := "target " + target.Topic
	t := writeTarget{
		topic:    target.Topic,
		qos:      qosLevel(target.Qos),
		retained: target.Retained,
		dataType: ch.DataType,
	}
	err := validateTopic(target.Topic)
	if err != nil {
		return t, err
	}
	if t.pointer, err = xjson.ParsePointer(target.Channel.Pointer); err != nil {
		return t, errors.Wrap(err, path)
	}
	if t.jsonType, err = xjson.ParseType(string(target.Channel.JSONType)); err != nil {
		return t, errors.Wrap(err, path)
	}
	if ch.DataType == telem.TimestampT {
		if target.Channel.TimeFormat == nil {
			return t, errors.Wrapf(
				validate.ErrValidation,
				"%s: a timestamp channel requires a time format", path,
			)
		}
		if t.format, err = parseTimeFormat(target.Channel.TimeFormat); err != nil {
			return t, errors.Wrap(err, path)
		}
	} else if err = xjson.CheckFromSample(ch.DataType, t.jsonType); err != nil {
		return t, errors.Wrapf(err, "%s: channel %s", path, ch.Name)
	}
	if len(target.Channel.EnumValues) > 0 {
		if t.jsonType != xjson.String {
			return t, errors.Wrapf(
				validate.ErrValidation,
				"%s: enum values require the string JSON type", path,
			)
		}
		t.enums = make(xjson.ReverseEnumMap, len(target.Channel.EnumValues))
		for _, e := range target.Channel.EnumValues {
			t.enums[e.Value] = e.Label
		}
	}
	if len(t.pointer) == 0 && len(target.Fields) > 0 {
		return t, errors.Wrapf(
			validate.ErrValidation,
			"%s: a channel value that is the whole payload allows no other fields",
			path,
		)
	}
	for _, f := range target.Fields {
		var extra extraField
		switch field := f.Variant.(type) {
		case StaticWriteField:
			if extra.pointer, err = xjson.ParsePointer(field.Pointer); err != nil {
				return t, errors.Wrap(err, path)
			}
			if !matchesJSONType(field.Value, field.JSONType) {
				return t, errors.Wrapf(
					validate.ErrValidation,
					"%s: static value at %s is not a %s",
					path, field.Pointer, field.JSONType,
				)
			}
			if extra.static, err = xjson.Marshal(field.Value); err != nil {
				return t, errors.Wrapf(
					validate.ErrValidation,
					"%s: static value at %s is not JSON", path, field.Pointer,
				)
			}
		case GeneratedWriteField:
			if extra.pointer, err = xjson.ParsePointer(field.Pointer); err != nil {
				return t, errors.Wrap(err, path)
			}
			extra.generator = field.Generator
			if extra.format, err = parseTimeFormat(field.TimeFormat); err != nil {
				return t, errors.Wrap(err, path)
			}
		}
		if len(extra.pointer) == 0 {
			return t, errors.Wrapf(
				validate.ErrValidation, "%s: a field requires a pointer", path,
			)
		}
		t.extras = append(t.extras, extra)
	}
	// A payload with a blocked path fails now, not on the first command.
	if _, err = t.payload(telem.MakeSeries(ch.DataType, 1), 0); err != nil {
		return t, errors.Wrap(err, path)
	}
	return t, nil
}

// matchesJSONType reports whether a decoded static value has the JSON type t.
func matchesJSONType(value any, t JSONType) bool {
	switch value.(type) {
	case string:
		return t == JSONTypeString
	case bool:
		return t == JSONTypeBoolean
	case int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64:
		return t == JSONTypeNumber
	default:
		return false
	}
}

// payload returns the JSON payload for sample i of series.
func (t writeTarget) payload(series telem.Series, i int) ([]byte, error) {
	var (
		value any
		err   error
	)
	if t.dataType == telem.TimestampT {
		value = xjson.FromTimeStamp(series.ValueAt[telem.TimeStamp](i), t.format)
	} else if value, err = xjson.FromSample(
		t.dataType, series.At(i), t.jsonType, t.enums,
	); err != nil {
		return nil, err
	}
	var doc any
	for _, extra := range t.extras {
		var extraValue any
		switch {
		case extra.static != nil:
			if extraValue, err = xjson.Decode(extra.static); err != nil {
				return nil, err
			}
		case extra.generator == GeneratorTypeTimestamp:
			extraValue = xjson.FromTimeStamp(telem.Now(), extra.format)
		default:
			extraValue = uuid.New().String()
		}
		if doc, err = extra.pointer.Set(doc, extraValue); err != nil {
			return nil, err
		}
	}
	if doc, err = t.pointer.Set(doc, value); err != nil {
		return nil, err
	}
	return xjson.Marshal(doc)
}

// writeSink is the driver.Sink of an MQTT write task.
type writeSink struct {
	pool       *pool
	attachment *attachment
	// targets holds the targets that each command channel triggers.
	targets map[channel.Key][]target
	dev     device.Device
}

var _ driver.Sink = (*writeSink)(nil)

// Start implements driver.Sink.
func (s *writeSink) Start(ctx context.Context) (err error) {
	// A write task receives no messages, so its queue holds none.
	if s.attachment, err = s.pool.attach(s.dev, 0); err != nil {
		return err
	}
	if err = s.attachment.settle(ctx); err != nil {
		s.pool.release(s.dev.Key, s.attachment)
	}
	return err
}

// Health implements driver.Sink. It reports each loss and recovery of the connection.
func (s *writeSink) Health(ctx context.Context) error {
	_, err := s.attachment.next(ctx)
	return err
}

// Stop implements driver.Sink.
func (s *writeSink) Stop() error {
	s.pool.release(s.dev.Key, s.attachment)
	return nil
}

// Write implements driver.Sink. It publishes every sample it can, and returns the
// first error it met.
func (s *writeSink) Write(ctx context.Context, fr framer.Frame) error {
	var first error
	for key, series := range fr.Entries() {
		for _, t := range s.targets[key] {
			for i := range int(series.Len()) {
				pub, err := t.publication(series, i)
				if err != nil {
					// A value with no form in the payload, such as NaN in JSON, drops
					// its command. It does not stop the task.
					err = errors.Wrap(driver.ErrTemporary, err.Error())
				} else {
					err = s.attachment.publish(
						ctx, pub.topic, pub.qos, pub.retained, pub.payload,
					)
				}
				if err != nil && !errors.Is(err, driver.ErrTemporary) {
					return errors.Wrap(err, t.String())
				}
				if first == nil {
					first = err
				}
			}
		}
	}
	return first
}
