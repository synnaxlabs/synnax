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
	"fmt"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
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

type slotKind uint8

const (
	slotValue slotKind = iota
	slotTimestamp
	slotUUID
)

// slot is how a payload fills one slot of its template.
type slot struct {
	kind   slotKind
	format xjson.TimeFormat
}

// writeTarget is one plain write target, ready to convert samples into payloads.
type writeTarget struct {
	topic    string
	template xjson.Template
	// slots[i] fills the slot of template field i.
	slots    []slot
	encoder  *xjson.Encoder
	qos      byte
	retained bool
}

var _ target = writeTarget{}

// String implements target.
func (t writeTarget) String() string { return "target " + t.topic }

// publication implements target.
func (t writeTarget) publication(series telem.Series, i int) (publication, error) {
	payload, err := t.template.Append(nil, payloadSlots{t: t, sample: series.At(i)})
	if err != nil {
		return publication{}, err
	}
	return publication{
		topic: t.topic, payload: payload, qos: t.qos, retained: t.retained,
	}, nil
}

// payloadSlots fills the slots of the payload of one sample.
type payloadSlots struct {
	t      writeTarget
	sample []byte
}

// AppendSlot implements xjson.Slots.
func (p payloadSlots) AppendSlot(dst []byte, field int) ([]byte, error) {
	s := p.t.slots[field]
	switch s.kind {
	case slotValue:
		return p.t.encoder.Append(dst, p.sample)
	case slotTimestamp:
		return xjson.AppendTimeStamp(dst, telem.Now(), s.format), nil
	default:
		dst = append(dst, '"')
		dst, err := uuid.New().AppendText(dst)
		return append(dst, '"'), err
	}
}

func parseTimeFormat(format *payload.TimeFormat) (xjson.TimeFormat, error) {
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
	}
	if err := validateTopic(target.Topic); err != nil {
		return writeTarget{}, err
	}
	pointer, err := xjson.ParsePointer(target.Channel.Pointer)
	if err != nil {
		return writeTarget{}, errors.Wrap(err, path)
	}
	f := xjson.Field{DataType: ch.DataType, Enums: enums(target.Channel.EnumValues)}
	if f.Type, err = xjson.ParseType(string(target.Channel.JSONType)); err != nil {
		return writeTarget{}, errors.Wrap(err, path)
	}
	if ch.DataType == telem.TimestampT {
		if f.TimeFormat, err = parseTimeFormat(target.Channel.TimeFormat); err != nil {
			return writeTarget{}, errors.Wrap(err, path)
		}
	}
	if len(f.Enums) > 0 && f.Type != xjson.String {
		return writeTarget{}, errors.Wrapf(
			validate.ErrValidation,
			"%s: enum values require the string JSON type", path,
		)
	}
	if t.encoder, err = xjson.NewEncoder(f); err != nil {
		return writeTarget{}, errors.Wrapf(err, "%s: channel %s", path, ch.Name)
	}
	fields := []xjson.TemplateField{{Label: "channel.pointer", Pointer: pointer}}
	t.slots = []slot{{kind: slotValue}}
	for _, wf := range target.Fields {
		var (
			base  BaseWriteField
			field xjson.TemplateField
			s     slot
		)
		switch v := wf.Variant.(type) {
		case StaticWriteField:
			base = v.BaseWriteField
			if !matchesJSONType(v.Value, v.JSONType) {
				return writeTarget{}, errors.Wrapf(
					validate.ErrValidation,
					"%s: static value at %s is not a %s",
					path, v.Pointer, v.JSONType,
				)
			}
			if field.Literal, err = xjson.Marshal(v.Value); err != nil {
				return writeTarget{}, errors.Wrapf(
					validate.ErrValidation,
					"%s: static value at %s is not JSON", path, v.Pointer,
				)
			}
		case GeneratedWriteField:
			base = v.BaseWriteField
			s.kind = slotUUID
			if v.Generator == payload.GeneratorTypeTimestamp {
				s.kind = slotTimestamp
				if s.format, err = parseTimeFormat(v.TimeFormat); err != nil {
					return writeTarget{}, errors.Wrap(err, path)
				}
			}
		case nil:
			return writeTarget{}, errors.Wrapf(
				validate.ErrValidation,
				"%s: a field is null",
				path,
			)
		default:
			panic(errors.Newf("mqtt: unknown write field variant %T", v))
		}
		field.Label = fmt.Sprintf("field %q", base.Key)
		if field.Pointer, err = xjson.ParsePointer(base.Pointer); err != nil {
			return writeTarget{}, errors.Wrap(err, path)
		}
		if len(field.Pointer) == 0 {
			return writeTarget{}, errors.Wrapf(
				validate.ErrValidation, "%s: a field requires a pointer", path,
			)
		}
		fields = append(fields, field)
		t.slots = append(t.slots, s)
	}
	if t.template, err = xjson.NewTemplate(fields); err != nil {
		return writeTarget{}, errors.Wrap(err, path)
	}
	return t, nil
}

// matchesJSONType reports whether a decoded static value has the JSON type t.
func matchesJSONType(value any, t payload.JSONType) bool {
	switch value.(type) {
	case string:
		return t == payload.JSONTypeString
	case bool:
		return t == payload.JSONTypeBoolean
	case int8, int16, int32, int64, uint8, uint16, uint32, uint64, float32, float64:
		return t == payload.JSONTypeNumber
	default:
		return false
	}
}

// writeSink is the driver.Sink of an MQTT write task.
type writeSink struct {
	pool *pool
	// targets holds the targets that each command channel triggers.
	targets map[channel.Key][]target
	// broker is the key of the broker device.
	broker device.Key
}

var _ driver.Sink = (*writeSink)(nil)

// Open implements driver.Sink.
func (s *writeSink) Open(ctx context.Context) (driver.Writer, error) {
	// A write task receives no messages, so its queue holds none.
	a, err := s.pool.attach(ctx, s.broker, 0)
	if err != nil {
		return nil, err
	}
	if err = a.settle(ctx); err != nil {
		s.pool.release(s.broker, a)
		return nil, err
	}
	return &writer{writeSink: s, attachment: a}, nil
}

// writer is the driver.Writer of one run of an MQTT write task.
type writer struct {
	*writeSink
	attachment *attachment
}

var _ driver.Writer = (*writer)(nil)

// Health implements driver.Writer. It reports each loss and recovery of the connection.
func (w *writer) Health(ctx context.Context) error {
	_, err := w.attachment.next(ctx)
	return err
}

// Close implements driver.Writer.
func (w *writer) Close() error {
	w.pool.release(w.broker, w.attachment)
	return nil
}

// Write implements driver.Writer. It publishes every sample it can, and returns the
// first error it met.
func (w *writer) Write(ctx context.Context, fr framer.Frame) error {
	var first error
	for key, series := range fr.Entries() {
		for _, t := range w.targets[key] {
			for i := range int(series.Len()) {
				pub, err := t.publication(series, i)
				if err != nil {
					// A value with no form in the payload, such as NaN in JSON, drops
					// its command. It does not stop the task.
					err = errors.Mark(errors.Wrap(err, t.String()), driver.ErrDegraded)
				} else {
					err = w.attachment.publish(
						ctx, pub.topic, pub.qos, pub.retained, pub.payload,
					)
				}
				if err != nil &&
					!errors.IsAny(err, driver.ErrTemporary, driver.ErrDegraded) {
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
