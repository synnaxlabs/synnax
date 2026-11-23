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
	"github.com/synnaxlabs/x/validate"
)

// checkBindings returns an error wrapping validate.ErrValidation at the first binding
// of a symbol node of s that its symbol refuses. It does not check the bindings of a
// missing symbol.
func (w Writer) checkBindings(ctx context.Context, s Schematic) error {
	custom, err := w.customSymbols(ctx, s)
	if err != nil {
		return err
	}
	for i, n := range s.Nodes {
		cfg, ok := n.Config.Variant.(SymbolNodeConfig)
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

// customSymbols returns the custom symbols that the symbol nodes of s draw, by key.
func (w Writer) customSymbols(
	ctx context.Context,
	s Schematic,
) (map[symbol.Key]symbol.Symbol, error) {
	var keys []symbol.Key
	for _, n := range s.Nodes {
		if cfg, ok := n.Config.Variant.(SymbolNodeConfig); ok {
			if ref, ok := cfg.Spec.Variant.(symbol.CustomRef); ok {
				keys = append(keys, ref.Key)
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	var symbols []symbol.Symbol
	if err := w.symbols.NewRetrieve().
		Where(symbol.MatchKeys(lo.Uniq(keys)...)).
		SkipMissingKeys().
		Entries(&symbols).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return lo.KeyBy(symbols, func(s symbol.Symbol) symbol.Key { return s.Key }), nil
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
