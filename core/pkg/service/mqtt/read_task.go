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
	"encoding/json/jsontext"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// degradedHold is how long a read task shows a warning after it last lost data.
const degradedHold = 5 * time.Second

// qosLevel returns the MQTT quality of service level of q. It returns an error that
// wraps validate.ErrValidation for an unknown q.
func qosLevel(q QoS) (byte, error) {
	switch q {
	case QoSAtMostOnce:
		return 0, nil
	case QoSAtLeastOnce:
		return 1, nil
	case QoSExactlyOnce:
		return 2, nil
	default:
		return 0, errors.Wrapf(validate.ErrValidation, "qos: unknown value %q", q)
	}
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

// topicSubscription is a topic that a read task subscribes to.
type topicSubscription struct {
	topic           string
	qos             byte
	retainedIgnored bool
}

// topicSample holds the samples that one message on a topic gives.
type topicSample struct {
	// index is the index channel of keys. Zero for virtual channels.
	index channel.Key
	keys  channel.Keys
	// series can share memory that the next convert of the topic overwrites.
	series []telem.Series
	stamp  telem.TimeStamp
}

// topicMessage reads the documents of one ICD message into its bound channels.
type topicMessage struct {
	name string
	// keys are the channels of one frame: the index, when there is one, then the
	// bound channels that are not the index.
	keys channel.Keys
	// index is the index channel of every binding. Zero for virtual channels.
	index channel.Key
	// indexField is the position in bindings of the field that gives the timestamp.
	// It is -1 when the sample takes its arrival time.
	indexField int
	dataTypes  []telem.DataType
}

// readTopic reads the topic of one plain entry, whose documents are each one of several
// ICD messages.
type readTopic struct {
	topicSubscription
	messages []*topicMessage
	decoder  *codec.JSON
	// samples holds, for each message, the sample of each binding in the last
	// document that message matched.
	samples [][][]byte
	// text holds a payload that is not JSON as a JSON string.
	text []byte
}

// newReadTopic validates entry against ic, the ICD it names, and channels, which holds
// every channel that a binding of the entry names.
func newReadTopic(
	entry PlainReadEntry,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
) (*readTopic, error) {
	qos, err := qosLevel(entry.Qos)
	if err != nil {
		return nil, err
	}
	t := &readTopic{
		topic:           entry.Topic,
		qos:             qos,
		retainedIgnored: entry.RetainedIgnored,
	}
	if err = validateTopic(entry.Topic); err != nil {
		return nil, err
	}
	if len(entry.Messages) == 0 {
		return nil, errors.Wrapf(
			validate.ErrValidation, "topic %s reads no messages", entry.Topic,
		)
	}
	decoded := make([]codec.JSONMessage, len(entry.Messages))
	for i, rm := range entry.Messages {
		m, err := newTopicMessage(rm, ic, channels, &decoded[i])
		if err != nil {
			return nil, errors.Wrapf(err, "topic %s", entry.Topic)
		}
		t.messages = append(t.messages, m)
		t.samples = append(t.samples, make([][]byte, len(decoded[i].Bindings)))
	}
	if t.decoder, err = codec.NewJSON(decoded); err != nil {
		return nil, errors.Wrapf(err, "topic %s", entry.Topic)
	}
	return t, nil
}

// newTopicMessage validates rm and fills decoded with what the decoder needs to read
// it.
func newTopicMessage(
	rm icd.ReadMessage,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
	decoded *codec.JSONMessage,
) (*topicMessage, error) {
	var msg *icd.Message
	for i := range ic.Messages {
		if ic.Messages[i].Key == rm.Message {
			msg = &ic.Messages[i]
			break
		}
	}
	if msg == nil {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"message %s does not exist in ICD %s", rm.Message, ic.Name,
		)
	}
	payload, ok := msg.Payload.Variant.(icd.JSONPayload)
	if !ok {
		return nil, errors.Wrapf(
			validate.ErrValidation, "message %s is not a JSON message", msg.Name,
		)
	}
	if len(rm.Bindings) == 0 {
		return nil, errors.Wrapf(
			validate.ErrValidation, "message %s binds no fields", msg.Name,
		)
	}
	m := &topicMessage{name: msg.Name, indexField: -1}
	bindings, err := m.bind(rm, channels)
	if err != nil {
		return nil, errors.Wrapf(err, "message %s", msg.Name)
	}
	*decoded = codec.JSONMessage{Name: msg.Name, Payload: payload, Bindings: bindings}
	if m.index != 0 {
		m.keys = append(m.keys, m.index)
	}
	for i, b := range rm.Bindings {
		if i != m.indexField {
			m.keys = append(m.keys, b.Channel)
		}
	}
	return m, nil
}

// bind sets the data types, the index channel, and the index field of m from the
// bindings of rm and their channels. It returns the codec binding of each.
func (m *topicMessage) bind(
	rm icd.ReadMessage,
	channels map[channel.Key]channel.Channel,
) ([]codec.Binding, error) {
	bindings := make([]codec.Binding, len(rm.Bindings))
	for i, b := range rm.Bindings {
		ch, ok := channels[b.Channel]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation, "channel %d does not exist", b.Channel,
			)
		}
		bindings[i] = codec.Binding{Field: b.Field, DataType: ch.DataType}
		m.dataTypes = append(m.dataTypes, ch.DataType)
		if i == 0 {
			m.index = ch.Index()
		} else if ch.Index() != m.index {
			return nil, errors.Wrap(
				validate.ErrValidation, "every channel must share one index channel",
			)
		}
		isIndexField := rm.Index != nil && *rm.Index == b.Field
		switch {
		case isIndexField && !ch.IsIndex:
			return nil, errors.Wrap(
				validate.ErrValidation,
				"the index field must write to an index channel",
			)
		case !isIndexField && ch.IsIndex:
			return nil, errors.Wrap(
				validate.ErrValidation,
				"only the index field can write to an index channel",
			)
		case isIndexField:
			m.indexField = i
		}
	}
	if rm.Index != nil && m.indexField == -1 {
		return nil, errors.Wrap(
			validate.ErrValidation, "the index field has no channel",
		)
	}
	return bindings, nil
}

// convert reads msg as the message whose match rule its document meets. A payload that
// is not JSON is read as a bare string, such as ON.
func (t *readTopic) convert(msg message) (topicSample, error) {
	for _, samples := range t.samples {
		for i := range samples {
			samples[i] = samples[i][:0]
		}
	}
	i, ok, err := t.decoder.Decode(msg.payload, t.samples)
	if err != nil &&
		!jsontext.Value(msg.payload).IsValid(jsontext.AllowDuplicateNames(true)) {
		if t.text, err = jsontext.AppendQuote(t.text[:0], msg.payload); err != nil {
			return topicSample{}, err
		}
		i, ok, err = t.decoder.Decode(t.text, t.samples)
	}
	if err != nil {
		return topicSample{}, err
	}
	if !ok {
		return topicSample{}, errors.New("the payload matches no message")
	}
	return t.messages[i].sample(t.samples[i], msg.received), nil
}

// sample builds a topic sample from the samples of each binding in one document,
// stamped received when the message has no index field.
func (m *topicMessage) sample(samples [][]byte, received telem.TimeStamp) topicSample {
	s := topicSample{
		index:  m.index,
		keys:   m.keys,
		series: make([]telem.Series, 0, len(m.keys)),
		stamp:  received,
	}
	if m.index != 0 {
		s.series = append(s.series, telem.Series{DataType: telem.TimestampT})
	}
	for i, data := range samples {
		if i == m.indexField {
			s.stamp = telem.TimeStamp(telem.ByteOrder.Uint64(data))
			continue
		}
		s.series = append(s.series, telem.Series{DataType: m.dataTypes[i], Data: data})
	}
	if m.index != 0 {
		s.series[0].Data = telem.ByteOrder.AppendUint64(nil, uint64(s.stamp))
	}
	return s
}

// batch gathers the samples of many messages into one frame, with one series for each
// channel.
type batch struct {
	keys   channel.Keys
	series []telem.Series
	// ended holds the indexes whose domains end after the samples of the batch. The
	// batch takes no more samples once it holds any.
	ended channel.Keys
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

func (b *batch) result() driver.Batch {
	res := driver.Batch{EndDomains: b.ended}
	if len(b.keys) > 0 {
		res.Frame = frame.NewMulti(b.keys, b.series)
	}
	return res
}

// readSource is the driver.Source of an MQTT read task.
type readSource struct {
	pool   *pool
	topics map[string]*readTopic
	// tags holds the Sparkplug B tags of the task.
	tags map[tagID]readTag
	// lastStamp is the last timestamp of each index channel, stored or written by any
	// run.
	lastStamp map[channel.Key]telem.TimeStamp
	framer    *framer.Service
	// stored holds the index channels that each run reads the stored data of. Empty
	// when the task saves no data.
	stored channel.Keys
	// subject identifies the task to the writers of its historical values.
	subject control.Subject
	ins     alamos.Instrumentation
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
		skips:      driver.NewSkips("message"),
		awaited:    make(map[tagScope]time.Time),
		offline:    make(set.Set[tagScope]),
	}
	if err = r.subscribe(ctx); err != nil {
		s.pool.release(s.broker, a)
		return nil, err
	}
	r.history = newHistory(s)
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
	history    *history
	// held is a birth that waits for the next Read, since the domains that it ends
	// must end before its samples and after the samples of the last batch.
	held *message
	// awaited holds the edge nodes and devices that must publish a birth, each with the
	// time it has until.
	awaited map[tagScope]time.Time
	// offline holds the edge nodes and devices with no valid birth, and offlineErr
	// the warning that names them.
	offline    set.Set[tagScope]
	offlineErr error
	// skips counts the messages on plain topics that the task reads and skips.
	skips *driver.Skips
	// degraded is the other data loss that the task currently warns about.
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
		sub := t.topicSubscription
		if err := r.attachment.subscribe(ctx, sub.topic, sub.qos); err != nil {
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

// Close implements driver.Reader. The task calls it after it closes its writer, which
// frees the task to write the historical values that wait for a live value.
func (r *reader) Close() error {
	r.pool.release(r.broker, r.attachment)
	return r.history.close()
}

// next returns the held message, or else the next message on the queue.
func (r *reader) next(ctx context.Context) (message, error) {
	if r.held != nil {
		msg := *r.held
		r.held = nil
		return msg, nil
	}
	return r.attachment.next(ctx)
}

// Read implements driver.Reader. It returns every message that waits on the queue, up
// to the size of the queue, in one batch.
func (r *reader) Read(ctx context.Context) (driver.Batch, error) {
	waitCtx := ctx
	if deadline, ok := r.deadline(); ok {
		// The deadline wakes a quiet task, so that it can change its warning.
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	msg, err := r.next(waitCtx)
	now := time.Now()
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			r.expire(now)
			return driver.Batch{}, r.health(now)
		}
		return driver.Batch{}, err
	}
	var b batch
	r.apply(now, msg, &b)
	for range r.queueSize - 1 {
		if len(b.ended) > 0 {
			break
		}
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
	return b.result(), r.health(now)
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

// deadline returns the next time at which the warning of the task changes, or its
// historical values need attention, with no message.
func (r *reader) deadline() (deadline time.Time, ok bool) {
	if r.degraded.err != nil {
		deadline, ok = r.degraded.until, true
	}
	if until, due := r.skips.Deadline(); due && (!ok || until.Before(deadline)) {
		deadline, ok = until, true
	}
	for _, until := range r.awaited {
		if !ok || until.Before(deadline) {
			deadline, ok = until, true
		}
	}
	if until, due := r.history.deadline(time.Now()); due &&
		(!ok || until.Before(deadline)) {
		deadline, ok = until, true
	}
	return deadline, ok
}

// expire ends the warning of a data loss that is over, takes the edge nodes and devices
// that gave no birth in time as offline, and moves the historical values along.
func (r *reader) expire(now time.Time) {
	if r.degraded.err != nil && !now.Before(r.degraded.until) {
		r.degraded.err = nil
	}
	if err := r.history.tick(now); err != nil {
		r.degrade(now, err)
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

// health returns the warning of the task at now, or nil for a task with none.
func (r *reader) health(now time.Time) error {
	skipped := r.skips.Warning(now)
	if r.degraded.err != nil {
		return r.degraded.err
	}
	if skipped != nil {
		return skipped.Degraded()
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
	if !ok {
		return
	}
	sub := t.topicSubscription
	if msg.retained && sub.retainedIgnored {
		return
	}
	r.skips.Read()
	s, err := t.convert(msg)
	if err != nil {
		r.skips.Skip(now, fmt.Sprintf("Message on topic %s: %s", sub.topic, err))
		return
	}
	if s.index != 0 {
		// An index takes timestamps in rising order only. A retained message repeats a
		// value that the task may have, so a stale one is no loss.
		if s.stamp <= r.lastStamp[s.index] {
			if !msg.retained {
				r.skips.Skip(now, fmt.Sprintf(
					"Message on topic %s: its timestamp is not after the last one "+
						"written",
					sub.topic,
				))
			}
			return
		}
		r.lastStamp[s.index] = s.stamp
	}
	for i, key := range s.keys {
		b.append(key, s.series[i].DataType, s.series[i].Data)
	}
}

// convertEvent applies one Sparkplug B message: a birth or a death changes which edge
// nodes are offline, a lost birth gives the edge node the time of birthGrace to publish
// a new one, and the values of the tags of the task go into b. A death ends the domains
// of its tags, and so does a birth that starts a new session, for a death that went
// missing. A message of an older session gives its values only, since a new session can
// reach the reader before the messages queued in the old one.
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
			b.ended = r.endDomains(scope)
		}
		return
	case sparkplug.NBirth, sparkplug.DBirth:
		birth = true
		if current {
			if ev.NewSession {
				if b.ended = r.endDomains(scope); len(b.ended) > 0 {
					r.held = &msg
					return
				}
			}
			delete(r.awaited, scope)
			delete(r.offline, scope)
			if ev.Type == sparkplug.NBirth {
				r.awaitDevices(now, ev.Node)
			}
			r.refreshOffline()
			r.checkBirth(now, scope, ev.Metrics)
		}
	case sparkplug.NData,
		sparkplug.DData,
		sparkplug.NCmd,
		sparkplug.DCmd,
		sparkplug.State:
	}
	for _, m := range ev.Metrics {
		r.appendMetric(now, msg.received, scope, m, birth, b)
	}
}

// appendMetric puts the value of m into b when m is a tag of the task in scope.
// received is the time that the message of m arrived.
func (r *reader) appendMetric(
	now time.Time,
	received telem.TimeStamp,
	scope tagScope,
	m sparkplug.Metric,
	birth bool,
	b *batch,
) {
	t, ok := r.tags[tagID{tagScope: scope, name: m.Name}]
	if !ok || m.Value == nil {
		return
	}
	data, err := sparkplug.AppendSample(nil, t.dataType, m.Value)
	if err != nil {
		r.degrade(now, errors.Wrapf(err, "rejected a value of %s", t))
		return
	}
	if m.Historical {
		if err = r.history.add(now, t, m.Timestamp, data); err != nil {
			r.degrade(now, err)
		}
		return
	}
	if t.index != 0 && !r.appendIndex(now, received, t, m, birth, b) {
		return
	}
	b.append(t.channel, t.dataType, data)
}

// appendIndex puts the timestamp of m into the index of t in b. It returns false when
// it rejects a timestamp that is not after the last one written.
func (r *reader) appendIndex(
	now time.Time,
	received telem.TimeStamp,
	t readTag,
	m sparkplug.Metric,
	birth bool,
	b *batch,
) bool {
	stamp := m.Timestamp
	if stamp == 0 {
		stamp = received
	}
	// An index takes timestamps in rising order only. A birth repeats values that the
	// task may have, so a stale one is no loss.
	if stamp <= r.lastStamp[t.index] {
		if !birth {
			r.degrade(now, errors.Newf(
				"rejected a value of %s: its timestamp is not after the last one written",
				t,
			))
		}
		return false
	}
	r.lastStamp[t.index] = stamp
	if err := r.history.live(now, t.index, stamp); err != nil {
		r.degrade(now, err)
	}
	b.append(
		t.index,
		telem.TimestampT,
		telem.ByteOrder.AppendUint64(nil, uint64(stamp)),
	)
	return true
}

// reads reports whether the task has a tag in scope, or, for an edge node, in one of
// its devices.
func (r *reader) reads(scope tagScope) bool {
	for id := range r.tags {
		if scope.holds(id) {
			return true
		}
	}
	return false
}

// endDomains opens a gap for each tag in scope and returns the indexes that end a
// domain.
func (r *reader) endDomains(scope tagScope) channel.Keys {
	var ended channel.Keys
	for id, t := range r.tags {
		if scope.holds(id) && r.history.end(t.index, r.lastStamp[t.index]) {
			ended = append(ended, t.index)
		}
	}
	return ended
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
