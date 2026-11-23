// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// checkBindings returns an error wrapping validate.ErrValidation at the first binding
// of a symbol node of s that its symbol refuses. A nil only checks every node, and
// otherwise it checks the nodes whose keys only holds. It does not check the bindings
// of a missing symbol.
func (w Writer) checkBindings(
	ctx context.Context,
	s Schematic,
	only set.Set[string],
) error {
	var keys []symbol.Key
	for _, n := range s.Nodes {
		if cfg, ok := checked(n, only); ok {
			if ref, ok := cfg.Spec.Variant.(symbol.CustomRef); ok {
				keys = append(keys, ref.Key)
			}
		}
	}
	custom, err := w.customSymbols(ctx, lo.Uniq(keys))
	if err != nil {
		return err
	}
	for i, n := range s.Nodes {
		cfg, ok := checked(n, only)
		if !ok {
			continue
		}
		var (
			states   []symbol.State
			commands []symbol.Command
		)
		switch ref := cfg.Spec.Variant.(type) {
		case symbol.BuiltinRef:
			b, ok := w.builtins[ref.Name]
			if !ok {
				return invalid(
					at(i, "spec", "name"),
					"%q names no built-in symbol",
					ref.Name,
				)
			}
			states, commands = b.States, b.Commands
		case symbol.CustomRef:
			sym, ok := custom[ref.Key]
			// The spec.key reference refuses the write of a symbol that does not exist.
			if !ok {
				continue
			}
			states, commands = sym.Data.States, sym.Data.Commands
		case symbol.MissingRef:
			continue
		}
		if err = checkStateBindings(i, cfg.States, states); err != nil {
			return err
		}
		if err = checkCommandBindings(i, cfg.Commands, commands); err != nil {
			return err
		}
	}
	return nil
}

// dropOrphans returns cfg without the bindings that a symbol with states and commands
// orphans: a binding of a state or command that the symbol lacks, an override that the
// type of its state lacks, and a command binding of a command that has a state.
func dropOrphans(
	cfg SymbolNodeConfig,
	states []symbol.State,
	commands []symbol.Command,
) SymbolNodeConfig {
	cfg.States = lo.FilterMap(
		cfg.States,
		func(b StateBinding, _ int) (StateBinding, bool) {
			st, ok := lo.Find(states, func(st symbol.State) bool {
				return st.Base().Key == b.Key
			})
			if !ok {
				return b, false
			}
			if _, ok = st.Variant.(symbol.NumberState); ok {
				b.Enumeration, b.Match, b.Default = nil, nil, nil
			} else {
				b.Bounds = nil
			}
			return b, true
		},
	)
	cfg.Commands = lo.Filter(cfg.Commands, func(b CommandBinding, _ int) bool {
		cmd, ok := lo.Find(
			commands,
			func(c symbol.Command) bool { return c.Key == b.Key },
		)
		return ok && cmd.State == nil
	})
	return cfg
}

// checked returns the config of n when n is a symbol node that checkBindings checks.
func checked(n Node, only set.Set[string]) (SymbolNodeConfig, bool) {
	cfg, ok := n.Config.Variant.(SymbolNodeConfig)
	return cfg, ok && (only == nil || only.Contains(n.Key))
}

// customSymbols returns the custom symbols with the keys that tx holds, by key.
func (w Writer) customSymbols(
	ctx context.Context,
	keys []symbol.Key,
) (map[symbol.Key]symbol.Symbol, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var symbols []symbol.Symbol
	if err := w.symbols.NewRetrieve().
		Where(symbol.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&symbols).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return lo.KeyBy(symbols, func(s symbol.Symbol) symbol.Key { return s.Key }), nil
}

// rebound returns the keys of the nodes whose symbol or bindings acts can change.
// Reduce must accept acts first, so that each path into a node goes into a field of it.
func rebound(acts []Action) (set.Set[string], error) {
	keys := set.Set[string]{}
	for _, a := range acts {
		var path []string
		switch a.Type {
		case ActionTypeSetNode:
			keys.Add(a.SetNode.Node.Key)
			continue
		case ActionTypeSetNodePosition,
			ActionTypeRemoveNode,
			ActionTypeAddEdge,
			ActionTypeRemoveEdge:
			continue
		case ActionTypeSet:
			path = a.Set.Path
		case ActionTypeRemove:
			path = append(slices.Clone(a.Remove.Path), a.Remove.Key)
		case ActionTypeInsert:
			path = a.Insert.Path
		case ActionTypeMove:
			path = a.Move.Path
		default:
			return nil, errors.Newf("unknown action type %q", a.Type)
		}
		if path[0] == "nodes" && rebinds(path) {
			keys.Add(path[1])
		}
	}
	return keys, nil
}

// rebinds reports whether a patch at path, into a field of the node path[1], can change
// the symbol or bindings of the node.
func rebinds(path []string) bool {
	if path[2] != "config" {
		return false
	}
	if len(path) == 3 {
		return true
	}
	switch path[3] {
	case "spec", "states", "commands":
		return true
	}
	return false
}

func checkStateBindings(
	node int,
	bindings []StateBinding,
	states []symbol.State,
) error {
	for j, b := range bindings {
		p := at(node, "states", strconv.Itoa(j))
		if slices.ContainsFunc(bindings[:j], func(o StateBinding) bool {
			return o.Key == b.Key
		}) {
			return invalid(
				append(p, "key"),
				"another state binding has the key %q",
				b.Key,
			)
		}
		st, ok := lo.Find(states, func(st symbol.State) bool {
			return st.Base().Key == b.Key
		})
		if !ok {
			return invalid(append(p, "key"), "%q names no state of the symbol", b.Key)
		}
		if err := checkOverrides(b, st); err != nil {
			return validate.PathedError(err, p...)
		}
	}
	return nil
}

// checkOverrides refuses an override that the type of st lacks, and bounds whose lower
// is not below their upper.
func checkOverrides(b StateBinding, st symbol.State) error {
	if _, ok := st.Variant.(symbol.NumberState); ok {
		for _, o := range []struct {
			field string
			set   bool
		}{
			{"enumeration", b.Enumeration != nil},
			{"match", b.Match != nil},
			{"default", b.Default != nil},
		} {
			if o.set {
				return invalid(
					[]string{o.field},
					"number state %q has no %s",
					b.Key,
					o.field,
				)
			}
		}
		if b.Bounds == nil {
			return nil
		}
		return symbol.CheckBounds([]string{"bounds"}, *b.Bounds)
	}
	if b.Bounds != nil {
		return invalid([]string{"bounds"}, "enumeration state %q has no bounds", b.Key)
	}
	if b.Match == nil {
		return nil
	}
	if w, ok := b.Match.Variant.(symbol.WithinMatcher); ok {
		return symbol.CheckBounds([]string{"match", "bounds"}, w.Bounds)
	}
	return nil
}

func checkCommandBindings(
	node int,
	bindings []CommandBinding,
	commands []symbol.Command,
) error {
	for j, b := range bindings {
		p := at(node, "commands", strconv.Itoa(j), "key")
		if slices.ContainsFunc(bindings[:j], func(o CommandBinding) bool {
			return o.Key == b.Key
		}) {
			return invalid(p, "another command binding has the key %q", b.Key)
		}
		cmd, ok := lo.Find(
			commands,
			func(c symbol.Command) bool { return c.Key == b.Key },
		)
		if !ok {
			return invalid(p, "%q names no command of the symbol", b.Key)
		}
		if cmd.State != nil {
			return invalid(
				p,
				"command %q writes to the channel of state %q",
				b.Key,
				*cmd.State,
			)
		}
	}
	return nil
}

// at returns the path of a field in the config of node i.
func at(i int, fields ...string) []string {
	return append([]string{"nodes", strconv.Itoa(i), "config"}, fields...)
}

func invalid(p []string, format string, args ...any) error {
	err := errors.Wrapf(validate.ErrValidation, format, args...)
	return validate.PathedError(err, p...)
}
