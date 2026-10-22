// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/x/set"
)

// Hash returns the hash of the ICD's name and entries. The Core stores it in the ICD's
// Hash field, and a writer sends the hash it read so the Core can reject a stale save.
func Hash(ic ICD) (string, error) {
	return library.Hash(struct {
		Name    string  `json:"name"`
		Entries []Entry `json:"entries"`
	}{Name: ic.Name, Entries: ic.Entries})
}

// TaskHash returns the hash of the entries a task uses: the messages with the given
// keys and the enums their fields name. It ignores keys of messages the ICD does not
// hold. Entries hash in key order, so reordering an ICD's entries does not change it.
func TaskHash(ic ICD, messages []EntryKey) (string, error) {
	used := set.New(messages...)
	enums := make(set.Set[EntryKey])
	var entries []Entry
	for _, e := range ic.Entries {
		m, ok := e.Variant.(MessageEntry)
		if !ok || !used.Contains(m.Key) {
			continue
		}
		entries = append(entries, e)
		for _, b := range m.Payload.FieldBases() {
			if b.Enumeration != nil {
				enums.Add(*b.Enumeration)
			}
		}
	}
	for _, e := range ic.Entries {
		if en, ok := e.Variant.(EnumEntry); ok && enums.Contains(en.Key) {
			entries = append(entries, e)
		}
	}
	slices.SortFunc(entries, func(a, b Entry) int {
		return a.Base().Key.Compare(b.Base().Key)
	})
	return library.Hash(entries)
}
