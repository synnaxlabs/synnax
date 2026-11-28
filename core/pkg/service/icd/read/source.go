// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package read

import (
	"bytes"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Group is the channels of a read that a document writes or drops together: the
// channels on one index channel, or one virtual channel.
type Group struct {
	// Read is the position of the read message of the group in the messages of its
	// Read.
	Read int
	// Message is the name of the ICD message of the read.
	Message string
	// Index is the index channel of the group, or zero for a virtual channel.
	Index channel.Key
	// Stamped is true when the index field of the read gives the timestamps of Index.
	// Else they are the arrival time of each document.
	Stamped bool
	// Channels are the data channels of the group, in binding order.
	Channels channel.Keys
	// DataTypes holds the data type of each channel.
	DataTypes []telem.DataType
}

// Keys returns the channels that the group writes: Index first, when it is not zero,
// and then Channels.
func (g Group) Keys() channel.Keys {
	keys := make(channel.Keys, 0, len(g.Channels)+1)
	if g.Index != 0 {
		keys = append(keys, g.Index)
	}
	return append(keys, g.Channels...)
}

// Sample is the samples that one payload gives a group.
type Sample struct {
	// Group is the position of the group in Groups.
	Group int
	// Stamp is the timestamp of the index field, or zero when the group is not stamped.
	Stamp telem.TimeStamp
	// Data holds one sample for each channel of the group.
	Data [][]byte
}

// Source decodes the payloads of one source, such as an MQTT topic, as reads of ICD
// messages of one format. A field that does not decode drops only its group, and the
// other groups of the payload still write. Source is not safe for concurrent use.
type Source struct {
	format  payload.Format
	decoder decoder
	groups  []group
	reads   []sourceRead
	// samples holds, for each read, the sample of each binding in the last payload.
	samples [][][]byte
	// last is the read of the last payload, whose samples the next Decode clears, or -1
	// before the first.
	last    int
	dropped []bool
	out     []Sample
}

// decoder decodes payloads that each belong to one of several messages into samples, as
// codec.JSON does.
type decoder interface {
	Decode(
		data []byte,
		samples [][][]byte,
	) (message int, ok bool, failures []codec.Failure, err error)
}

// text is a codec.Text decoder of payloads that each hold one line, with or without its
// line end. A line never fails as a whole.
type text struct{ codec *codec.Text }

func (t text) Decode(
	line []byte,
	samples [][][]byte,
) (int, bool, []codec.Failure, error) {
	line = bytes.TrimSuffix(line, []byte("\n"))
	message, ok, failures := t.codec.Decode(line, samples)
	return message, ok, failures, nil
}

type group struct {
	Group
	// stamp is the position in the bindings of the index field, or -1.
	stamp int
	// bindings holds the position in the bindings of each channel of the group.
	bindings []int
	data     [][]byte
}

type sourceRead struct {
	// groups holds the position in Source.groups of the group of each binding.
	groups []int
	// first and end bound the positions in Source.groups of the groups of the read.
	first, end int
}

// newSource returns the Source of rd, a read of ic that passes Checker.Read. channels
// holds the channels of the bindings. A field on an index channel and the fields on its
// data channels make one group. Each field on a virtual channel is a group of its own.
func newSource(
	rd Read,
	ic icd.ICD,
	channels map[channel.Key]icdv0.Channel,
) (*Source, error) {
	s := &Source{
		reads:   make([]sourceRead, len(rd.Messages)),
		samples: make([][][]byte, len(rd.Messages)),
		last:    -1,
	}
	var (
		jsonMessages   []codec.JSONMessage
		textMessages   []codec.TextMessage
		binaryMessages []codec.BinaryMessage
	)
	for r, rm := range rd.Messages {
		path := slices.Concat(rd.Path, []string{"messages", strconv.Itoa(r)})
		i := slices.IndexFunc(ic.Messages, func(m icd.Message) bool {
			return m.Key == rm.Message
		})
		if i < 0 {
			panic(errors.Newf("read: message %s is not in ICD %s", rm.Message, ic.Name))
		}
		msg := ic.Messages[i]
		s.format = msg.Payload.Format()
		if len(rm.Bindings) == 0 {
			return nil, validate.PathedError(
				validate.Errorf("message %s binds no fields", msg.Name),
				path...,
			)
		}
		bindings, err := s.add(r, rm, msg, channels)
		if err != nil {
			return nil, validate.PathedError(err, path...)
		}
		switch p := msg.Payload.Variant.(type) {
		case icd.JSONPayload:
			jsonMessages = append(jsonMessages, codec.JSONMessage{
				Name: msg.Name, Payload: p, Bindings: bindings,
			})
		case icd.TextPayload:
			textMessages = append(textMessages, codec.TextMessage{
				Name: msg.Name, Payload: p, Bindings: bindings,
			})
		case icd.BinaryPayload:
			binaryMessages = append(binaryMessages, codec.BinaryMessage{
				Name: msg.Name, Payload: p, Bindings: bindings,
			})
		default:
			panic(errors.Newf("read: unknown payload %T", p))
		}
		s.samples[r] = make([][]byte, len(rm.Bindings))
	}
	most := 0
	for _, rd := range s.reads {
		most = max(most, rd.end-rd.first)
	}
	s.dropped = make([]bool, most)
	var err error
	switch s.format {
	case payload.FormatJSON:
		var j *codec.JSON
		if j, err = codec.NewJSON(jsonMessages); err == nil {
			s.decoder = j
		}
	case payload.FormatText:
		var t *codec.Text
		if t, err = codec.NewText(textMessages); err == nil {
			s.decoder = text{codec: t}
		}
	case payload.FormatBinary:
		var b *codec.Binary
		if b, err = codec.NewBinary(binaryMessages); err == nil {
			s.decoder = b
		}
	}
	if err != nil {
		return nil, validate.PathedError(err, rd.Path...)
	}
	return s, nil
}

// add groups the bindings of rm, read r of msg, and returns the codec binding of each.
// It returns a validation error at the path of a binding that has no channel, or whose
// channel does not exist.
func (s *Source) add(
	r int,
	rm icd.ReadMessage,
	msg icd.Message,
	channels map[channel.Key]icdv0.Channel,
) ([]codec.Binding, error) {
	rd := sourceRead{groups: make([]int, len(rm.Bindings)), first: len(s.groups)}
	bindings := make([]codec.Binding, len(rm.Bindings))
	for i, b := range rm.Bindings {
		at := []string{"bindings", strconv.Itoa(i), "channel"}
		if b.Channel == 0 {
			return nil, validate.PathedError(validate.Errorf(
				"field %s has no channel", fieldsOf(msg.Payload)[b.Field].name,
			), at...)
		}
		ch, ok := channels[b.Channel]
		if !ok {
			return nil, validate.PathedError(
				validate.Errorf("channel %d does not exist", b.Channel),
				at...,
			)
		}
		bindings[i] = codec.Binding{Field: b.Field, DataType: ch.DataType}
		isIndexField := rm.Index != nil && *rm.Index == b.Field
		index := ch.IndexKey
		if isIndexField {
			index = b.Channel
		}
		g := -1
		if index != 0 {
			if k := slices.IndexFunc(s.groups[rd.first:], func(g group) bool {
				return g.Index == index
			}); k >= 0 {
				g = rd.first + k
			}
		}
		if g < 0 {
			g = len(s.groups)
			s.groups = append(
				s.groups,
				group{Read: r, Message: msg.Name, Index: index, stamp: -1},
			)
		}
		rd.groups[i] = g
		grp := &s.groups[g]
		if isIndexField {
			grp.Stamped, grp.stamp = true, i
			continue
		}
		grp.Channels = append(grp.Channels, b.Channel)
		grp.DataTypes = append(grp.DataTypes, ch.DataType)
		grp.bindings = append(grp.bindings, i)
	}
	rd.end = len(s.groups)
	for g := rd.first; g < rd.end; g++ {
		s.groups[g].data = make([][]byte, len(s.groups[g].bindings))
	}
	s.reads[r] = rd
	return bindings, nil
}

// Groups returns the groups of every read, in read order and then binding order.
func (s *Source) Groups() []Group {
	return lo.Map(s.groups, func(g group, _ int) Group { return g.Group })
}

// Decode decodes data, one payload, as the read whose match rule it meets, and returns
// a sample for each group of that read whose fields all decode. ok is false when data
// meets no match rule. err is not nil when data does not decode as a whole, such as a
// document that is not JSON, meets two match rules, or holds a field that does not
// decode, which err names. samples can hold the other groups of the payload along with
// that err. samples is valid until the next Decode.
func (s *Source) Decode(data []byte) (samples []Sample, ok bool, err error) {
	if s.last >= 0 {
		for i := range s.samples[s.last] {
			s.samples[s.last][i] = s.samples[s.last][i][:0]
		}
	}
	r, ok, failures, err := s.decoder.Decode(data, s.samples)
	if err != nil || !ok {
		return nil, ok, err
	}
	s.last = r
	rd := s.reads[r]
	dropped := s.dropped[:rd.end-rd.first]
	clear(dropped)
	for _, f := range failures {
		dropped[rd.groups[f.Binding]-rd.first] = true
	}
	s.out = s.out[:0]
	for g := rd.first; g < rd.end; g++ {
		if dropped[g-rd.first] {
			continue
		}
		grp := &s.groups[g]
		// A binary payload holds no sample for a field that its multiplexors do not
		// select, and a group writes only when it has each of its samples.
		unselected := false
		for j, b := range grp.bindings {
			grp.data[j] = s.samples[r][b]
			unselected = unselected || len(grp.data[j]) == 0
		}
		if unselected {
			continue
		}
		sample := Sample{Group: g, Data: grp.data}
		if grp.stamp >= 0 {
			sample.Stamp = telem.TimeStamp(
				telem.ByteOrder.Uint64(s.samples[r][grp.stamp]),
			)
		}
		s.out = append(s.out, sample)
	}
	if len(failures) > 0 {
		err = failures[0].Err
	}
	return s.out, true, err
}
