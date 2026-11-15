// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import "slices"

// merge returns existing with its messages replaced by imported. An imported message
// that matches an existing message by name keeps the existing key, and each of its
// fields keeps the key of the existing field with its name. Each existing message and
// field matches once, so repeated names match in order. All other values come from
// imported, and existing messages and fields that imported does not hold are dropped.
// Multiplexor references between imported fields are rewritten to the kept keys. merge
// does not modify its arguments.
func merge(existing ICD, imported []Message) ICD {
	old := make(map[string][]Message, len(existing.Messages))
	for _, m := range existing.Messages {
		old[m.Name] = append(old[m.Name], m)
	}
	fieldKeys := make(map[FieldKey]FieldKey)
	messages := make([]Message, len(imported))
	for i, m := range imported {
		if matches := old[m.Name]; len(matches) > 0 {
			old[m.Name] = matches[1:]
			m.Key = matches[0].Key
			keepFieldKeys(&m.Payload, matches[0].Payload, fieldKeys)
		}
		messages[i] = m
	}
	for i, m := range messages {
		messages[i] = remap(m, fieldKeys)
	}
	existing.Messages = messages
	return existing
}

// keepFieldKeys gives each field of p named like a field of old that field's key. It
// records each replaced key in keys.
func keepFieldKeys(
	p *Payload,
	old Payload,
	keys map[FieldKey]FieldKey,
) {
	byName := make(map[string][]FieldKey)
	for _, b := range old.FieldBases() {
		byName[b.Name] = append(byName[b.Name], b.Key)
	}
	p.UpdateFieldBases(func(b *BaseField) {
		if matches := byName[b.Name]; len(matches) > 0 {
			byName[b.Name] = matches[1:]
			keys[b.Key] = matches[0]
			b.Key = matches[0]
		}
	})
}

// remap rewrites the multiplexor references of m through the given key map.
func remap(
	m Message,
	fieldKeys map[FieldKey]FieldKey,
) Message {
	p, ok := m.Payload.Variant.(BinaryPayload)
	if !ok {
		return m
	}
	p.Fields = slices.Clone(p.Fields)
	for i, f := range p.Fields {
		if f.Multiplexor == nil {
			continue
		}
		if key, ok := fieldKeys[*f.Multiplexor]; ok {
			p.Fields[i].Multiplexor = &key
		}
	}
	m.Payload.Variant = p
	return m
}
