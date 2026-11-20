// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// Channel is the part of a channel that a read field must agree with.
type Channel struct {
	// Name is the name of the channel.
	Name string
	// DataType is the data type of the channel.
	DataType telem.DataType
	// Index is true when the channel is an index channel.
	Index bool
}

// Channels returns the channels among keys, read through snap. A key with no channel is
// absent from the result.
type Channels func(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]Channel, error)

// ResolveReadConfig moves the fields of the inline records of cfg into a new ICD named
// name, with one JSON message for each record, named after its record key, and adds
// each record, read from its message, to the topic of cfg named by the inline topic. It
// adds that topic when cfg lacks it. It returns the ICD, which the caller must create,
// and false when cfg holds no inline records. A field takes the kind of its channel,
// read through snap with channels, and binds no channel when that channel does not
// exist. A disabled field gets no binding, and a record with no binding is disabled.
// note receives the path and each change of a field.
func ResolveReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	name string,
	cfg *ReadConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	staged := cfg.Inline.Records
	found, err := inlineChannels(ctx, snap, channels, staged)
	if err != nil {
		return icd.ICD{}, false, err
	}
	ic := icd.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icd.Message, len(staged)),
	}
	t := slices.IndexFunc(cfg.Topics, func(topic ReadTopic) bool {
		return topic.Name == cfg.Inline.Topic
	})
	if t < 0 {
		t = len(cfg.Topics)
		cfg.Topics = append(cfg.Topics, ReadTopic{
			Key:     uuid.New().String(),
			Name:    cfg.Inline.Topic,
			Records: []ReadRecord{},
		})
	}
	names := make(set.Set[string])
	for i, r := range staged {
		msg, read := liftRecord(r, found, "records."+strconv.Itoa(i), note)
		msg.Name = uniqueName(names, lo.CoalesceOrEmpty(r.RecordKey, "Any key"))
		ic.Messages[i] = msg
		cfg.Topics[t].Records = append(cfg.Topics[t].Records, ReadRecord{
			Key:       lo.CoalesceOrEmpty(r.Key, uuid.New().String()),
			Disabled:  len(read.Bindings) == 0,
			RecordKey: r.RecordKey,
			ICD:       ic.Key,
			Messages:  []ReadMessage{read},
		})
	}
	cfg.Inline = nil
	return ic, true, nil
}

// liftRecord returns the message that holds the fields of r, with no name, and the read
// of that message. path is the path of r.
func liftRecord(
	r InlineRecord,
	found map[channel.Key]Channel,
	path string,
	note func(path, change string),
) (icd.Message, ReadMessage) {
	var (
		fields = make([]icd.JSONField, len(r.Fields))
		read   = ReadMessage{Message: uuid.New(), Bindings: []ReadBinding{}}
		names  = make(set.Set[string])
	)
	for i, old := range r.Fields {
		var ch *Channel
		if c, exists := found[old.Channel]; exists {
			ch = &c
		}
		isIndex := r.Index != "" && old.Key == r.Index
		key := uuid.New()
		field, dataType, changes := liftField(old, isIndex, ch, icd.JSONBaseField{
			Key:     key,
			Name:    uniqueName(names, fieldName(old)),
			Scale:   1,
			Pointer: old.Pointer,
		})
		for _, c := range changes {
			note(path+".fields."+strconv.Itoa(i), c)
		}
		fields[i] = field
		if isIndex {
			read.Index = &key
		}
		if old.Disabled {
			continue
		}
		b := ReadBinding{Field: key}
		if ch != nil {
			b.Channel = old.Channel
		} else if dataType != "" {
			b.DataType = dataType
		}
		read.Bindings = append(read.Bindings, b)
	}
	if r.Index != "" && read.Index == nil {
		note(path+".index", "index field "+r.Index+" does not exist and was dropped")
	}
	return icd.Message{
		Key:     read.Message,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}, read
}

// fieldName returns the name of old, or else its pointer.
func fieldName(old ReadField) string {
	if old.Name != "" {
		return old.Name
	}
	return lo.CoalesceOrEmpty(strings.TrimPrefix(old.Pointer, "/"), "value")
}

// uniqueName returns name, or name with the lowest suffix " (n)" that taken lacks, and
// adds the result to taken.
func uniqueName(taken set.Set[string], name string) string {
	unique := name
	for n := 2; taken.Contains(unique); n++ {
		unique = name + " (" + strconv.Itoa(n) + ")"
	}
	taken.Add(unique)
	return unique
}

func inlineChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	staged []InlineRecord,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, r := range staged {
		for _, f := range r.Fields {
			if f.Channel != 0 {
				keys = append(keys, f.Channel)
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, snap, lo.Uniq(keys))
}
