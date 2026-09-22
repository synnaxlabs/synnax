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
	"encoding/json"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// tagScope is an edge node, or one device of it when device is not empty.
type tagScope struct {
	node   sparkplug.NodeID
	device string
}

func (s tagScope) String() string {
	if s.device == "" {
		return "edge node " + s.node.String()
	}
	return "device " + s.node.String() + "/" + s.device
}

func (s tagScope) validate() error {
	if err := s.node.Validate(); err != nil {
		return err
	}
	if s.device == "" {
		return nil
	}
	return sparkplug.ValidateID("device", s.device)
}

// tagID identifies one Sparkplug B tag on a broker.
type tagID struct {
	tagScope
	name string
}

func (t tagID) String() string {
	id := t.node.String()
	if t.device != "" {
		id += "/" + t.device
	}
	return "tag " + id + "/" + t.name
}

func (t tagID) validate() error {
	if err := t.tagScope.validate(); err != nil {
		return err
	}
	// A tag name is not a topic level, so it can hold a slash.
	if t.name == "" {
		return errors.Wrap(validate.ErrValidation, "tag: required")
	}
	return nil
}

// readTag is one Sparkplug B read entry, ready to convert values into samples.
type readTag struct {
	tagID
	channel channel.Key
	// index is the index channel of channel. Zero for a virtual channel.
	index    channel.Key
	dataType telem.DataType
}

// newReadTag validates entry against channels, which holds its channel.
func newReadTag(
	entry SparkplugReadEntry,
	channels map[channel.Key]channel.Channel,
) (readTag, error) {
	t := readTag{
		node:    sparkplug.NodeID{Group: entry.Group, EdgeNode: entry.EdgeNode},
		device:  entry.Device,
		name:    entry.Tag,
		channel: entry.Channel,
	}
	if err := t.validate(); err != nil {
		return t, err
	}
	ch, ok := channels[entry.Channel]
	if !ok {
		return t, errors.Wrapf(
			validate.ErrValidation, "%s: channel %d does not exist", t, entry.Channel,
		)
	}
	if ch.IsIndex || !xjson.SupportsSampleTarget(ch.DataType) {
		return t, errors.Wrapf(
			validate.ErrValidation,
			"%s: channel %s cannot take the values of a tag", t, ch.Name,
		)
	}
	t.index, t.dataType = ch.Index(), ch.DataType
	return t, nil
}

func (t readTag) keys() channel.Keys {
	if t.index == 0 {
		return channel.Keys{t.channel}
	}
	return channel.Keys{t.index, t.channel}
}

// appendSample converts the value of a tag to one sample of the channel of t.
func (t readTag) appendSample(dst []byte, value any) ([]byte, error) {
	format := xjson.ISO8601
	switch v := value.(type) {
	case int64:
		value = json.Number(strconv.FormatInt(v, 10))
	case uint64:
		value = json.Number(strconv.FormatUint(v, 10))
	case telem.TimeStamp:
		value, format = json.Number(
			strconv.FormatInt(int64(v), 10),
		), xjson.UnixNanosecond
		if t.dataType != telem.TimestampT {
			// A DateTime counts milliseconds.
			value = json.Number(strconv.FormatInt(int64(v/telem.MillisecondTS), 10))
		}
	}
	return xjson.AppendSample(dst, t.dataType, value, format, nil)
}

// sparkplugTypes maps the config form of a Sparkplug B data type to its wire form.
var sparkplugTypes = map[SparkplugDataType]sparkplug.DataType{
	SparkplugDataTypeInt8:     sparkplug.Int8,
	SparkplugDataTypeInt16:    sparkplug.Int16,
	SparkplugDataTypeInt32:    sparkplug.Int32,
	SparkplugDataTypeInt64:    sparkplug.Int64,
	SparkplugDataTypeUInt8:    sparkplug.UInt8,
	SparkplugDataTypeUInt16:   sparkplug.UInt16,
	SparkplugDataTypeUInt32:   sparkplug.UInt32,
	SparkplugDataTypeUInt64:   sparkplug.UInt64,
	SparkplugDataTypeFloat:    sparkplug.Float,
	SparkplugDataTypeDouble:   sparkplug.Double,
	SparkplugDataTypeBoolean:  sparkplug.Boolean,
	SparkplugDataTypeString:   sparkplug.String,
	SparkplugDataTypeDateTime: sparkplug.DateTime,
}

// commandTarget is one Sparkplug B write target: it sends the samples of a command
// channel as commands for one tag.
type commandTarget struct {
	tagID
	topic         string
	dataType      telem.DataType
	sparkplugType sparkplug.DataType
}

var _ target = commandTarget{}

// newCommandTarget validates cfg against ch, its command channel.
func newCommandTarget(
	cfg SparkplugWriteTarget,
	ch channel.Channel,
) (commandTarget, error) {
	t := commandTarget{
		node:     sparkplug.NodeID{Group: cfg.Group, EdgeNode: cfg.EdgeNode},
		device:   cfg.Device,
		name:     cfg.Tag,
		dataType: ch.DataType,
	}
	if err := t.validate(); err != nil {
		return t, err
	}
	t.topic = sparkplug.CommandTopic(t.node, t.device).String()
	var ok bool
	if t.sparkplugType, ok = sparkplugTypes[cfg.SparkplugType]; !ok {
		return t, errors.Wrapf(
			validate.ErrValidation,
			"%s: unknown Sparkplug B data type %s", t, cfg.SparkplugType,
		)
	}
	// A channel and a type that never match fail now, not on the first command.
	probe := telem.MakeSeries(ch.DataType, 1)
	if ch.DataType.IsVariable() {
		probe = telem.NewSeriesV("")
	}
	if _, err := t.publication(probe, 0); err != nil {
		return t, errors.Wrapf(
			validate.ErrValidation,
			"%s: channel %s of type %s cannot be sent as %s",
			t, ch.Name, ch.DataType, t.sparkplugType,
		)
	}
	return t, nil
}

// sampleValue returns sample i of series in a form that a Sparkplug B metric takes.
func sampleValue(dataType telem.DataType, series telem.Series, i int) (any, error) {
	switch dataType {
	case telem.TimestampT:
		return series.ValueAt[telem.TimeStamp](i), nil
	case telem.StringT:
		return string(series.At(i)), nil
	}
	return xjson.FromSample(dataType, series.At(i), xjson.Number, nil)
}

// publication implements target.
func (t commandTarget) publication(series telem.Series, i int) (publication, error) {
	value, err := sampleValue(t.dataType, series, i)
	if err != nil {
		return publication{}, err
	}
	payload, err := sparkplug.EncodeCommand(
		sparkplug.Metric{Name: t.name, DataType: t.sparkplugType, Value: value},
		telem.Now(),
	)
	// Sparkplug B sends every command at quality of service 0 and not retained.
	return publication{topic: t.topic, payload: payload}, err
}
