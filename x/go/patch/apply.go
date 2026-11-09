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
	"context"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ErrStale is returned when an update names a list entry that is missing, or when a
// guard of an update does not hold in the stored record.
var ErrStale = errors.New("stale")

const staleType = "sy.patch.stale"

func encodeStale(_ context.Context, err error) (errors.Payload, bool) {
	if errors.CheapIs(err, ErrStale) {
		return errors.Payload{Type: staleType, Data: err.Error()}, true
	}
	return errors.Payload{}, false
}

func decodeStale(_ context.Context, p errors.Payload) (error, bool) {
	if p.Type != staleType {
		return nil, false
	}
	return errors.Wrap(ErrStale, p.Data), true
}

func init() { errors.Register(encodeStale, decodeStale) }

// Check returns an error wrapping ErrStale when a guard of u does not hold in stored,
// the record that u applies to.
func (u Update[V]) Check(stored any) error {
	if len(u.Guards) == 0 {
		return nil
	}
	normalized, err := normalize(stored)
	if err != nil {
		return err
	}
	doc, ok := normalized.(map[string]any)
	if !ok {
		return errors.Wrap(validate.ErrValidation, "the stored record is not an object")
	}
	for _, g := range u.Guards {
		holds, err := guardsHold(doc, []Guard{g})
		if err != nil {
			return err
		}
		if !holds {
			return errors.Wrapf(ErrStale, "%s is not %s", format(g.Path), g.Value)
		}
	}
	return nil
}

// ApplyTo applies c to the required field dst. It returns an error wrapping
// validate.ErrValidation when c clears the field.
func (c Change[T]) ApplyTo(dst *T) error {
	switch c.op {
	case opSet:
		*dst = c.value
	case opClear:
		return errors.Wrap(validate.ErrValidation, "a required field cannot be cleared")
	}
	return nil
}

// ApplyOptional applies c to the optional field dst. It never writes the value that
// dst points to.
func (c Change[T]) ApplyOptional(dst **T) {
	switch c.op {
	case opSet:
		v := c.value
		*dst = &v
	case opClear:
		*dst = nil
	}
}

// ApplyTo applies s to the required struct field dst. merge merges a partial into dst.
func (s Struct[U, T]) ApplyTo(dst *T, merge func(*T, U) error) error {
	switch s.op {
	case opMerge:
		return merge(dst, s.merge)
	case opSet:
		*dst = s.value
	}
	return nil
}

// ApplyOptionalStruct applies c to the optional struct field dst. merge merges a
// partial into a copy of the value that dst points to. A merge into an absent field
// merges into the value that defaults returns for the partial. defaults returns an
// error
// wrapping validate.ErrValidation for a field with no default that the partial does not
// give.
func ApplyOptionalStruct[U, T any](
	dst **T,
	c Change[Struct[U, T]],
	merge func(*T, U) error,
	defaults func(U) (T, error),
) error {
	if c.Cleared() {
		*dst = nil
		return nil
	}
	s, ok := c.Get()
	if !ok {
		return nil
	}
	var v T
	if *dst != nil {
		v = **dst
	} else if u, merges := s.Merged(); merges {
		var err error
		if v, err = defaults(u); err != nil {
			return err
		}
	}
	if err := s.ApplyTo(&v, merge); err != nil {
		return err
	}
	*dst = &v
	return nil
}

// ApplyList applies ops to the keyed list dst. key and updateKey return the key of an
// entry and of a partial entry, and merge merges a partial into an entry. It replaces
// dst with a new slice and never writes the old one. It returns an error wrapping
// validate.ErrValidation when ops holds set beside another operator, ErrStale for a key
// that is missing, and query.ErrUniqueViolation for a key that the list already holds.
func ApplyList[E any, K comparable, U any](
	dst *[]E,
	ops List[E, K, U],
	key func(E) K,
	updateKey func(U) K,
	merge func(*E, U) error,
) error {
	if ops.Set != nil {
		if ops.Insert != nil || ops.Remove != nil || ops.Update != nil ||
			ops.Move != nil {
			return errors.Wrap(
				validate.ErrValidation,
				"set cannot be combined with other operators",
			)
		}
		list, err := appendUnique(make([]E, 0, len(ops.Set)), ops.Set, key)
		if err != nil {
			return err
		}
		*dst = list
		return nil
	}
	if ops.Insert == nil && ops.Remove == nil && ops.Update == nil && ops.Move == nil {
		return nil
	}
	list := slices.Clone(*dst)
	indexOf := func(k K) (int, error) {
		i := slices.IndexFunc(list, func(e E) bool { return key(e) == k })
		if i < 0 {
			return i, errors.Wrapf(ErrStale, "%v not found", k)
		}
		return i, nil
	}
	for _, k := range ops.Remove {
		i, err := indexOf(k)
		if err != nil {
			return err
		}
		list = slices.Delete(list, i, i+1)
	}
	for _, u := range ops.Update {
		i, err := indexOf(updateKey(u))
		if err != nil {
			return err
		}
		if err = merge(&list[i], u); err != nil {
			return err
		}
	}
	list, err := appendUnique(list, ops.Insert, key)
	if err != nil {
		return err
	}
	for _, m := range ops.Move {
		if m.After != nil && *m.After == m.Key {
			return errors.Wrapf(
				validate.ErrValidation,
				"%v cannot move after itself",
				m.Key,
			)
		}
		i, err := indexOf(m.Key)
		if err != nil {
			return err
		}
		entry := list[i]
		list = slices.Delete(list, i, i+1)
		at := 0
		if m.After != nil {
			if at, err = indexOf(*m.After); err != nil {
				return err
			}
			at++
		}
		list = slices.Insert(list, at, entry)
	}
	*dst = list
	return nil
}

// ApplyValueList applies ops to the value list dst, with the same rules as ApplyList
// and each value as its own key.
func ApplyValueList[E comparable](dst *[]E, ops ValueList[E]) error {
	same := func(e E) E { return e }
	return ApplyList(dst, List[E, E, E]{
		Set:    ops.Set,
		Insert: ops.Insert,
		Remove: ops.Remove,
		Move:   ops.Move,
	}, same, same, nil)
}

// appendUnique appends entries to list, and fails for a key that list already holds.
func appendUnique[E any, K comparable](list, entries []E, key func(E) K) ([]E, error) {
	if len(entries) == 0 {
		return list, nil
	}
	keys := make(map[K]struct{}, len(list)+len(entries))
	for _, e := range list {
		keys[key(e)] = struct{}{}
	}
	for _, e := range entries {
		k := key(e)
		if _, exists := keys[k]; exists {
			return nil, errors.Wrapf(query.ErrUniqueViolation, "%v already exists", k)
		}
		keys[k] = struct{}{}
		list = append(list, e)
	}
	return list, nil
}
