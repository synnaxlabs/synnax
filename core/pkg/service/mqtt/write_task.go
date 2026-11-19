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
	"iter"
	"slices"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// publication is one message to publish.
type publication struct {
	topic    string
	payload  []byte
	qos      byte
	retained bool
}

// target turns the commands of a frame into messages.
type target interface {
	// publications returns the message, or the error, of each command in fr.
	publications(fr framer.Frame) iter.Seq2[publication, error]
	// reset forgets the commands of an earlier run.
	reset()
	// String names the target in an error.
	String() string
}

// messageTarget publishes the documents of one ICD message to one topic. Document i of
// a frame holds command i of each bound channel, and the last command of a channel with
// fewer. A field with no command since the start sends its default. The target
// publishes nothing until each field with no default has a command. A command that its
// field cannot hold is dropped, and the field keeps its last command.
type messageTarget struct {
	topic    string
	qos      byte
	retained bool
	encoder  *codec.JSONEncoder
	// channels holds the command channel of each binding.
	channels channel.Keys
	received []bool
	// last holds the last command of each binding that received one.
	last [][]byte
	// next holds the commands of the document to encode, nil for a binding with none.
	next [][]byte
	// spare holds a buffer for each binding to copy a new command into.
	spare [][]byte
	// rows holds the commands of each binding in one frame.
	rows [][][]byte
}

var _ target = (*messageTarget)(nil)

// newMessageTarget validates target against ic, the ICD that holds its message.
// channels holds its command channels.
func newMessageTarget(
	target PlainWriteTarget,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
) (*messageTarget, error) {
	path := "target " + target.Topic
	qos, err := qosLevel(target.Qos)
	if err != nil {
		return nil, errors.Wrap(err, path)
	}
	if err = validateTopic(target.Topic); err != nil {
		return nil, err
	}
	msg, payload, err := jsonMessageOf(ic, target.Message.Message)
	if err != nil {
		return nil, errors.Wrap(err, path)
	}
	bindings := target.Message.Bindings
	if len(bindings) == 0 {
		return nil, errors.Wrapf(
			validate.ErrValidation, "%s: message %s binds no fields", path, msg.Name,
		)
	}
	n := len(bindings)
	t := &messageTarget{
		topic:    target.Topic,
		qos:      qos,
		retained: target.Retained,
		channels: make(channel.Keys, n),
		received: make([]bool, n),
		last:     make([][]byte, n),
		next:     make([][]byte, n),
		spare:    make([][]byte, n),
		rows:     make([][][]byte, n),
	}
	encoded := make([]codec.Binding, n)
	for i, b := range bindings {
		ch, ok := channels[b.Channel]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"%s: channel %d does not exist", path, b.Channel,
			)
		}
		encoded[i] = codec.Binding{Field: b.Field, DataType: ch.DataType}
		t.channels[i] = b.Channel
		t.last[i], t.spare[i] = []byte{}, []byte{}
	}
	if t.encoder, err = codec.NewJSONEncoder(codec.JSONMessage{
		Name:     msg.Name,
		Payload:  payload,
		Bindings: encoded,
	}); err != nil {
		return nil, errors.Wrap(err, path)
	}
	return t, nil
}

// String implements target.
func (t *messageTarget) String() string { return "target " + t.topic }

// reset implements target.
func (t *messageTarget) reset() { clear(t.received) }

// publications implements target.
func (t *messageTarget) publications(fr framer.Frame) iter.Seq2[publication, error] {
	return func(yield func(publication, error) bool) {
		for row := range t.gather(fr) {
			for b, rows := range t.rows {
				t.next[b] = nil
				if row < len(rows) {
					t.next[b] = append(t.spare[b][:0], rows[row]...)
				} else if t.received[b] {
					t.next[b] = t.last[b]
				}
			}
			if !t.ready() {
				t.commit(row)
				continue
			}
			payload, err := t.encoder.Encode(nil, t.next)
			if err == nil {
				t.commit(row)
			}
			pub := publication{
				topic: t.topic, payload: payload, qos: t.qos, retained: t.retained,
			}
			if !yield(pub, err) {
				return
			}
		}
	}
}

// gather fills rows with the commands of each binding in fr, and returns the count of
// documents that they make.
func (t *messageTarget) gather(fr framer.Frame) int {
	for b := range t.rows {
		t.rows[b] = t.rows[b][:0]
	}
	for key, s := range fr.Entries() {
		for b, ch := range t.channels {
			if ch == key {
				t.rows[b] = slices.AppendSeq(t.rows[b], s.Samples())
			}
		}
	}
	n := 0
	for _, rows := range t.rows {
		n = max(n, len(rows))
	}
	return n
}

// ready reports whether next holds a command for each field with no default.
func (t *messageTarget) ready() bool {
	for b, command := range t.next {
		if command == nil && !t.encoder.HasDefault(b) {
			return false
		}
	}
	return true
}

// commit makes the commands of row the last commands of their bindings.
func (t *messageTarget) commit(row int) {
	for b, rows := range t.rows {
		if row < len(rows) {
			t.last[b], t.spare[b] = t.next[b], t.last[b]
			t.received[b] = true
		}
	}
}

// writeSink is the driver.Sink of an MQTT write task.
type writeSink struct {
	pool    *pool
	targets []target
	// byChannel holds the indices in targets of the targets that each command channel
	// feeds.
	byChannel map[channel.Key][]int
	// broker is the key of the broker device.
	broker device.Key
}

var _ driver.Sink = (*writeSink)(nil)

// add adds t to s, fed by the command channels of keys.
func (s *writeSink) add(t target, keys ...channel.Key) {
	for _, key := range keys {
		s.byChannel[key] = append(s.byChannel[key], len(s.targets))
	}
	s.targets = append(s.targets, t)
}

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
	for _, t := range s.targets {
		t.reset()
	}
	return &writer{
		writeSink:  s,
		attachment: a,
		skips:      driver.NewSkips("command"),
		touched:    make([]bool, len(s.targets)),
		fed:        make([]int, 0, len(s.targets)),
	}, nil
}

// writer is the driver.Writer of one run of an MQTT write task.
type writer struct {
	*writeSink
	attachment *attachment
	// skips counts the commands that the run sends and drops.
	skips *driver.Skips
	// touched marks the targets that the frame of a Write feeds, and fed lists them.
	touched []bool
	fed     []int
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

// Write implements driver.Writer. It publishes every message it can, and returns the
// first error it met. A command with no form in its message, such as NaN in JSON, is
// dropped, and Write returns the warning of the skips.
func (w *writer) Write(ctx context.Context, fr framer.Frame) error {
	w.fed = w.fed[:0]
	for key := range fr.Keys() {
		for _, i := range w.byChannel[key] {
			if !w.touched[i] {
				w.touched[i] = true
				w.fed = append(w.fed, i)
			}
		}
	}
	for _, i := range w.fed {
		w.touched[i] = false
	}
	var first error
	for _, i := range w.fed {
		t := w.targets[i]
		for pub, err := range t.publications(fr) {
			w.skips.Read()
			if err != nil {
				w.skips.Skip(time.Now(), fmt.Sprintf("Command for %s: %s", t, err))
				continue
			}
			err = w.attachment.publish(
				ctx, pub.topic, pub.qos, pub.retained, pub.payload,
			)
			if err != nil &&
				!errors.IsAny(err, driver.ErrTemporary, driver.ErrDegraded) {
				return errors.Wrap(err, t.String())
			}
			if first == nil {
				first = err
			}
		}
	}
	if first != nil {
		return first
	}
	if skipped := w.skips.Warning(time.Now()); skipped != nil {
		return skipped.Degraded()
	}
	return nil
}
