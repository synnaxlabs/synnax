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
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// degradedHold is how long a read task shows a warning after it last lost data.
const degradedHold = 5 * time.Second

// qosLevel returns the MQTT quality of service level of q.
func qosLevel(q QoS) byte {
	switch q {
	case QoSAtLeastOnce:
		return 1
	case QoSExactlyOnce:
		return 2
	}
	return 0
}

// validateTopic checks a topic that a task subscribes or publishes to. A task names
// one topic, so that a message maps to one set of channels.
func validateTopic(topic string) error {
	if topic == "" {
		return errors.Wrap(validate.ErrValidation, "topic: required")
	}
	if strings.ContainsAny(topic, "+#") {
		return errors.Wrapf(
			validate.ErrValidation,
			"topic %s: must not hold the wildcards + or #", topic,
		)
	}
	return nil
}

// readField is one value to extract from a payload.
type readField struct {
	enums   xjson.EnumMap
	name    string
	pointer xjson.Pointer
	channel channel.Key
	format  xjson.TimeFormat
	// dataType is the data type of the channel, which the Core is the authority on.
	dataType telem.DataType
}

// readTopic is one plain read entry, ready to convert payloads into frames.
type readTopic struct {
	topic  string
	fields []readField
	// keys are the channels of one frame: the index, when there is one, then the
	// fields that are not the index.
	keys channel.Keys
	// index is the index channel of every field. Zero for virtual channels.
	index channel.Key
	// indexField is the position in fields of the field that gives the timestamp.
	// It is -1 when the sample takes its arrival time.
	indexField      int
	qos             byte
	retainedIgnored bool
}

// newReadTopic validates entry against channels, which holds every channel that a
// field of the entry names.
func newReadTopic(
	entry PlainReadEntry,
	channels map[channel.Key]channel.Channel,
) (readTopic, error) {
	t := readTopic{
		topic:           entry.Topic,
		qos:             qosLevel(entry.Qos),
		retainedIgnored: entry.RetainedIgnored,
		indexField:      -1,
	}
	if err := validateTopic(entry.Topic); err != nil {
		return t, err
	}
	indexSet := false
	for _, f := range entry.Fields {
		if f.Disabled {
			continue
		}
		path := "field " + f.Name + " of topic " + entry.Topic
		ch, ok := channels[f.Channel]
		if !ok {
			return t, errors.Wrapf(
				validate.ErrValidation,
				"%s: channel %d does not exist",
				path,
				f.Channel,
			)
		}
		if !xjson.SupportsSampleTarget(ch.DataType) {
			return t, errors.Wrapf(
				validate.ErrValidation,
				"%s: channel %s has the unsupported data type %s",
				path, ch.Name, ch.DataType,
			)
		}
		pointer, err := xjson.ParsePointer(f.Pointer)
		if err != nil {
			return t, errors.Wrap(err, path)
		}
		field := readField{
			name:     f.Name,
			pointer:  pointer,
			channel:  f.Channel,
			dataType: ch.DataType,
		}
		if ch.DataType == telem.TimestampT {
			if f.TimeFormat == nil {
				return t, errors.Wrapf(
					validate.ErrValidation,
					"%s: a timestamp channel requires a time format", path,
				)
			}
			if field.format, err = xjson.ParseTimeFormat(
				string(*f.TimeFormat),
			); err != nil {
				return t, errors.Wrap(err, path)
			}
		}
		if len(f.EnumValues) > 0 {
			field.enums = make(xjson.EnumMap, len(f.EnumValues))
			for _, e := range f.EnumValues {
				field.enums[e.Label] = e.Value
			}
		}
		if !indexSet {
			t.index, indexSet = ch.Index(), true
		} else if ch.Index() != t.index {
			return t, errors.Wrapf(
				validate.ErrValidation,
				"%s: every channel of a topic must share one index channel", path,
			)
		}
		if f.Key == entry.Index {
			if !ch.IsIndex {
				return t, errors.Wrapf(
					validate.ErrValidation,
					"%s: the index field must write to an index channel", path,
				)
			}
			t.indexField = len(t.fields)
		}
		t.fields = append(t.fields, field)
	}
	if len(t.fields) == 0 {
		return t, errors.Wrapf(
			validate.ErrValidation, "topic %s has no enabled fields", entry.Topic,
		)
	}
	if entry.Index != "" && t.indexField == -1 {
		return t, errors.Wrapf(
			validate.ErrValidation,
			"topic %s: index field %s does not exist or is disabled",
			entry.Topic, entry.Index,
		)
	}
	if t.index != 0 {
		t.keys = append(t.keys, t.index)
	}
	for _, f := range t.fields {
		if f.channel != t.index {
			t.keys = append(t.keys, f.channel)
		}
	}
	return t, nil
}

// convert turns one payload into a frame of one sample for each channel of the
// topic. It converts every field or none, because a frame that writes to an index
// must hold every channel of that index.
func (t readTopic) convert(msg message) (framer.Frame, telem.TimeStamp, error) {
	var doc any
	dec := json.NewDecoder(bytes.NewReader(msg.payload))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		// A payload that is not JSON is taken as a bare string, such as ON.
		doc = string(msg.payload)
	}
	var (
		stamp  = msg.received
		series = make([]telem.Series, 0, len(t.keys))
	)
	if t.index != 0 {
		series = append(series, telem.Series{DataType: telem.TimestampT})
	}
	for i, f := range t.fields {
		value, ok := f.pointer.Get(doc)
		if !ok {
			return framer.Frame{}, 0, errors.Newf(
				"field %s: no value at %s", f.name, f.pointer,
			)
		}
		data, err := xjson.AppendSample(nil, f.dataType, value, f.format, f.enums)
		if err != nil {
			return framer.Frame{}, 0, errors.Wrapf(err, "field %s", f.name)
		}
		if i == t.indexField {
			stamp = telem.TimeStamp(telem.ByteOrder.Uint64(data))
			continue
		}
		series = append(series, telem.Series{DataType: f.dataType, Data: data})
	}
	if t.index != 0 {
		series[0].Data = telem.ByteOrder.AppendUint64(nil, uint64(stamp))
	}
	return frame.NewMulti(t.keys, series), stamp, nil
}

// readSource is the driver.Source of an MQTT read task.
type readSource struct {
	pool       *pool
	attachment *attachment
	topics     map[string]readTopic
	// tags holds the Sparkplug B tags of the task.
	tags map[tagID]readTag
	// lastStamp is the last timestamp written to each index channel.
	lastStamp map[channel.Key]telem.TimeStamp
	// awaited holds the edge nodes that have until awaitedUntil to publish a birth.
	awaited      set.Set[sparkplug.NodeID]
	awaitedUntil time.Time
	// offline holds the edge nodes and devices with no valid birth.
	offline set.Set[tagScope]
	// degraded is the data loss that the task currently warns about.
	degraded struct {
		err   error
		until time.Time
	}
	dev       device.Device
	queueSize int
	// birthGrace is how long an edge node has to publish a birth before it is
	// offline.
	birthGrace time.Duration
	// dropped is the drop count of the attachment that was last reported.
	dropped uint64
}

var _ driver.Source = (*readSource)(nil)

// Start implements driver.Source.
func (s *readSource) Start(ctx context.Context) (err error) {
	if s.attachment, err = s.pool.attach(s.dev, s.queueSize); err != nil {
		return err
	}
	s.dropped = 0
	if err = s.attachment.settle(ctx); err != nil {
		s.pool.release(s.dev.Key, s.attachment)
		return err
	}
	for _, t := range s.topics {
		if err = s.attachment.subscribe(ctx, t.topic, t.qos); err != nil {
			s.pool.release(s.dev.Key, s.attachment)
			return err
		}
	}
	s.offline = make(set.Set[tagScope])
	s.await(time.Now())
	for node := range s.awaited {
		if err = s.attachment.follow(ctx, node); err != nil {
			s.pool.release(s.dev.Key, s.attachment)
			return err
		}
	}
	return nil
}

// await gives every edge node of the task the time of birthGrace to publish a birth. A
// task that connects has missed the last birth of each one.
func (s *readSource) await(now time.Time) {
	s.awaited = make(set.Set[sparkplug.NodeID])
	for id := range s.tags {
		s.awaited.Add(id.node)
	}
	s.awaitedUntil = now.Add(s.birthGrace)
}

// Stop implements driver.Source.
func (s *readSource) Stop() error {
	s.pool.release(s.dev.Key, s.attachment)
	return nil
}

// Read implements driver.Source.
func (s *readSource) Read(ctx context.Context) (framer.Frame, error) {
	waitCtx := ctx
	if deadline, ok := s.deadline(); ok {
		// The deadline wakes a quiet task, so that it can change its warning.
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	msg, err := s.attachment.next(waitCtx)
	now := time.Now()
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			s.expire(now)
			return framer.Frame{}, s.health()
		}
		return framer.Frame{}, err
	}
	if dropped := s.attachment.dropped.Load(); dropped != s.dropped {
		s.dropped = dropped
		s.degrade(now, errors.New(
			"messages arrive faster than the task can write them, and the oldest "+
				"are dropped",
		))
	}
	var fr framer.Frame
	switch {
	case msg.sparkplug != nil:
		fr = s.convertEvent(now, msg)
	case msg.topic == "":
		// The connection is back, and the births of the outage are lost.
		s.await(now)
	default:
		fr = s.convert(now, msg)
	}
	s.expire(now)
	return fr, s.health()
}

// deadline returns the next time at which the warning of the task changes with no
// message.
func (s *readSource) deadline() (deadline time.Time, ok bool) {
	if s.degraded.err != nil {
		deadline, ok = s.degraded.until, true
	}
	if len(s.awaited) > 0 && (!ok || s.awaitedUntil.Before(deadline)) {
		deadline, ok = s.awaitedUntil, true
	}
	return deadline, ok
}

// expire ends the warning of a data loss that is over, and takes the edge nodes that
// gave no birth in time as offline.
func (s *readSource) expire(now time.Time) {
	if s.degraded.err != nil && !now.Before(s.degraded.until) {
		s.degraded.err = nil
	}
	if len(s.awaited) > 0 && !now.Before(s.awaitedUntil) {
		for node := range s.awaited {
			s.offline.Add(tagScope{node: node})
		}
		clear(s.awaited)
	}
}

// health returns the warning of the task, or nil for a task with none.
func (s *readSource) health() error {
	if s.degraded.err != nil {
		return errors.Wrap(driver.ErrDegraded, s.degraded.err.Error())
	}
	if len(s.offline) == 0 {
		return nil
	}
	scopes := make([]string, 0, len(s.offline))
	for scope := range s.offline {
		scopes = append(scopes, scope.String())
	}
	slices.Sort(scopes)
	return errors.Wrapf(
		driver.ErrDegraded, "%s is offline", strings.Join(scopes, ", "),
	)
}

// degrade starts or extends the warning of the task. The warning keeps its first
// text for as long as it lasts, so that a burst of losses writes one status.
func (s *readSource) degrade(now time.Time, err error) {
	if s.degraded.err == nil {
		s.degraded.err = err
	}
	s.degraded.until = now.Add(degradedHold)
}

// convert returns the frame of msg, or an empty frame for a message that carries
// nothing to write.
func (s *readSource) convert(now time.Time, msg message) framer.Frame {
	t, ok := s.topics[msg.topic]
	if !ok || (msg.retained && t.retainedIgnored) {
		return framer.Frame{}
	}
	fr, stamp, err := t.convert(msg)
	if err != nil {
		s.degrade(now, errors.Wrapf(err, "rejected a message on topic %s", t.topic))
		return framer.Frame{}
	}
	if t.index != 0 {
		// An index takes timestamps in rising order only.
		if stamp <= s.lastStamp[t.index] {
			s.degrade(now, errors.Newf(
				"rejected a message on topic %s: its timestamp is not after the "+
					"last one written",
				t.topic,
			))
			return framer.Frame{}
		}
		s.lastStamp[t.index] = stamp
	}
	return fr
}

// convertEvent applies one Sparkplug B message: a birth or a death changes which edge
// nodes are offline, and the values of the tags of the task become a frame.
func (s *readSource) convertEvent(now time.Time, msg message) framer.Frame {
	ev := msg.sparkplug
	scope := tagScope{node: ev.Node, device: ev.Device}
	birth := false
	switch ev.Type {
	case sparkplug.NDeath, sparkplug.DDeath:
		if s.reads(scope) {
			s.offline.Add(scope)
		}
		return framer.Frame{}
	case sparkplug.NBirth, sparkplug.DBirth:
		birth = true
		delete(s.awaited, ev.Node)
		delete(s.offline, scope)
		s.checkBirth(now, scope, ev.Metrics)
	}
	var (
		keys   channel.Keys
		series []telem.Series
		// at is the position in series of each channel.
		at = make(map[channel.Key]int)
	)
	add := func(key channel.Key, dt telem.DataType) *telem.Series {
		i, ok := at[key]
		if !ok {
			i, at[key] = len(series), len(series)
			keys = append(keys, key)
			series = append(series, telem.Series{DataType: dt})
		}
		return &series[i]
	}
	for _, m := range ev.Metrics {
		t, ok := s.tags[tagID{tagScope: scope, name: m.Name}]
		if !ok || m.Value == nil || m.Historical {
			continue
		}
		data, err := t.appendSample(nil, m.Value)
		if err != nil {
			s.degrade(now, errors.Wrapf(err, "rejected a value of %s", t))
			continue
		}
		if t.index != 0 {
			stamp := m.Timestamp
			if stamp == 0 {
				stamp = msg.received
			}
			// An index takes timestamps in rising order only. A birth repeats values
			// that the task may have, so a stale one is no loss.
			if stamp <= s.lastStamp[t.index] {
				if !birth {
					s.degrade(now, errors.Newf(
						"rejected a value of %s: its timestamp is not after the last "+
							"one written",
						t,
					))
				}
				continue
			}
			s.lastStamp[t.index] = stamp
			index := add(t.index, telem.TimestampT)
			index.Data = telem.ByteOrder.AppendUint64(index.Data, uint64(stamp))
		}
		values := add(t.channel, t.dataType)
		values.Data = append(values.Data, data...)
	}
	if len(keys) == 0 {
		return framer.Frame{}
	}
	return frame.NewMulti(keys, series)
}

// reads reports whether the task has a tag in scope, or, for an edge node, in one of
// its devices.
func (s *readSource) reads(scope tagScope) bool {
	for id := range s.tags {
		if id.node == scope.node && (scope.device == "" || id.device == scope.device) {
			return true
		}
	}
	return false
}

// checkBirth warns about the tags of the task that the birth of scope did not declare.
func (s *readSource) checkBirth(
	now time.Time,
	scope tagScope,
	metrics []sparkplug.Metric,
) {
	declared := make(set.Set[string], len(metrics))
	for _, m := range metrics {
		declared.Add(m.Name)
	}
	for id := range s.tags {
		if id.tagScope != scope {
			continue
		}
		if !declared.Contains(id.name) {
			s.degrade(now, errors.Newf("%s is not in the birth of its %s", id, scope))
		}
	}
}
