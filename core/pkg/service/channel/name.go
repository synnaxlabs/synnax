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
func NormalizeNames(names []string) []string {
	return lo.Map(
		names,
		func(n string, _ int) string { return validate.NormalizeName(n) },
	)
}

// normalizeNameActions returns acts with each name that they set in NFC, so the Core
// stores and broadcasts one form.
func normalizeNameActions(acts []Action) []Action {
	out := slices.Clone(acts)
	for i, a := range out {
		if a.Rename != nil {
			p := *a.Rename
			p.Name = validate.NormalizeName(p.Name)
			out[i].Rename = &p
		}
		if a.Set != nil && slices.Equal(a.Set.Path, []string{"name"}) {
			if name, ok := a.Set.Value.(string); ok {
				p := *a.Set
				p.Value = validate.NormalizeName(name)
				out[i].Set = &p
			}
		}
	}
	return out
}
