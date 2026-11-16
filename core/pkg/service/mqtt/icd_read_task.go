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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// icdMessage reads the documents of one ICD message into its bound channels.
type icdMessage struct {
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

// icdTopic reads one topic, whose documents are each one of several ICD messages.
type icdTopic struct {
	topicSubscription
	messages []*icdMessage
	decoder  *codec.JSON
	// samples holds, for each message, the sample of each binding in the last
	// document that message matched.
	samples [][][]byte
}

var _ topicReader = (*icdTopic)(nil)

// newICDTopic validates entry against ic, the ICD it names, and channels, which holds
// every channel that a binding of the entry names.
func newICDTopic(
	entry ICDReadEntry,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
) (*icdTopic, error) {
	if err := validateTopic(entry.Topic); err != nil {
		return nil, err
	}
	t := &icdTopic{
		topic:           entry.Topic,
		qos:             qosLevel(entry.Qos),
		retainedIgnored: entry.RetainedIgnored,
	}
	if len(entry.Messages) == 0 {
		return nil, errors.Wrapf(
			validate.ErrValidation, "topic %s reads no messages", entry.Topic,
		)
	}
	decoded := make([]codec.JSONMessage, len(entry.Messages))
	for i, rm := range entry.Messages {
		m, err := newICDMessage(rm, ic, channels, &decoded[i])
		if err != nil {
			return nil, errors.Wrapf(err, "topic %s", entry.Topic)
		}
		t.messages = append(t.messages, m)
		t.samples = append(t.samples, make([][]byte, len(decoded[i].Bindings)))
	}
	var err error
	if t.decoder, err = codec.NewJSON(decoded); err != nil {
		return nil, errors.Wrapf(err, "topic %s", entry.Topic)
	}
	return t, nil
}

// newICDMessage validates rm and fills decoded with what the decoder needs to read it.
func newICDMessage(
	rm ICDReadMessage,
	ic icd.ICD,
	channels map[channel.Key]channel.Channel,
	decoded *codec.JSONMessage,
) (*icdMessage, error) {
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
	m := &icdMessage{name: msg.Name, indexField: -1}
	bindings := make([]codec.Binding, len(rm.Bindings))
	indexSet := false
	for i, b := range rm.Bindings {
		ch, ok := channels[b.Channel]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"message %s: channel %d does not exist", msg.Name, b.Channel,
			)
		}
		bindings[i] = codec.Binding{Field: b.Field, DataType: ch.DataType}
		m.dataTypes = append(m.dataTypes, ch.DataType)
		if !indexSet {
			m.index, indexSet = ch.Index(), true
		} else if ch.Index() != m.index {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"message %s: every channel must share one index channel", msg.Name,
			)
		}
		isIndexField := rm.Index != nil && *rm.Index == b.Field
		switch {
		case isIndexField && !ch.IsIndex:
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"message %s: the index field must write to an index channel",
				msg.Name,
			)
		case !isIndexField && ch.IsIndex:
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"message %s: only the index field can write to an index channel",
				msg.Name,
			)
		case isIndexField:
			m.indexField = i
		}
	}
	if rm.Index != nil && m.indexField == -1 {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"message %s: the index field has no channel", msg.Name,
		)
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

// convert reads msg as the message whose match rule its document meets.
func (t *icdTopic) convert(msg message) (topicSample, error) {
	for _, samples := range t.samples {
		for i := range samples {
			samples[i] = samples[i][:0]
		}
	}
	i, ok, err := t.decoder.Decode(msg.payload, t.samples)
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
func (m *icdMessage) sample(samples [][]byte, received telem.TimeStamp) topicSample {
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

func (f *Factory) retrieveICD(ctx context.Context, key icd.Key) (icd.ICD, error) {
	var ic icd.ICD
	err := f.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
		return f.cfg.ICD.NewRetrieve().
			Where(icd.MatchKeys(key)).
			Entry(&ic).
			Exec(ctx, snap)
	})
	if errors.Is(err, query.ErrNotFound) {
		return icd.ICD{}, errors.Wrapf(
			validate.ErrValidation, "icd: ICD %s does not exist", key,
		)
	}
	return ic, err
}

func (f *Factory) configureICDRead(
	ctx context.Context,
	t task.Task,
) (autoStarter, bool, error) {
	var cfg ICDReadConfig
	if err := t.Config.Unmarshal(&cfg); err != nil {
		return nil, false, err
	}
	dev, err := f.retrieveDevice(ctx, cfg.Device)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	var (
		enabled []ICDReadEntry
		keys    channel.Keys
	)
	for _, e := range cfg.Entries {
		if e.Disabled {
			continue
		}
		for _, m := range e.Messages {
			for _, b := range m.Bindings {
				keys = append(keys, b.Channel)
			}
		}
		enabled = append(enabled, e)
	}
	if len(enabled) == 0 {
		return nil, cfg.AutoStart, errors.Wrap(
			validate.ErrValidation, "entries: the task has no enabled entries",
		)
	}
	channels, err := f.retrieveChannels(ctx, keys)
	if err != nil {
		return nil, cfg.AutoStart, err
	}
	src := &readSource{
		pool:       f.pool,
		broker:     dev.Key,
		queueSize:  f.cfg.QueueSize,
		birthGrace: f.cfg.BirthGrace,
		topics:     make(map[string]topicReader),
		tags:       make(map[tagID]readTag),
		lastStamp:  make(map[channel.Key]telem.TimeStamp),
		framer:     f.cfg.Framer,
		subject:    control.Subject{Key: t.Key.String(), Name: t.Name},
		ins:        f.cfg.Child(t.Key.String()),
	}
	var (
		// written names the message that writes to each channel.
		written    = make(map[channel.Key]string)
		writerKeys channel.Keys
		indexes    channel.Keys
		icds       = make(map[icd.Key]icd.ICD)
	)
	for _, entry := range enabled {
		ic, ok := icds[entry.ICD]
		if !ok {
			if ic, err = f.retrieveICD(ctx, entry.ICD); err != nil {
				return nil, cfg.AutoStart, err
			}
			icds[entry.ICD] = ic
		}
		topic, err := newICDTopic(entry, ic, channels)
		if err != nil {
			return nil, cfg.AutoStart, err
		}
		if _, ok := src.topics[topic.topic]; ok {
			return nil, cfg.AutoStart, errors.Wrapf(
				validate.ErrValidation,
				"entries: topic %s appears more than once", topic.topic,
			)
		}
		src.topics[topic.topic] = topic
		// Two messages on one index would race for the rising order of its timestamps,
		// or write frames that lack the channels of the other.
		for _, m := range topic.messages {
			name := "message " + m.name + " of topic " + topic.topic
			for _, key := range m.keys {
				if other, ok := written[key]; ok {
					return nil, cfg.AutoStart, errors.Wrapf(
						validate.ErrValidation,
						"entries: %s and %s write to the same channel %d",
						other, name, key,
					)
				}
				written[key] = name
				writerKeys = append(writerKeys, key)
			}
			if m.index != 0 {
				indexes = append(indexes, m.index)
			}
		}
	}
	rt, err := f.openRead(t, src, writerKeys, indexes, cfg.DataSavingDisabled)
	return rt, cfg.AutoStart, err
}
