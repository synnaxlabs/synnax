// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"slices"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/selector"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// checkSpec returns an error wrapping validate.ErrValidation at the first field of s
// that breaks a rule of the symbol model: a repeated key in a list, a key that names no
// part, state, or command, a paint, condition, or motion on the wrong type of state, an
// ask command with no state, bounds whose lower is not below their upper, a next that
// is not two fixed commands on one state with different values, or a selector that does
// not name shapes.
func checkSpec(s Spec) error {
	c, err := newChecker(s)
	if err != nil {
		return err
	}
	for i, p := range s.Parts {
		for j, sel := range p.Selectors {
			if !selector.IsName(sel) {
				return invalid(
					at("parts", i, "selectors", strconv.Itoa(j)),
					"%q is not an #id, .class, or [data-region-id] selector",
					sel,
				)
			}
		}
	}
	for i, st := range s.States {
		if err = checkState(i, st); err != nil {
			return err
		}
	}
	for i, cmd := range s.Commands {
		if err = c.checkCommand(i, cmd); err != nil {
			return err
		}
	}
	for i, l := range s.Looks {
		if err = c.checkLook(i, l); err != nil {
			return err
		}
	}
	for i, in := range s.Interactions {
		if err = c.checkInteraction(i, in); err != nil {
			return err
		}
	}
	return nil
}

// newChecker indexes the parts, states, and commands of s by key. It refuses a key
// that repeats in a list.
func newChecker(s Spec) (c checker, err error) {
	partKey := func(p Part) string { return p.Key }
	if c.parts, err = keyIndex("parts", "part", s.Parts, partKey); err != nil {
		return c, err
	}
	handleKey := func(h Handle) string { return h.Key }
	if _, err = keyIndex("handles", "handle", s.Handles, handleKey); err != nil {
		return c, err
	}
	stateKey := func(st State) string { return st.Base().Key }
	if c.states, err = keyIndex("states", "state", s.States, stateKey); err != nil {
		return c, err
	}
	commandKey := func(cmd Command) string { return cmd.Key }
	if c.commands, err = keyIndex(
		"commands",
		"command",
		s.Commands,
		commandKey,
	); err != nil {
		return c, err
	}
	lookKey := func(l Look) string { return l.Key }
	if _, err = keyIndex("looks", "look", s.Looks, lookKey); err != nil {
		return c, err
	}
	interactionKey := func(in Interaction) string { return in.Base().Key }
	_, err = keyIndex("interactions", "interaction", s.Interactions, interactionKey)
	return c, err
}

type checker struct {
	parts    map[string]Part
	states   map[string]State
	commands map[string]Command
}

func checkState(i int, st State) error {
	switch v := st.Variant.(type) {
	case EnumerationState:
		if w, ok := v.Match.Variant.(WithinMatcher); ok {
			return checkBounds(at("states", i, "match", "bounds"), w.Bounds)
		}
	case NumberState:
		return checkBounds(at("states", i, "bounds"), v.Bounds)
	}
	return nil
}

func checkBounds(p []string, b spatial.Bounds) error {
	if b.Lower < b.Upper {
		return nil
	}
	return invalid(p, "lower %v must be below upper %v", b.Lower, b.Upper)
}

func (c checker) checkCommand(i int, cmd Command) error {
	if cmd.State == nil {
		if _, ok := cmd.Value.Variant.(AskCommandValue); ok {
			return invalid(at("commands", i, "state"), "an ask command needs a state")
		}
		return nil
	}
	_, err := c.state(at("commands", i, "state"), *cmd.State)
	return err
}

func (c checker) checkLook(i int, l Look) error {
	if err := c.part(at("looks", i, "part"), l.Part); err != nil {
		return err
	}
	if l.When != nil {
		if eq, ok := l.When.Variant.(EqualsCondition); ok {
			if err := c.enumerationState(
				at("looks", i, "when", "state"),
				eq.State,
			); err != nil {
				return err
			}
		}
	}
	for _, paint := range []struct {
		field string
		paint *Paint
	}{{"fill", l.Fill}, {"stroke", l.Stroke}} {
		if paint.paint == nil {
			continue
		}
		if e, ok := paint.paint.Variant.(EntryPaint); ok {
			if err := c.enumerationState(
				at("looks", i, paint.field, "state"),
				e.State,
			); err != nil {
				return err
			}
		}
	}
	if l.Motion == nil {
		return nil
	}
	if state := motionState(*l.Motion); state != nil {
		return c.numberState(at("looks", i, "motion", "state"), *state)
	}
	return nil
}

func motionState(m Motion) *string {
	switch v := m.Variant.(type) {
	case SpinMotion:
		return v.State
	case FlowMotion:
		return v.State
	case LevelMotion:
		return &v.State
	case RotateMotion:
		return &v.State
	case MoveMotion:
		return &v.State
	}
	return nil
}

func (c checker) checkInteraction(i int, in Interaction) error {
	if err := c.part(at("interactions", i, "part"), in.Base().Part); err != nil {
		return err
	}
	switch v := in.Variant.(type) {
	case FireInteraction:
		_, err := c.command(at("interactions", i, "command"), v.Command)
		return err
	case NextInteraction:
		return c.next(at("interactions", i, "commands"), v.Commands)
	}
	return nil
}

// next checks that keys name two fixed commands on one state whose values differ
// after the match of that state.
func (c checker) next(p, keys []string) error {
	if len(keys) != 2 {
		return invalid(p, "next needs two commands, got %d", len(keys))
	}
	var (
		values [2]float64
		states [2]string
	)
	for i, key := range keys {
		cmd, err := c.command(append(slices.Clone(p), strconv.Itoa(i)), key)
		if err != nil {
			return err
		}
		fixed, ok := cmd.Value.Variant.(FixedCommandValue)
		if !ok || cmd.State == nil {
			return invalid(p, "command %q must be fixed and have a state", key)
		}
		values[i], states[i] = fixed.Value, *cmd.State
	}
	if states[0] != states[1] {
		return invalid(p, "commands %q and %q must have one state", keys[0], keys[1])
	}
	st := c.states[states[0]]
	if matched(st, values[0]) == matched(st, values[1]) {
		return invalid(
			p,
			"commands %q and %q must write different values to state %q",
			keys[0],
			keys[1],
			states[0],
		)
	}
	return nil
}

// matched returns v after the match of st.
func matched(st State, v float64) float64 {
	e, ok := st.Variant.(EnumerationState)
	if !ok {
		return v
	}
	w, ok := e.Match.Variant.(WithinMatcher)
	if !ok {
		return v
	}
	if v >= w.Bounds.Lower && v < w.Bounds.Upper {
		return 1
	}
	return 0
}

func (c checker) part(p []string, key *string) error {
	if key == nil {
		return nil
	}
	if _, ok := c.parts[*key]; !ok {
		return invalid(p, "%q names no part", *key)
	}
	return nil
}

func (c checker) state(p []string, key string) (State, error) {
	st, ok := c.states[key]
	if !ok {
		return State{}, invalid(p, "%q names no state", key)
	}
	return st, nil
}

func (c checker) enumerationState(p []string, key string) error {
	st, err := c.state(p, key)
	if err != nil {
		return err
	}
	if _, ok := st.Variant.(NumberState); ok {
		return invalid(p, "%q names a number state, not an enumeration state", key)
	}
	return nil
}

func (c checker) numberState(p []string, key string) error {
	st, err := c.state(p, key)
	if err != nil {
		return err
	}
	if _, ok := st.Variant.(NumberState); !ok {
		return invalid(p, "%q names an enumeration state, not a number state", key)
	}
	return nil
}

func (c checker) command(p []string, key string) (Command, error) {
	cmd, ok := c.commands[key]
	if !ok {
		return Command{}, invalid(p, "%q names no command", key)
	}
	return cmd, nil
}

// keyIndex maps each item to its key, and refuses a key that repeats in field.
func keyIndex[T any](
	field, noun string,
	items []T,
	key func(T) string,
) (map[string]T, error) {
	byKey := make(map[string]T, len(items))
	for i, item := range items {
		k := key(item)
		if _, ok := byKey[k]; ok {
			return nil, invalid(
				at(field, i, "key"),
				"another %s has the key %q",
				noun,
				k,
			)
		}
		byKey[k] = item
	}
	return byKey, nil
}

// at returns the path of a field of item i in a list of a symbol's data.
func at(list string, i int, fields ...string) []string {
	return append([]string{"data", list, strconv.Itoa(i)}, fields...)
}

func invalid(p []string, format string, args ...any) error {
	err := errors.Wrapf(validate.ErrValidation, format, args...)
	return validate.PathedError(err, p...)
}
