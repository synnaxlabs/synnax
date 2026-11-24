// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"slices"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/set"
)

// ResolveReadConfig moves the fields of the inline entries of cfg into a new ICD named
// name, with one JSON message for each entry, named after its topic, and adds each
// entry back to the entries of cfg, read from its message. It returns the ICD, which
// the caller must lift and create, and false when cfg holds no inline entries. A field
// takes the kind of its channel, read through snap with channels, and binds no channel
// when that channel does not exist. A disabled field gets no binding. note receives the
// path and each change of a field.
func ResolveReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels icdv0.Channels,
	name string,
	cfg *ReadConfig,
	note func(path, change string),
) (icdv0.ICD, bool, error) {
	if cfg.Inline == nil {
		return icdv0.ICD{}, false, nil
	}
	staged := cfg.Inline.Entries
	found, err := channels.Of(
		ctx,
		snap,
		lo.FlatMap(staged, func(e InlineEntry, _ int) []icdv0.InlineField {
			return lo.Map(e.Fields, toInline)
		}),
	)
	if err != nil {
		return icdv0.ICD{}, false, err
	}
	ic := icdv0.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icdv0.Message, len(staged)),
	}
	names := make(set.Set[string])
	for i, e := range staged {
		msg, read := icdv0.LiftMessage(
			lo.Map(e.Fields, toInline),
			e.Index,
			found,
			"entries."+strconv.Itoa(int(e.At)),
			note,
		)
		msg.Name = icdv0.UniqueName(names, lo.CoalesceOrEmpty(e.Topic, "Untitled"))
		ic.Messages[i] = msg
		lifted, err := icd.MigrateReadMessage(ctx, read)
		if err != nil {
			return icdv0.ICD{}, false, err
		}
		cfg.Entries = slices.Insert(
			cfg.Entries,
			min(int(e.At), len(cfg.Entries)),
			ReadEntry{Variant: PlainReadEntry{
				BaseReadEntry:   e.BaseReadEntry,
				Topic:           e.Topic,
				Qos:             e.Qos,
				RetainedIgnored: e.RetainedIgnored,
				ICD:             ic.Key,
				Messages:        []icd.ReadMessage{lifted},
			}},
		)
	}
	cfg.Inline = nil
	return ic, true, nil
}

func toInline(f ReadField, _ int) icdv0.InlineField {
	return icdv0.InlineField{
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
