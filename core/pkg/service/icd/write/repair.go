// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package write

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Repair returns the message of w suited to next, the ICD that replaces prev, and the
// changes that suit it, in order. prev names the messages and fields that next lacks; a
// migration passes the stored ICD as both. channels holds the channels of the bindings
// of w. Repair returns nil when w can no longer send its message: next lacks it, it is
// not JSON on an unframed write, or it has no channel binding left. A write of another
// ICD keeps its message.
func Repair(
	w Write,
	prev, next icd.ICD,
	channels map[channel.Key]icdv0.Channel,
) (*icd.WriteMessage, []icd.Change) {
	if w.ICD != next.Key {
		return &w.Message, nil
	}
	r := repair{prev: prev, channels: channels}
	wm, ok := r.message(w.Message, next, w.Framed)
	if !ok {
		return nil, r.changes
	}
	return &wm, r.changes
}

// RepairTargets repairs each of targets as Repair does, and removes a target that can
// no longer send its message.
func RepairTargets(
	targets []icd.WriteTarget,
	prev, next icd.ICD,
	channels map[channel.Key]icdv0.Channel,
) ([]icd.WriteTarget, []icd.Change) {
	var (
		kept    = targets[:0:0]
		changes []icd.Change
	)
	for i, w := range Targets(targets) {
		wm, repaired := Repair(w, prev, next, channels)
		changes = append(changes, repaired...)
		if wm == nil {
			continue
		}
		t := targets[i]
		t.Message = *wm
		kept = append(kept, t)
	}
	return kept, changes
}

// repair suits the message of one write to an edit of its ICD, and holds the changes it
// makes.
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

// message returns wm suited to next, and false when the write can no longer send it.
func (r *repair) message(
	wm icd.WriteMessage,
	next icd.ICD,
	framed bool,
) (icd.WriteMessage, bool) {
	prev, hasPrev := messageOf(r.prev, wm.Message)
	m, ok := messageOf(next, wm.Message)
	if !ok {
		r.notef(true, "The task stops sending %s.", gone("message", prev.Name))
		return icd.WriteMessage{}, false
	}
	if _, isJSON := m.Payload.Variant.(icd.JSONPayload); !isJSON && !framed {
		r.notef(
			true,
			"The task stops sending message %s, which is no longer a JSON message.",
			m.Name,
		)
		return icd.WriteMessage{}, false
	}
	bindings := wm.Bindings
	names := make(map[icd.FieldKey]string)
	if hasPrev {
		for _, f := range fieldsOf(prev.Payload) {
			names[f.key] = f.name
		}
		if prev.Payload.Format() != m.Payload.Format() && len(bindings) > 0 {
			r.notef(
				true,
				"Message %s changed its format, so the task drops its bindings.",
				m.Name,
			)
			bindings = nil
		}
	}
	fields := fieldsOf(m.Payload)
	at := fieldIndexes(fields)
	out := icd.WriteMessage{Message: wm.Message, Bindings: bindings[:0:0]}
	// valued holds the payload with each value as the default of its field, which can
	// fix a multiplexor.
	valued := m.Payload
	for _, b := range bindings {
		if b, kept := r.binding(b, m.Name, &valued, fields, at, names); kept {
			out.Bindings = append(out.Bindings, b)
		}
	}
	fields = fieldsOf(valued)
	out.Bindings = r.unselect(out.Bindings, m.Name, fields, at)
	out.Bindings = r.fill(out.Bindings, m.Name, fields, at)
	if len(ChannelBindings(wm)) > 0 && len(ChannelBindings(out)) == 0 {
		r.notef(
			true,
			"The task stops sending message %s, which has no channel bindings left.",
			m.Name,
		)
		return icd.WriteMessage{}, false
	}
	return out, true
}

// binding returns b suited to fields, the fields of p in the message named message, and
// false when the write can no longer send its field. A value becomes the default of its
// field in p. names holds the name of each field before the edit.
func (r *repair) binding(
	b icd.WriteBinding,
	message string,
	p *icd.Payload,
	fields []field,
	at map[icd.FieldKey]int,
	names map[icd.FieldKey]string,
) (icd.WriteBinding, bool) {
	key := b.Base().Field
	i, exists := at[key]
	if !exists {
		r.notef(
			true,
			"In message %s, the task drops the binding of %s.",
			message, gone("field", names[key]),
		)
		return icd.WriteBinding{}, false
	}
	f := fields[i]
	if f.identifier {
		r.notef(
			true,
			"In message %s, field %s now holds the identifier, so the task drops its "+
				"binding.",
			message, f.name,
		)
		return icd.WriteBinding{}, false
	}
	switch v := b.Variant.(type) {
	case icd.ValueWriteBinding:
		valued, err := withFieldValue(*p, i, v.Value)
		if err != nil {
			r.notef(
				true,
				"In message %s, field %s can no longer take its value, so the task "+
					"drops the value.",
				message, f.name,
			)
			return icd.WriteBinding{}, false
		}
		*p = valued
	case icd.ChannelWriteBinding:
		b.Variant = r.channel(v, message, f)
	default:
		panic(errors.Newf("write: unknown write binding variant %T", v))
	}
	return b, true
}

// channel returns b suited to f, the field of b in the message named message.
func (r *repair) channel(
	b icd.ChannelWriteBinding,
	message string,
	f field,
) icd.ChannelWriteBinding {
	if ch, ok := r.channels[b.Channel]; ok &&
		codec.CheckWrite(f.kind, f.scaled, ch.DataType) != nil {
		r.notef(
			true,
			"In message %s, channel %s of data type %s can no longer set field %s. "+
				"Configure creates a new channel for it.",
			message, ch.Name, ch.DataType, f.name,
		)
		b.Channel = 0
	}
	if b.DataType != nil && codec.CheckWrite(f.kind, f.scaled, *b.DataType) != nil {
		if b.Channel == 0 {
			r.notef(
				true,
				"In message %s, data type %s can no longer set field %s. Configure "+
					"creates a channel of its default data type.",
				message, *b.DataType, f.name,
			)
		} else {
			r.notef(
				false,
				"In message %s, data type %s can no longer set field %s, so its "+
					"binding drops that data type.",
				message, *b.DataType, f.name,
			)
		}
		b.DataType = nil
	}
	return b
}

// unselect drops each channel binding of bindings to a field that a multiplexor with a
// fixed value never selects. A drop changes no other result, as each field below a
// field that is never selected is never selected either.
func (r *repair) unselect(
	bindings []icd.WriteBinding,
	message string,
	fields []field,
	at map[icd.FieldKey]int,
) []icd.WriteBinding {
	channels := channeled(bindings)
	return slices.DeleteFunc(bindings, func(b icd.WriteBinding) bool {
		if b.Type() != icd.ChannelWriteBindingType {
			return false
		}
		f := fields[at[b.Base().Field]]
		mux, never := neverSelected(f, fields, at, channels)
		if never {
			r.notef(
				true,
				"In message %s, multiplexor %s never selects field %s now, so the "+
					"task drops its channel binding.",
				message, mux, f.name,
			)
		}
		return never
	})
}

// fill adds a channel binding with no channel for each of fields that a multiplexor can
// select and that has no binding and no default.
func (r *repair) fill(
	bindings []icd.WriteBinding,
	message string,
	fields []field,
	at map[icd.FieldKey]int,
) []icd.WriteBinding {
	bound := make(set.Set[icd.FieldKey], len(bindings))
	for _, b := range bindings {
		bound.Add(b.Base().Field)
	}
	channels := channeled(bindings)
	for _, f := range fields {
		if bound.Contains(f.key) || f.defaulted {
			continue
		}
		if _, never := neverSelected(f, fields, at, channels); never {
			continue
		}
		r.notef(
			false,
			"In message %s, field %s has no default, so Configure creates a channel "+
				"for it.",
			message, f.name,
		)
		bindings = append(bindings, icd.WriteBinding{
			Variant: icd.ChannelWriteBinding{
				WriteBaseBinding: icd.WriteBaseBinding{Field: f.key},
			},
		})
	}
	return bindings
}

// channeled returns the index of the binding of each field that a channel sets.
func channeled(bindings []icd.WriteBinding) map[icd.FieldKey]int {
	channels := make(map[icd.FieldKey]int, len(bindings))
	for j, b := range bindings {
		if b.Type() == icd.ChannelWriteBindingType {
			channels[b.Base().Field] = j
		}
	}
	return channels
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
