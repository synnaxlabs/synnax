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
	"slices"
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
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
		return readTopic{}, err
	}
	indexSet := false
	for _, f := range entry.Fields {
		if f.Disabled {
			continue
		}
		path := "field " + f.Name + " of topic " + entry.Topic
		ch, ok := channels[f.Channel]
		if !ok {
			return readTopic{}, errors.Wrapf(
				validate.ErrValidation,
				"%s: channel %d does not exist",
				path,
				f.Channel,
			)
		}
		if err := xjson.CheckToSample(ch.DataType); err != nil {
			return readTopic{}, errors.Wrapf(err, "%s: channel %s", path, ch.Name)
		}
		pointer, err := xjson.ParsePointer(f.Pointer)
		if err != nil {
			return readTopic{}, errors.Wrap(err, path)
		}
		field := readField{
			name:     f.Name,
			pointer:  pointer,
			channel:  f.Channel,
			dataType: ch.DataType,
		}
		if ch.DataType == telem.TimestampT {
			if f.TimeFormat == nil {
				return readTopic{}, errors.Wrapf(
					validate.ErrValidation,
					"%s: a timestamp channel requires a time format", path,
				)
			}
			if field.format, err = xjson.ParseTimeFormat(
				string(*f.TimeFormat),
			); err != nil {
				return readTopic{}, errors.Wrap(err, path)
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
			return readTopic{}, errors.Wrapf(
				validate.ErrValidation,
				"%s: every channel of a topic must share one index channel", path,
			)
		}
		if f.Key == entry.Index {
			if !ch.IsIndex {
				return readTopic{}, errors.Wrapf(
					validate.ErrValidation,
					"%s: the index field must write to an index channel", path,
				)
			}
			t.indexField = len(t.fields)
		} else if ch.IsIndex {
			return readTopic{}, errors.Wrapf(
				validate.ErrValidation,
				"%s: only the index field can write to an index channel", path,
			)
		}
		t.fields = append(t.fields, field)
	}
	if len(t.fields) == 0 {
		return readTopic{}, errors.Wrapf(
			validate.ErrValidation, "topic %s has no enabled fields", entry.Topic,
		)
	}
	if entry.Index != "" && t.indexField == -1 {
		return readTopic{}, errors.Wrapf(
			validate.ErrValidation,
			"topic %s: index field %s does not exist or is disabled",
			entry.Topic, entry.Index,
		)
	}
	if t.index != 0 {
		t.keys = append(t.keys, t.index)
	}
	for i, f := range t.fields {
		if i != t.indexField {
			t.keys = append(t.keys, f.channel)
		}
	}
	return t, nil
}

// convert turns one payload into one sample for each channel of the topic, as a series
// in the order of keys. It converts every field or none, because a frame that writes to
// an index must hold every channel of that index.
func (t readTopic) convert(msg message) ([]telem.Series, telem.TimeStamp, error) {
	doc, err := xjson.Decode(msg.payload)
	if err != nil {
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
			return nil, 0, errors.Newf("field %s: no value at %s", f.name, f.pointer)
		}
		data, err := xjson.AppendSample(nil, f.dataType, value, f.format, f.enums)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "field %s", f.name)
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
	return series, stamp, nil
}

// batch gathers the samples of many messages into one frame, with one series for each
// channel.
type batch struct {
	keys   channel.Keys
	series []telem.Series
	// at is the position in series of each channel.
	at map[channel.Key]int
}

// append adds data, samples of the data type dt, to the series of key.
func (b *batch) append(key channel.Key, dt telem.DataType, data []byte) {
	i, ok := b.at[key]
	if !ok {
		if b.at == nil {
			b.at = make(map[channel.Key]int)
		}
		i, b.at[key] = len(b.series), len(b.series)
		b.keys = append(b.keys, key)
		b.series = append(b.series, telem.Series{DataType: dt})
	}
	b.series[i].Data = append(b.series[i].Data, data...)
}

func (b *batch) frame() framer.Frame {
	if len(b.keys) == 0 {
		return framer.Frame{}
	}
	return frame.NewMulti(b.keys, b.series)
}

// readSource is the driver.Source of an MQTT read task.
type readSource struct {
	pool   *pool
	topics map[string]readTopic
	// tags holds the Sparkplug B tags of the task.
	tags map[tagID]readTag
	// lastStamp is the last timestamp of each index channel, stored or written by any
	// run.
	lastStamp map[channel.Key]telem.TimeStamp
	framer    *framer.Service
	// stored holds the index channels that each run reads the stored data of. Empty
	// when the task saves no data.
	stored channel.Keys
	// broker is the key of the broker device.
	broker    device.Key
	queueSize int
	// birthGrace is how long an edge node has to publish a birth before it is offline.
	birthGrace time.Duration
}

var _ driver.Source = (*readSource)(nil)

// Open implements driver.Source.
func (s *readSource) Open(ctx context.Context) (driver.Reader, error) {
	if err := s.seed(ctx); err != nil {
		return nil, err
	}
	a, err := s.pool.attach(ctx, s.broker, s.queueSize)
	if err != nil {
		return nil, err
	}
	r := &reader{
		readSource: s,
		attachment: a,
		awaited:    make(map[tagScope]time.Time),
		offline:    make(set.Set[tagScope]),
	}
	if err = r.subscribe(ctx); err != nil {
		s.pool.release(s.broker, a)
		return nil, err
	}
	return r, nil
}

// seed raises lastStamp to the last timestamp stored for each index. After a restart,
// the broker repeats values that Cesium holds, and Cesium rejects a write inside stored
// data.
func (s *readSource) seed(ctx context.Context) error {
	for _, key := range s.stored {
		if err := s.seedIndex(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// seedIndex opens an iterator for one index only, because an iterator over channels on
// several Cores reports the seek result of one Core.
func (s *readSource) seedIndex(ctx context.Context, key channel.Key) error {
	iter, err := s.framer.OpenIterator(ctx, framer.IteratorConfig{
		Keys:      channel.Keys{key},
		Bounds:    telem.TimeRangeMax,
		ChunkSize: 1,
	})
	if err != nil {
		return err
	}
	if iter.SeekLast() && iter.Prev(framer.IteratorAutoSpan) {
		for _, series := range iter.Value().Entries() {
			if series.Len() > 0 {
				last := series.ValueAt[telem.TimeStamp](-1)
				s.lastStamp[key] = max(s.lastStamp[key], last)
			}
		}
	}
	return errors.Combine(iter.Error(), iter.Close())
}

// reader is the driver.Reader of one run of an MQTT read task.
type reader struct {
	*readSource
	attachment *attachment
	// awaited holds the edge nodes and devices that must publish a birth, each with the
	// time it has until.
	awaited map[tagScope]time.Time
	// offline holds the edge nodes and devices with no valid birth, and offlineErr
	// the warning that names them.
	offline    set.Set[tagScope]
	offlineErr error
	// degraded is the data loss that the task currently warns about.
	degraded struct {
		err   error
		until time.Time
	}
	// dropped is the drop count of the attachment that was last reported.
	dropped uint64
	// session is the session of the connection that awaited and offline belong to.
	session uint64
}

var _ driver.Reader = (*reader)(nil)

func (r *reader) subscribe(ctx context.Context) error {
	if err := r.attachment.settle(ctx); err != nil {
		return err
	}
	for _, t := range r.topics {
		if err := r.attachment.subscribe(ctx, t.topic, t.qos); err != nil {
			return err
		}
	}
	nodes := make(set.Set[sparkplug.NodeID])
	for id := range r.tags {
		nodes.Add(id.node)
	}
	for node := range nodes {
		if err := r.attachment.follow(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

// await gives every edge node of the task the time of birthGrace to publish a birth.
// A new session of the connection has missed the last birth of each one.
func (r *reader) await(now time.Time) {
	r.awaited = make(map[tagScope]time.Time)
	until := now.Add(r.birthGrace)
	for id := range r.tags {
		r.awaited[tagScope{node: id.node}] = until
	}
}

// awaitDevices gives each device of node that the task reads the time of birthGrace to
// publish a birth. A new session of an edge node needs a new birth of each device.
func (r *reader) awaitDevices(now time.Time, node sparkplug.NodeID) {
	until := now.Add(r.birthGrace)
	for id := range r.tags {
		if id.node == node && id.device != "" {
			r.awaited[id.tagScope] = until
		}
	}
}

// Close implements driver.Reader.
func (r *reader) Close() error {
	r.pool.release(r.broker, r.attachment)
	return nil
}

// Read implements driver.Reader. It returns every message that waits on the queue, up
// to the size of the queue, in one frame.
func (r *reader) Read(ctx context.Context) (framer.Frame, error) {
	waitCtx := ctx
	if deadline, ok := r.deadline(); ok {
		// The deadline wakes a quiet task, so that it can change its warning.
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	msg, err := r.attachment.next(waitCtx)
	now := time.Now()
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			r.expire(now)
			return framer.Frame{}, r.health()
		}
		return framer.Frame{}, err
	}
	var b batch
	r.apply(now, msg, &b)
	for range r.queueSize - 1 {
		msg, ok := r.attachment.poll()
		if !ok {
			break
		}
		r.apply(now, msg, &b)
	}
	if dropped := r.attachment.dropped.Load(); dropped != r.dropped {
		r.dropped = dropped
		r.degrade(now, errors.New(
			"messages arrive faster than the task can write them, and the oldest "+
				"are dropped",
		))
	}
	r.expire(now)
	return b.frame(), r.health()
}

// apply adds the samples of msg to b. The first message of a new session awaits the
// births of that session, also when it is itself a birth.
func (r *reader) apply(now time.Time, msg message, b *batch) {
	if msg.session > r.session {
		r.session = msg.session
		r.await(now)
	}
	switch {
	case msg.sparkplug != nil:
		r.convertEvent(now, msg, b)
	case msg.topic != "":
		r.convert(now, msg, b)
	}
}

// deadline returns the next time at which the warning of the task changes with no
// message.
func (r *reader) deadline() (deadline time.Time, ok bool) {
	if r.degraded.err != nil {
		deadline, ok = r.degraded.until, true
	}
	for _, until := range r.awaited {
		if !ok || until.Before(deadline) {
			deadline, ok = until, true
		}
	}
	return deadline, ok
}

// expire ends the warning of a data loss that is over, and takes the edge nodes and
// devices that gave no birth in time as offline.
func (r *reader) expire(now time.Time) {
	if r.degraded.err != nil && !now.Before(r.degraded.until) {
		r.degraded.err = nil
	}
	expired := false
	for scope, until := range r.awaited {
		if !now.Before(until) {
			r.offline.Add(scope)
			delete(r.awaited, scope)
			expired = true
		}
	}
	if expired {
		r.refreshOffline()
	}
}

// health returns the warning of the task, or nil for a task with none.
func (r *reader) health() error {
	if r.degraded.err != nil {
		return r.degraded.err
	}
	return r.offlineErr
}

// refreshOffline rebuilds the offline warning after r.offline changes.
func (r *reader) refreshOffline() {
	if len(r.offline) == 0 {
		r.offlineErr = nil
		return
	}
	scopes := make([]string, 0, len(r.offline))
	for scope := range r.offline {
		scopes = append(scopes, scope.String())
	}
	slices.Sort(scopes)
	r.offlineErr = errors.Mark(
		errors.Newf("%s is offline", strings.Join(scopes, ", ")), driver.ErrDegraded,
	)
}

// degrade starts or extends the warning of the task. The warning keeps its first
// text for as long as it lasts, so that a burst of losses writes one status.
func (r *reader) degrade(now time.Time, err error) {
	if r.degraded.err == nil {
		r.degraded.err = errors.Mark(err, driver.ErrDegraded)
	}
	r.degraded.until = now.Add(degradedHold)
}

// convert adds the samples of msg to b. A message that carries nothing to write adds
// none.
func (r *reader) convert(now time.Time, msg message, b *batch) {
	t, ok := r.topics[msg.topic]
	if !ok || (msg.retained && t.retainedIgnored) {
		return
	}
	series, stamp, err := t.convert(msg)
	if err != nil {
		r.degrade(now, errors.Wrapf(err, "rejected a message on topic %s", t.topic))
		return
	}
	if t.index != 0 {
		// An index takes timestamps in rising order only. A retained message repeats a
		// value that the task may have, so a stale one is no loss.
		if stamp <= r.lastStamp[t.index] {
			if !msg.retained {
				r.degrade(now, errors.Newf(
					"rejected a message on topic %s: its timestamp is not after the "+
						"last one written",
					t.topic,
				))
			}
			return
		}
		r.lastStamp[t.index] = stamp
	}
	for i, key := range t.keys {
		b.append(key, series[i].DataType, series[i].Data)
	}
}

// convertEvent applies one Sparkplug B message: a birth or a death changes which edge
// nodes are offline, a lost birth gives the edge node the time of birthGrace to publish
// a new one, and the values of the tags of the task go into b. A message of an older
// session gives its values only, since a new session can reach the reader before the
// messages queued in the old one.
func (r *reader) convertEvent(now time.Time, msg message, b *batch) {
	ev := msg.sparkplug
	current := msg.session == r.session
	if ev.BirthLost {
		if node := (tagScope{node: ev.Node}); current && r.reads(node) {
			r.awaited[node] = now.Add(r.birthGrace)
		}
		return
	}
	scope := tagScope{node: ev.Node, device: ev.Device}
	birth := false
	switch ev.Type {
	case sparkplug.NDeath, sparkplug.DDeath:
		if current && r.reads(scope) {
			r.offline.Add(scope)
			r.refreshOffline()
		}
		return
	case sparkplug.NBirth, sparkplug.DBirth:
		birth = true
		if current {
			delete(r.awaited, scope)
			delete(r.offline, scope)
			if ev.Type == sparkplug.NBirth {
				r.awaitDevices(now, ev.Node)
			}
			r.refreshOffline()
			r.checkBirth(now, scope, ev.Metrics)
		}
	}
	for _, m := range ev.Metrics {
		t, ok := r.tags[tagID{tagScope: scope, name: m.Name}]
		if !ok || m.Value == nil {
			continue
		}
		if m.Historical {
			r.degrade(now, errors.Newf(
				"rejected a historical value of %s: the task does not write values "+
					"that an edge node stored while offline",
				t,
			))
			continue
		}
		data, err := sparkplug.AppendSample(nil, t.dataType, m.Value)
		if err != nil {
			r.degrade(now, errors.Wrapf(err, "rejected a value of %s", t))
			continue
		}
		if t.index != 0 {
			stamp := m.Timestamp
			if stamp == 0 {
				stamp = msg.received
			}
			// An index takes timestamps in rising order only. A birth repeats values
			// that the task may have, so a stale one is no loss.
			if stamp <= r.lastStamp[t.index] {
				if !birth {
					r.degrade(now, errors.Newf(
						"rejected a value of %s: its timestamp is not after the last "+
							"one written",
						t,
					))
				}
				continue
			}
			r.lastStamp[t.index] = stamp
			b.append(
				t.index,
				telem.TimestampT,
				telem.ByteOrder.AppendUint64(nil, uint64(stamp)),
			)
		}
		b.append(t.channel, t.dataType, data)
	}
}

// reads reports whether the task has a tag in scope, or, for an edge node, in one of
// its devices.
func (r *reader) reads(scope tagScope) bool {
	for id := range r.tags {
		if id.node == scope.node && (scope.device == "" || id.device == scope.device) {
			return true
		}
	}
	return false
}

// checkBirth warns about the tags of the task that the birth of scope did not declare.
func (r *reader) checkBirth(
	now time.Time,
	scope tagScope,
	metrics []sparkplug.Metric,
) {
	declared := make(set.Set[string], len(metrics))
	for _, m := range metrics {
		declared.Add(m.Name)
	}
	for id := range r.tags {
		if id.tagScope != scope {
			continue
		}
		if !declared.Contains(id.name) {
			r.degrade(now, errors.Newf("%s is not in the birth of its %s", id, scope))
		}
	}
}
