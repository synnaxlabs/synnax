// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package patch applies path actions to the JSON form of a record. A segment of a path
// into an object names a field, and a segment into a list names the key of an entry,
// never an index. Actions change the document in place. An action whose guards fail, or
// whose path goes through a missing list entry, does nothing. An action the document
// cannot hold returns an error that wraps validate.ErrValidation.
package patch

import (
	"encoding/json/v2"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

const keyField = "key"

// Apply replaces the value at the path in doc.
func (s Set) Apply(doc map[string]any) error {
	parent, last, err := split(s.Path)
	if err != nil {
		return err
	}
	if ok, err := guardsHold(doc, s.Guards); !ok || err != nil {
		return err
	}
	value, err := normalize(s.Value)
	if err != nil {
		return err
	}
	return update(doc, parent, true, func(container any) (any, error) {
		switch c := container.(type) {
		case map[string]any:
			c[last] = value
			return c, nil
		case []any:
			i := entryIndex(c, last)
			if i < 0 {
				return c, nil
			}
			if keyOf(value) != last {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"%s: the entry must be an object with the key %s",
					format(s.Path), last,
				)
			}
			c[i] = value
			return c, nil
		default:
			return nil, notContainer(parent)
		}
	})
}

// Apply removes the entry with the key from the list at the path in doc, or the field
// with the key from the object at the path.
func (r Remove) Apply(doc map[string]any) error {
	if ok, err := guardsHold(doc, r.Guards); !ok || err != nil {
		return err
	}
	return update(doc, r.Path, false, func(container any) (any, error) {
		switch c := container.(type) {
		case map[string]any:
			delete(c, r.Key)
			return c, nil
		case []any:
			return slices.DeleteFunc(c, func(e any) bool {
				return keyOf(e) == r.Key
			}), nil
		default:
			return nil, notContainer(r.Path)
		}
	})
}

// Apply inserts the entry into the list at the path in doc, creating the list when it
// is absent.
func (ins Insert) Apply(doc map[string]any) error {
	parent, last, err := split(ins.Path)
	if err != nil {
		return err
	}
	if ok, err := guardsHold(doc, ins.Guards); !ok || err != nil {
		return err
	}
	value, err := normalize(ins.Value)
	if err != nil {
		return err
	}
	key := keyOf(value)
	if key == "" {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: the entry must be an object with a string key",
			format(ins.Path),
		)
	}
	return update(doc, parent, true, func(container any) (any, error) {
		c, ok := container.(map[string]any)
		if !ok {
			return nil, notList(ins.Path)
		}
		list, err := listAt(c, last, ins.Path)
		if err != nil {
			return nil, err
		}
		if entryIndex(list, key) >= 0 {
			return c, nil
		}
		c[last] = slices.Insert(list, position(list, ins.After), value)
		return c, nil
	})
}

// Apply moves the entry with the key in the list at the path in doc.
func (m Move) Apply(doc map[string]any) error {
	if ok, err := guardsHold(doc, m.Guards); !ok || err != nil {
		return err
	}
	return update(doc, m.Path, false, func(container any) (any, error) {
		list, ok := container.([]any)
		if !ok {
			return nil, notList(m.Path)
		}
		i := entryIndex(list, m.Key)
		if i < 0 || (m.After != nil && *m.After == m.Key) {
			return list, nil
		}
		entry := list[i]
		list = slices.Delete(list, i, i+1)
		return slices.Insert(list, position(list, m.After), entry), nil
	})
}

// update runs fn on the container at path inside node and stores what fn returns in its
// place. A missing object on the path is created when create is true. A missing list
// entry, or a missing object when create is false, leaves node unchanged.
func update(
	node any,
	path []string,
	create bool,
	fn func(any) (any, error),
) error {
	_, err := updateAt(node, path, 0, create, fn)
	return err
}

func updateAt(
	node any,
	path []string,
	depth int,
	create bool,
	fn func(any) (any, error),
) (any, error) {
	if depth == len(path) {
		return fn(node)
	}
	seg := path[depth]
	switch n := node.(type) {
	case map[string]any:
		child, ok := n[seg]
		if !ok || child == nil {
			if !create {
				return n, nil
			}
			child = map[string]any{}
		}
		next, err := updateAt(child, path, depth+1, create, fn)
		if err != nil {
			return nil, err
		}
		n[seg] = next
		return n, nil
	case []any:
		i := entryIndex(n, seg)
		if i < 0 {
			return n, nil
		}
		next, err := updateAt(n[i], path, depth+1, create, fn)
		if err != nil {
			return nil, err
		}
		n[i] = next
		return n, nil
	default:
		return nil, notContainer(path[:depth])
	}
}

// guardsHold reports whether every guard's discriminator holds its value in doc. A
// guard whose path is missing does not hold.
func guardsHold(doc map[string]any, guards []Guard) (bool, error) {
	for _, g := range guards {
		if len(g.Path) == 0 {
			return false, errors.Wrap(validate.ErrValidation, "a guard needs a path")
		}
		var current any = doc
		for _, seg := range g.Path {
			switch c := current.(type) {
			case map[string]any:
				current = c[seg]
			case []any:
				i := entryIndex(c, seg)
				if i < 0 {
					return false, nil
				}
				current = c[i]
			default:
				return false, nil
			}
		}
		if v, ok := current.(string); !ok || v != g.Value {
			return false, nil
		}
	}
	return true, nil
}

// normalize converts v to the types JSON decodes into, so values from any wire codec
// compare and encode alike.
func normalize(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "encoding value: %s", err)
	}
	var out any
	if err = json.Unmarshal(b, &out); err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "decoding value: %s", err)
	}
	return out, nil
}

func split(path []string) ([]string, string, error) {
	if len(path) == 0 {
		return nil, "", errors.Wrap(validate.ErrValidation, "the path is empty")
	}
	return path[:len(path)-1], path[len(path)-1], nil
}

func listAt(c map[string]any, field string, path []string) ([]any, error) {
	raw := c[field]
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, notList(path)
	}
	return list, nil
}

// position returns the index after the entry with the key after, the first index when
// after is nil, and the end of the list when no entry has the key.
func position(list []any, after *string) int {
	if after == nil {
		return 0
	}
	if i := entryIndex(list, *after); i >= 0 {
		return i + 1
	}
	return len(list)
}

func entryIndex(list []any, key string) int {
	return slices.IndexFunc(list, func(e any) bool { return keyOf(e) == key })
}

func keyOf(entry any) string {
	m, ok := entry.(map[string]any)
	if !ok {
		return ""
	}
	key, _ := m[keyField].(string)
	return key
}

func format(path []string) string { return "[" + strings.Join(path, ", ") + "]" }

func notContainer(path []string) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%s: the value is not an object or a list",
		format(path),
	)
}

func notList(path []string) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%s: the value is not a list",
		format(path),
	)
}
