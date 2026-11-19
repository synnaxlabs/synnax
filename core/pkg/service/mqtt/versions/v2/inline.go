// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

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

// Channels returns the channels among keys, read through snap. A key with no channel
// is absent from the result.
type Channels func(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]Channel, error)

// ResolveReadConfig moves the fields of the inline entries of cfg into a new ICD named
// name, with one JSON message for each entry, named after its topic, and adds each
// entry back to the entries of cfg, read from its message. It returns the ICD, which
// the caller must create, and false when cfg holds no inline entries. A field takes the
// kind of its channel, read through snap with channels, and binds no channel when that
// channel does not exist. A disabled field gets no binding. note receives the path and
// each change of a field.
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
	staged := cfg.Inline.Entries
	found, err := inlineChannels(ctx, snap, channels, staged)
	if err != nil {
		return icd.ICD{}, false, err
	}
	ic := icd.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icd.Message, len(staged)),
	}
	names := make(set.Set[string])
	for i, e := range staged {
		msg, read := liftEntry(e, found, "entries."+strconv.Itoa(int(e.At)), note)
		msg.Name = uniqueName(names, lo.CoalesceOrEmpty(e.Topic, "Untitled"))
		ic.Messages[i] = msg
		cfg.Entries = slices.Insert(
			cfg.Entries,
			min(int(e.At), len(cfg.Entries)),
			ReadEntry{Variant: PlainReadEntry{
				BaseReadEntry:   e.BaseReadEntry,
				Topic:           e.Topic,
				Qos:             e.Qos,
				RetainedIgnored: e.RetainedIgnored,
				ICD:             ic.Key,
				Messages:        []icd.ReadMessage{read},
			}},
		)
	}
	cfg.Inline = nil
	return ic, true, nil
}

// liftEntry returns the message that holds the fields of e, with no name, and the read
// of that message. path is the path of e.
func liftEntry(
	e InlineEntry,
	found map[channel.Key]Channel,
	path string,
	note func(path, change string),
) (icd.Message, icd.ReadMessage) {
	var (
		fields = make([]icd.JSONField, len(e.Fields))
		read   = icd.ReadMessage{Message: uuid.New(), Bindings: []icd.ReadBinding{}}
		names  = make(set.Set[string])
	)
	for i, old := range e.Fields {
		var ch *Channel
		if c, exists := found[old.Channel]; exists {
			ch = &c
		}
		isIndex := e.Index != "" && old.Key == e.Index
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
		b := icd.ReadBinding{Field: key}
		if ch != nil {
			b.Channel = old.Channel
		} else if dataType != "" {
			b.DataType = dataType
		}
		read.Bindings = append(read.Bindings, b)
	}
	if e.Index != "" && read.Index == nil {
		note(path+".index", "index field "+e.Index+" does not exist and was dropped")
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
	staged []InlineEntry,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, e := range staged {
		for _, f := range e.Fields {
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
