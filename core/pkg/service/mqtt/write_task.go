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

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
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

// writeTarget is one plain write target, ready to convert samples into payloads.
type writeTarget struct {
	topic string
	// template places the channel value at field 0 and fields[i] at field i+1.
	template xjson.Template
	// gens[i] generates the value of template field i+1 when it has no literal.
	gens     []payload.Generator
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
	if field == 0 {
		return p.t.encoder.Append(dst, p.sample)
	}
	return p.t.gens[field-1].Append(dst, telem.Now())
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
	pointer, err := xjson.ParsePointer(target.Channel.Base().Pointer)
	if err != nil {
		return writeTarget{}, errors.Wrap(err, path)
	}
	f, err := channelOf(target.Channel).Field(ch.DataType)
	if err != nil {
		return writeTarget{}, errors.Wrapf(err, "%s: channel %s", path, ch.Name)
	}
	if t.encoder, err = xjson.NewEncoder(f); err != nil {
		return writeTarget{}, errors.Wrapf(err, "%s: channel %s", path, ch.Name)
	}
	fields := []xjson.TemplateField{{Label: "channel.pointer", Pointer: pointer}}
	for _, wf := range target.Fields {
		field, gen, err := payload.TemplateField(wf)
		if err != nil {
			return writeTarget{}, errors.Wrap(err, path)
		}
		fields = append(fields, field)
		t.gens = append(t.gens, gen)
	}
	if t.template, err = xjson.NewTemplate(fields); err != nil {
		return writeTarget{}, errors.Wrap(err, path)
	}
	return t, nil
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
