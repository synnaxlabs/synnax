// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch

import (
	"encoding/json/v2"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Action is a Set, Remove, Insert, or Move.
type Action interface {
	// Apply applies the action to doc in place.
	Apply(doc map[string]any) error
	// target returns the path that the action writes.
	target() []string
	guards() []Guard
}

func (s Set) target() []string { return s.Path }

func (r Remove) target() []string { return append(slices.Clone(r.Path), r.Key) }

func (ins Insert) target() []string { return ins.Path }

func (m Move) target() []string { return m.Path }

func (s Set) guards() []Guard { return s.Guards }

func (r Remove) guards() []Guard { return r.Guards }

func (ins Insert) guards() []Guard { return ins.Guards }

func (m Move) guards() []Guard { return m.Guards }

// EditMap applies the action to m, the map at root in the JSON form of a document, and
// returns the edited map. It may change m in place. Only the entry that the action's
// path names is edited, decoded, and passed to check, so no other entry can fail the
// edit. An action at root itself edits and checks every entry. A nil check accepts
// every entry. EditMap returns the error of the action or of check, or an error
// wrapping validate.ErrValidation when the action's path is outside root or an entry
// does not decode.
func EditMap[V any](
	m map[string]V,
	root []string,
	a Action,
	check func(key string, entry V) error,
) (map[string]V, error) {
	path := a.target()
	if !within(path, root) {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"%s: the path is outside %s",
			format(path),
			format(root),
		)
	}
	if len(path) == len(root) {
		next, err := editAt(m, root, a)
		if err != nil {
			return nil, err
		}
		for key, entry := range next {
			if err = checkEntry(check, key, entry); err != nil {
				return nil, err
			}
		}
		return next, nil
	}
	key := path[len(root)]
	// Guards may read other entries, so those entries go into the edit unchanged.
	named := make(map[string]V)
	for _, p := range append([][]string{path}, guardPaths(a)...) {
		if len(p) > len(root) && within(p, root) {
			if entry, ok := m[p[len(root)]]; ok {
				named[p[len(root)]] = entry
			}
		}
	}
	next, err := editAt(named, root, a)
	if err != nil {
		return nil, err
	}
	entry, ok := next[key]
	if !ok {
		delete(m, key)
		return m, nil
	}
	if err = checkEntry(check, key, entry); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]V)
	}
	m[key] = entry
	return m, nil
}

func checkEntry[V any](check func(string, V) error, key string, entry V) error {
	if check == nil {
		return nil
	}
	return check(key, entry)
}

func within(path, root []string) bool {
	return len(path) >= len(root) && slices.Equal(path[:len(root)], root)
}

func guardPaths(a Action) [][]string {
	paths := make([][]string, len(a.guards()))
	for i, g := range a.guards() {
		paths[i] = g.Path
	}
	return paths
}

// editAt applies the action to a document that holds entries at root and decodes the
// map that the action leaves there.
func editAt[V any](
	entries map[string]V,
	root []string,
	a Action,
) (map[string]V, error) {
	b, err := json.Marshal(entries)
	if err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "encoding entries: %s", err)
	}
	var doc map[string]any
	if err = json.Unmarshal(b, &doc); err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "decoding entries: %s", err)
	}
	for _, field := range slices.Backward(root) {
		doc = map[string]any{field: doc}
	}
	if err = a.Apply(doc); err != nil {
		return nil, err
	}
	var node any = doc
	for _, field := range root {
		obj, _ := node.(map[string]any)
		node = obj[field]
	}
	if b, err = json.Marshal(node); err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "encoding entries: %s", err)
	}
	var next map[string]V
	if err = json.Unmarshal(b, &next, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "decoding entries: %s", err)
	}
	return next, nil
}

// Edit applies fn to the JSON form of doc and decodes the result into a new D. The
// decode rejects members that D does not have, so an action outside D fails. Edit
// returns the error of fn, or an error wrapping validate.ErrValidation when doc does
// not encode or the result does not decode.
func Edit[D any](doc D, fn func(map[string]any) error) (D, error) {
	var zero D
	b, err := json.Marshal(doc)
	if err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "encoding document: %s", err)
	}
	var m map[string]any
	if err = json.Unmarshal(b, &m); err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "decoding document: %s", err)
	}
	if err = fn(m); err != nil {
		return zero, err
	}
	if b, err = json.Marshal(m); err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "encoding document: %s", err)
	}
	var next D
	if err = json.Unmarshal(b, &next, json.RejectUnknownMembers(true)); err != nil {
		return zero, errors.Wrapf(validate.ErrValidation, "decoding document: %s", err)
	}
	return next, nil
}
