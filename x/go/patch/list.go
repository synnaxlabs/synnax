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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Entry is a pointer to a list entry that paths name by its key.
type Entry[T any] interface {
	*T
	// Patch applies the path action at c, which is not done, to the entry.
	Patch(c Cursor) error
	// PatchKey returns the key that paths name the entry by.
	PatchKey() string
}

// List applies the action at c to a list of entries that paths name by key. It copies
// the list before it changes, and does nothing when the path goes through a missing
// entry.
func List[T any, P Entry[T]](dst *[]T, c Cursor) error {
	if c.Done() {
		switch c.kind {
		case kindInsert:
			return insertEntry[T, P](dst, c)
		case kindMove:
			return moveEntry[T, P](dst, c)
		default:
			return whole(dst, c)
		}
	}
	key := c.Segment()
	next := c.Next()
	if next.Done() && c.kind == kindInsert {
		return c.notList()
	}
	list := *dst
	i := entryAt[T, P](list, key)
	if i < 0 {
		return nil
	}
	if next.Done() {
		switch c.kind {
		case kindSet:
			return setEntry(dst, i, key, c)
		case kindRemove:
			return removeEntry[T, P](dst, key, c)
		default:
			return c.notList()
		}
	}
	entries := make([]T, len(list))
	copy(entries, list)
	if err := P(&entries[i]).Patch(next); err != nil {
		return err
	}
	*dst = entries
	return nil
}

// setEntry replaces the entry at index i, whose key is key.
func setEntry[T any](dst *[]T, i int, key string, c Cursor) error {
	if err := c.unguarded(); err != nil {
		return err
	}
	if keyOf(c.value) != key {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: the entry must be an object with the key %s",
			format(c.path),
			key,
		)
	}
	list := *dst
	entries := make([]T, len(list))
	copy(entries, list)
	if err := decode(&entries[i], c); err != nil {
		return err
	}
	*dst = entries
	return nil
}

// removeEntry removes every entry with the key.
func removeEntry[T any, P Entry[T]](dst *[]T, key string, c Cursor) error {
	if err := c.unguarded(); err != nil {
		return err
	}
	list := *dst
	entries := make([]T, 0, len(list)-1)
	for i := range list {
		if P(&list[i]).PatchKey() != key {
			entries = append(entries, list[i])
		}
	}
	*dst = entries
	return nil
}

// insertEntry inserts the action's value into the list, unless an entry has its key.
func insertEntry[T any, P Entry[T]](dst *[]T, c Cursor) error {
	if err := c.unguarded(); err != nil {
		return err
	}
	key := keyOf(c.value)
	if key == "" {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: the entry must be an object with a string key",
			format(c.path),
		)
	}
	list := *dst
	if entryAt[T, P](list, key) >= 0 {
		return nil
	}
	at := entryPosition[T, P](list, c.after)
	entries := make([]T, len(list)+1)
	copy(entries, list[:at])
	copy(entries[at+1:], list[at:])
	if err := decode(&entries[at], c); err != nil {
		return err
	}
	*dst = entries
	return nil
}

// moveEntry moves the entry with the action's key after the entry that it names.
func moveEntry[T any, P Entry[T]](dst *[]T, c Cursor) error {
	list := *dst
	i := entryAt[T, P](list, c.key)
	if i < 0 || (c.after != nil && *c.after == c.key) {
		return nil
	}
	if err := c.unguarded(); err != nil {
		return err
	}
	rest := make([]T, 0, len(list))
	rest = append(rest, list[:i]...)
	rest = append(rest, list[i+1:]...)
	at := entryPosition[T, P](rest, c.after)
	rest = append(rest[:at+1], rest[at:]...)
	rest[at] = list[i]
	*dst = rest
	return nil
}

// entryAt returns the index of the first entry with the key, or -1.
func entryAt[T any, P Entry[T]](list []T, key string) int {
	for i := range list {
		if P(&list[i]).PatchKey() == key {
			return i
		}
	}
	return -1
}

// entryPosition returns the index after the entry with the key after, the first index
// when after is nil, and the end of the list when no entry has the key.
func entryPosition[T any, P Entry[T]](list []T, after *string) int {
	if after == nil {
		return 0
	}
	if i := entryAt[T, P](list, *after); i >= 0 {
		return i + 1
	}
	return len(list)
}
