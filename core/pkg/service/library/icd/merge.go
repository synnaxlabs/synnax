// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import "github.com/synnaxlabs/synnax/pkg/service/library/versions"

// Merge returns existing with its entries replaced by imported. An imported entry that
// matches an existing entry of the same kind by name keeps the existing key, and each
// field of a matched message keeps the key of the existing field with its name. All
// other values come from imported, and existing entries and fields that imported does
// not hold are dropped. References between imported entries and fields are rewritten
// to the kept keys. Merge does not modify its arguments.
func Merge(existing versions.Library, imported []versions.Entry) versions.Library {
	old := make(map[string]versions.Entry, len(existing.Entries))
	for _, e := range existing.Entries {
		old[e.Base().Name] = e
	}
	var (
		entryKeys = make(map[versions.EntryKey]versions.EntryKey)
		fieldKeys = make(map[versions.FieldKey]versions.FieldKey)
		entries   = make([]versions.Entry, len(imported))
	)
	for i, e := range imported {
		switch v := e.Variant.(type) {
		case versions.EnumEntry:
			if match, ok := old[v.Name].Variant.(versions.EnumEntry); ok {
				entryKeys[v.Key] = match.Key
				v.Key = match.Key
			}
			entries[i] = versions.Entry{Variant: v}
		case versions.MessageEntry:
			if match, ok := old[v.Name].Variant.(versions.MessageEntry); ok {
				entryKeys[v.Key] = match.Key
				v.Key = match.Key
				keepFieldKeys(&v.Payload, match.Payload, fieldKeys)
			}
			entries[i] = versions.Entry{Variant: v}
		default:
			entries[i] = e
		}
	}
	for i, e := range entries {
		if m, ok := e.Variant.(versions.MessageEntry); ok {
			entries[i].Variant = remap(m, entryKeys, fieldKeys)
		}
	}
	existing.Entries = entries
	return existing
}

// keepFieldKeys gives each field of p named like a field of old that field's key. It
// records each replaced key in keys.
func keepFieldKeys(
	p *versions.Payload,
	old versions.Payload,
	keys map[versions.FieldKey]versions.FieldKey,
) {
	byName := make(map[string]versions.FieldKey)
	for _, b := range old.FieldBases() {
		byName[b.Name] = b.Key
	}
	p.UpdateFieldBases(func(b *versions.BaseField) {
		if key, ok := byName[b.Name]; ok {
			keys[b.Key] = key
			b.Key = key
		}
	})
}

// remap rewrites the entry and field references of m through the given key maps.
func remap(
	m versions.MessageEntry,
	entryKeys map[versions.EntryKey]versions.EntryKey,
	fieldKeys map[versions.FieldKey]versions.FieldKey,
) versions.MessageEntry {
	if p, ok := m.Payload.Variant.(versions.BinaryPayload); ok && p.Identifier != nil {
		if id, ok := p.Identifier.Variant.(versions.FieldIdentifier); ok {
			if key, ok := fieldKeys[id.Field]; ok {
				id.Field = key
				p.Identifier = &versions.Identifier{Variant: id}
				m.Payload.Variant = p
			}
		}
	}
	m.Payload.UpdateFieldBases(func(b *versions.BaseField) {
		if b.Enumeration != nil {
			if key, ok := entryKeys[*b.Enumeration]; ok {
				b.Enumeration = &key
			}
		}
		if b.Multiplexor != nil {
			if key, ok := fieldKeys[*b.Multiplexor]; ok {
				b.Multiplexor = &key
			}
		}
	})
	return m
}
