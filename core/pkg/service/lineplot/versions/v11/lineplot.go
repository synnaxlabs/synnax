// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11

import (
	"reflect"
	"slices"

	"github.com/synnaxlabs/gorp"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = LinePlot{}

func (l LinePlot) GorpKey() Key { return l.Key }

func (l LinePlot) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeLineplot, Key: l.Key.String()}
}

// SameRange returns true if a and b refer to the same saved range or the same window,
// ignoring the T0 and color of each.
func SameRange(a, b ranger.Ref) bool {
	switch av := a.Variant.(type) {
	case ranger.KeyRef:
		bv, ok := b.Variant.(ranger.KeyRef)
		return ok && av.Key == bv.Key
	case ranger.WindowRef:
		bv, ok := b.Variant.(ranger.WindowRef)
		return ok && reflect.DeepEqual(av.Window, bv.Window)
	}
	return false
}

// UniqueRanges returns refs without each reference to a range or window that an earlier
// reference names.
func UniqueRanges(refs []ranger.Ref) []ranger.Ref {
	unique := refs[:0:0]
	for _, r := range refs {
		named := func(u ranger.Ref) bool { return SameRange(u, r) }
		if !slices.ContainsFunc(unique, named) {
			unique = append(unique, r)
		}
	}
	return unique
}
