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
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/payload"
)

// Repair returns the messages of rd suited to next, the ICD that replaces prev, and the
// changes that suit them, in order. prev names the messages and fields that next lacks;
// a migration passes the stored ICD as both. channels holds the channels of the
// bindings of rd. A read of another ICD keeps its messages. The caller disables or
// removes the entry of a read that the repair leaves with no messages.
func Repair(
	rd Read,
	prev, next icd.ICD,
	channels map[channel.Key]icdv0.Channel,
) ([]icd.ReadMessage, []icd.Change) {
	if rd.ICD != next.Key {
		return rd.Messages, nil
	}
	r := repair{prev: prev, channels: channels}
	messages := rd.Messages[:0:0]
	for _, rm := range rd.Messages {
		if m, kept := r.message(rm, next, rd.Framed); kept {
			messages = append(messages, m)
		}
	}
	return messages, r.changes
}

// RepairEntries repairs each of entries as Repair does, and removes an entry that the
// repair leaves with no messages, since an entry has no disabled flag.
func RepairEntries(
	entries []icd.ReadEntry,
	prev, next icd.ICD,
	channels map[channel.Key]icdv0.Channel,
) ([]icd.ReadEntry, []icd.Change) {
	var (
		kept    = entries[:0:0]
		changes []icd.Change
	)
	for i, rd := range Entries(entries) {
		messages, repaired := Repair(rd, prev, next, channels)
		changes = append(changes, repaired...)
		if len(messages) == 0 && len(rd.Messages) > 0 {
			changes = append(changes, icd.Change{
				Description: fmt.Sprintf(
					"The task removes an entry of ICD %s, because the entry reads no "+
						"messages now.",
					prev.Name,
				),
				Lossy: true,
			})
			continue
		}
		e := entries[i]
		e.Messages = messages
		kept = append(kept, e)
	}
	return kept, changes
}

// repair suits the messages of one read to an edit of their ICD, and holds the changes
// it makes.
type repair struct {
	prev     icd.ICD
	channels map[channel.Key]icdv0.Channel
	changes  []icd.Change
}

func (r *repair) notef(lossy bool, format string, args ...any) {
	r.changes = append(r.changes, icd.Change{
		Description: fmt.Sprintf(format, args...),
		Lossy:       lossy,
	})
}

// message returns rm suited to next, and false when the read can no longer read it. A
// message that loses its last binding goes, but one that had none stays.
func (r *repair) message(
	rm icd.ReadMessage,
	next icd.ICD,
	framed bool,
) (icd.ReadMessage, bool) {
	prev, hasPrev := messageOf(r.prev, rm.Message)
	m, ok := messageOf(next, rm.Message)
	if !ok {
		r.notef(true, "The task stops reading %s.", gone("message", prev.Name))
		return icd.ReadMessage{}, false
	}
	if _, isJSON := m.Payload.Variant.(icd.JSONPayload); !isJSON && !framed {
		r.notef(
			true,
			"The task stops reading message %s, which is no longer a JSON message.",
			m.Name,
		)
		return icd.ReadMessage{}, false
	}
	if hasPrev && prev.Payload.Format() != m.Payload.Format() && len(rm.Bindings) > 0 {
		r.notef(
			true,
			"The task stops reading message %s, which changed its format.",
			m.Name,
		)
		return icd.ReadMessage{}, false
	}
	fields := fieldsOf(m.Payload)
	var prevFields map[icd.FieldKey]field
	if hasPrev {
		prevFields = fieldsOf(prev.Payload)
	}
	out := icd.ReadMessage{
		Message: rm.Message,
		Index:   r.index(rm, m.Name, fields, prevFields),
	}
	out.Bindings = rm.Bindings[:0:0]
	for _, b := range rm.Bindings {
		// The binding of a former index field writes to an index channel.
		if out.Index == nil && rm.Index != nil && b.Field == *rm.Index {
			continue
		}
		if b, kept := r.binding(b, m.Name, fields, prevFields); kept {
			out.Bindings = append(out.Bindings, b)
		}
	}
	if len(out.Bindings) == 0 && len(rm.Bindings) > 0 {
		r.notef(
			true,
			"The task stops reading message %s, which has no bindings left.",
			m.Name,
		)
		return icd.ReadMessage{}, false
	}
	return out, true
}

// index returns the index of rm suited to fields, the fields of the message named
// message, or nil when the read stamps samples on arrival. prevFields names a field
// that fields lacks.
func (r *repair) index(
	rm icd.ReadMessage,
	message string,
	fields, prevFields map[icd.FieldKey]field,
) *icd.FieldKey {
	if rm.Index == nil {
		return nil
	}
	reason := unindexable(fields, *rm.Index)
	if reason == "" {
		return rm.Index
	}
	name := "its index field"
	if f, ok := lo.Coalesce(fields[*rm.Index], prevFields[*rm.Index]); ok {
		name = "index field " + f.name
	}
	r.notef(
		true,
		"In message %s, the task stamps samples on arrival, because %s %s.",
		message, name, reason,
	)
	return nil
}

// unindexable returns why the field key of fields cannot be the index of a read, or ""
// when it can.
func unindexable(fields map[icd.FieldKey]field, key icd.FieldKey) string {
	f, exists := fields[key]
	switch {
	case !exists:
		return "no longer exists"
	case f.kind != payload.KindTimestamp:
		return "is no longer a timestamp field"
	case f.defaulted:
		return "has a default"
	}
	return ""
}

// binding returns b suited to fields, the fields of the message named message, and
// false when the read can no longer read its field. prevFields names a field that
// fields lacks.
func (r *repair) binding(
	b icd.ReadBinding,
	message string,
	fields, prevFields map[icd.FieldKey]field,
) (icd.ReadBinding, bool) {
	f, exists := fields[b.Field]
	if !exists {
		r.notef(
			true,
			"In message %s, the task stops reading %s.%s",
			message, gone("field", prevFields[b.Field].name), r.stops(b.Channel),
		)
		return icd.ReadBinding{}, false
	}
	if f.unplaced {
		r.notef(
			true,
			"In message %s, the task stops reading bytes field %s, which has no JSON "+
				"pointer.%s",
			message, f.name, r.stops(b.Channel),
		)
		return icd.ReadBinding{}, false
	}
	if ch, ok := r.channels[b.Channel]; ok && f.check(ch.DataType, "") != nil {
		r.notef(
			true,
			"In message %s, field %s no longer reads into channel %s of data type %s. "+
				"Configure creates a new channel for it.",
			message, f.name, ch.Name, ch.DataType,
		)
		b.Channel = 0
	}
	if b.DataType != "" && f.check(b.DataType, "") != nil {
		if b.Channel == 0 {
			r.notef(
				true,
				"In message %s, field %s no longer converts to data type %s. "+
					"Configure creates a channel of its default data type.",
				message, f.name, b.DataType,
			)
		} else {
			r.notef(
				false,
				"In message %s, field %s no longer converts to data type %s, so its "+
					"binding drops that data type.",
				message, f.name, b.DataType,
			)
		}
		b.DataType = ""
	}
	return b, true
}

// stops returns the sentence that says the channel key stops getting values, or "" when
// the repair does not hold the channel.
func (r *repair) stops(key channel.Key) string {
	if ch, ok := r.channels[key]; ok {
		return fmt.Sprintf(" Channel %s stops getting its values.", ch.Name)
	}
	return ""
}

func messageOf(ic icd.ICD, key icd.MessageKey) (icd.Message, bool) {
	return lo.Find(ic.Messages, func(m icd.Message) bool { return m.Key == key })
}

// gone returns the words for a noun named name that no longer exists. An empty name is
// a noun that prev does not hold either, as in a migration.
func gone(noun, name string) string {
	if name == "" {
		return "a " + noun + " that no longer exists"
	}
	return noun + " " + name + ", which no longer exists"
}
