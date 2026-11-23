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
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/write"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/errors"
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
	// publications returns the messages that the commands in fr send. It counts each
	// command in skips as read, and each command that it drops as skipped.
	publications(fr framer.Frame, skips *driver.Skips) iter.Seq[publication]
	// reset forgets the commands of an earlier run.
	reset()
	// String names the target in an error.
	String() string
}

// messageTarget publishes the documents of one ICD message to one topic.
type messageTarget struct {
	topic    string
	qos      byte
	retained bool
	docs     *write.Documents
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
	docs, err := write.NewDocuments(target.Message, ic, channels, path)
	if err != nil {
		return nil, errors.Wrap(err, path)
	}
	return &messageTarget{
		topic:    target.Topic,
		qos:      qos,
		retained: target.Retained,
		docs:     docs,
	}, nil
}

// String implements target.
func (t *messageTarget) String() string { return "target " + t.topic }

// reset implements target.
func (t *messageTarget) reset() { t.docs.Reset() }

// publications implements target.
func (t *messageTarget) publications(
	fr framer.Frame,
	skips *driver.Skips,
) iter.Seq[publication] {
	return func(yield func(publication) bool) {
		for doc := range t.docs.From(fr, skips) {
			if !yield(publication{
				topic: t.topic, payload: doc, qos: t.qos, retained: t.retained,
			}) {
				return
			}
		}
	}
}

// skip counts in skips a command for t that err dropped.
func skip(skips *driver.Skips, t target, err error) {
	skips.Skip(time.Now(), fmt.Sprintf("Command for %s: %s", t, err))
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
func (s *writeSink) Open(
	ctx context.Context,
	skips *driver.Skips,
) (driver.Writer, error) {
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
		skips:      skips,
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
// first error it met. It drops a command with no form in its message, such as NaN in
// JSON, and counts it in the skips.
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
		for pub := range t.publications(fr, w.skips) {
			err := w.attachment.publish(
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
	return first
}
