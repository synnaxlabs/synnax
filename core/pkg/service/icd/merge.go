// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

// merge returns existing with its messages replaced by imported. An imported message
// that matches an existing message by name keeps the existing key, and each of its
// fields keeps the key of the existing field with its name. All other values come from
// imported, and existing messages and fields that imported does not hold are dropped.
// Multiplexor references between imported fields are rewritten to the kept keys. merge
// does not modify its arguments.
func merge(existing ICD, imported []Message) ICD {
	old := make(map[string]Message, len(existing.Messages))
	for _, m := range existing.Messages {
		old[m.Name] = m
	}
	fieldKeys := make(map[FieldKey]FieldKey)
	messages := make([]Message, len(imported))
	for i, m := range imported {
		if match, ok := old[m.Name]; ok {
			m.Key = match.Key
			keepFieldKeys(&m.Payload, match.Payload, fieldKeys)
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
	byName := make(map[string]FieldKey)
	for _, b := range old.FieldBases() {
		byName[b.Name] = b.Key
	}
	p.UpdateFieldBases(func(b *BaseField) {
		if key, ok := byName[b.Name]; ok {
			keys[b.Key] = key
			b.Key = key
		}
	})
}

// remap rewrites the multiplexor references of m through the given key map.
func remap(
	m Message,
	fieldKeys map[FieldKey]FieldKey,
) Message {
	m.Payload.UpdateFieldBases(func(b *BaseField) {
		if b.Multiplexor != nil {
			if key, ok := fieldKeys[*b.Multiplexor]; ok {
				b.Multiplexor = &key
			}
		}
	})
	return m
}
