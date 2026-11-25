// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/validate"
)

// nameSpace is the space of the index that keeps channel names unique across the Core.
const nameSpace = "channel.name"

// newNameIndex returns the index that keeps channel names unique across the Core, by
// their NFC form. A write that keeps the name its channel holds passes, so channels
// with one name from before the index stay writable.
func newNameIndex() *gorp.UniqueIndex[Key, Channel, string] {
	return gorp.NewUniqueIndex(
		"unique_name",
		nameSpace,
		func(c Channel) (string, bool) { return validate.NormalizeName(c.Name), true },
	)
}

// NormalizeWhere returns w with each name that it matches in NFC, the form in which the
// Core stores names, so a lookup by another normal form finds the channel. A regular
// expression is left as is.
func NormalizeWhere(w Where) Where {
	if w.Name != nil {
		name := *w.Name
		name.Eq = normalizePtr(name.Eq)
		name.Contains = normalizePtr(name.Contains)
		name.StartsWith = normalizePtr(name.StartsWith)
		if name.In != nil {
			name.In = lo.Map(name.In, func(n string, _ int) string {
				return validate.NormalizeName(n)
			})
		}
		w.Name = &name
	}
	if w.And != nil {
		w.And = lo.Map(w.And, func(c Where, _ int) Where { return NormalizeWhere(c) })
	}
	if w.Or != nil {
		w.Or = lo.Map(w.Or, func(c Where, _ int) Where { return NormalizeWhere(c) })
	}
	if w.Not != nil {
		not := NormalizeWhere(*w.Not)
		w.Not = &not
	}
	return w
}

func normalizePtr(s *string) *string {
	if s == nil {
		return nil
	}
	n := validate.NormalizeName(*s)
	return &n
}

// NormalizeNames returns names, each in NFC, the form in which the Core stores names.
// It
// returns names itself when every name is in NFC, so the caller must not change the
// result.
func NormalizeNames(names []string) []string {
	var out []string
	for i, n := range names {
		nfc := validate.NormalizeName(n)
		if nfc == n {
			continue
		}
		// Copy only when a name changes, so names already in NFC allocate nothing.
		if out == nil {
			out = slices.Clone(names)
		}
		out[i] = nfc
	}
	if out == nil {
		return names
	}
	return out
}

// normalizeNameActions returns acts with each name that they set in NFC, so the Core
// stores and broadcasts one form.
func normalizeNameActions(acts []Action) []Action {
	var out []Action
	set := func(i int, a Action) {
		if out == nil {
			out = slices.Clone(acts)
		}
		out[i] = a
	}
	for i, a := range acts {
		if a.Rename != nil {
			if nfc := validate.NormalizeName(a.Rename.Name); nfc != a.Rename.Name {
				p := *a.Rename
				p.Name = nfc
				a.Rename = &p
				set(i, a)
			}
		}
		if a.Set != nil && slices.Equal(a.Set.Path, []string{"name"}) {
			if name, ok := a.Set.Value.(string); ok {
				if nfc := validate.NormalizeName(name); nfc != name {
					p := *a.Set
					p.Value = nfc
					a.Set = &p
					set(i, a)
				}
			}
		}
	}
	if out == nil {
		return acts
	}
	return out
}
