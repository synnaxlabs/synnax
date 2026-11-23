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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/set"
)

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
	channels icd.Channels,
	name string,
	cfg *ReadConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	staged := cfg.Inline.Entries
	found, err := channels.Of(
		ctx,
		snap,
		lo.FlatMap(staged, func(e InlineEntry, _ int) []icd.InlineField {
			return lo.Map(e.Fields, toInline)
		}),
	)
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
		msg, read := icd.LiftMessage(
			lo.Map(e.Fields, toInline),
			e.Index,
			found,
			"entries."+strconv.Itoa(int(e.At)),
			note,
		)
		msg.Name = icd.UniqueName(names, lo.CoalesceOrEmpty(e.Topic, "Untitled"))
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

func toInline(f ReadField, _ int) icd.InlineField {
	return icd.InlineField{
		Key:        f.Key,
		Name:       f.Name,
		Disabled:   f.Disabled,
		Channel:    f.Channel,
		Pointer:    f.Pointer,
		DataType:   f.DataType,
		TimeFormat: f.TimeFormat,
		EnumValues: f.EnumValues,
	}
}
