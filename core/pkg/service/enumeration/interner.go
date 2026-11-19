// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/x/set"
)

// Interner turns state lists into enumerations in one transaction. Lists with the same
// values, names, and colors share one enumeration, with keys and order ignored. An
// Interner is not safe for concurrent use.
type Interner struct {
	w    Writer
	made map[string]Key
}

// NewInterner opens an Interner that creates enumerations through w.
func (w Writer) NewInterner() *Interner {
	return &Interner{w: w, made: map[string]Key{}}
}

// Intern returns the key of an enumeration named name with entries. It returns the
// enumeration that an earlier call made when the entries are equal, and otherwise
// creates one. When entries repeat a value, the first entry stays.
func (in *Interner) Intern(
	ctx context.Context,
	name string,
	entries []Entry,
) (Key, error) {
	values := set.Set[float64]{}
	kept := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if !values.Contains(e.Value) {
			values.Add(e.Value)
			kept = append(kept, e)
		}
	}
	id, err := contentID(kept)
	if err != nil {
		return Key{}, err
	}
	if key, ok := in.made[id]; ok {
		return key, nil
	}
	e := Enumeration{Name: name, Entries: kept}
	if err = in.w.create(ctx, &e); err != nil {
		return Key{}, err
	}
	in.made[id] = e.Key
	return e.Key, nil
}

// contentID returns a string that is equal for two lists with the same values, names,
// and colors.
func contentID(entries []Entry) (string, error) {
	type content struct {
		Value float64
		Name  string
		Color *color.Ref
	}
	contents := make([]content, len(entries))
	for i, e := range entries {
		contents[i] = content{Value: e.Value, Name: e.Name, Color: e.Color}
	}
	slices.SortFunc(contents, func(a, b content) int {
		return cmp.Compare(a.Value, b.Value)
	})
	b, err := json.Marshal(contents)
	return string(b), err
}
