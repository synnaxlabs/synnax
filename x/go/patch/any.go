// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch

// Any applies the action at c to a required field that holds any JSON value. Below the
// field, the action applies to the JSON form of the value.
func Any(dst *any, c Cursor) error {
	if c.Done() && c.kind == kindRemove {
		return zero(dst, c)
	}
	if err := c.unguarded(); err != nil {
		return err
	}
	if c.Done() && c.kind == kindSet {
		v, err := normalize(c.value)
		if err != nil {
			return err
		}
		*dst = v
		return nil
	}
	v, err := normalize(*dst)
	if err != nil {
		return err
	}
	// The action keeps its whole path, so its errors name the path in the document.
	field := c.path[:c.depth]
	doc := map[string]any{field[len(field)-1]: v}
	root := doc
	for i := len(field) - 2; i >= 0; i-- {
		root = map[string]any{field[i]: root}
	}
	if err = c.action().Apply(root); err != nil {
		return err
	}
	*dst = doc[field[len(field)-1]]
	return nil
}

// OptionalAny applies the action at c to an optional field that holds any JSON value.
func OptionalAny(dst **any, c Cursor) error {
	if c.Done() && (c.kind == kindRemove || (c.kind == kindSet && c.value == nil)) {
		return optional(dst, c)
	}
	var v any
	if *dst != nil {
		v = **dst
	}
	if err := Any(&v, c); err != nil {
		return err
	}
	if v != nil || *dst != nil {
		*dst = &v
	}
	return nil
}

// action returns the action at c without its guards.
func (c Cursor) action() Action {
	switch c.kind {
	case kindSet:
		return Set{Path: c.path, Value: c.value}
	case kindRemove:
		return Remove{Path: c.path[:len(c.path)-1], Key: c.path[len(c.path)-1]}
	case kindInsert:
		return Insert{Path: c.path, Value: c.value, After: c.after}
	default:
		return Move{Path: c.path, Key: c.key, After: c.after}
	}
}
