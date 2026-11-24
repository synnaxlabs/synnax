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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/set"
)

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
	channels icd.Channels,
	name string,
	cfg *ReadConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	staged := cfg.Inline.Records
	found, err := channels.Of(
		ctx,
		snap,
		lo.FlatMap(staged, func(r InlineRecord, _ int) []icd.InlineField {
			return lo.Map(r.Fields, toInline)
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
		msg, read := icd.LiftMessage(
			lo.Map(r.Fields, toInline),
			r.Index,
			found,
			"records."+strconv.Itoa(i),
			note,
		)
		msg.Name = icd.UniqueName(names, lo.CoalesceOrEmpty(r.RecordKey, "Any key"))
		ic.Messages[i] = msg
		cfg.Topics[t].Records = append(cfg.Topics[t].Records, ReadRecord{
			Key:       lo.CoalesceOrEmpty(r.Key, uuid.New().String()),
			Disabled:  len(read.Bindings) == 0,
			RecordKey: r.RecordKey,
			ICD:       ic.Key,
			Messages:  []icd.ReadMessage{read},
		})
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
