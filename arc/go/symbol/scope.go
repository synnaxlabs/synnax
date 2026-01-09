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
	"context"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// CreateRootScope creates a new scope representing the root scope of a program.
//
// The root scope is initialized with a new ID counter starting at 0 and an optional
// global resolver for built-in symbols. All scopes added to the root will share the
// same counter unless they are functions, which create their own counters.
func CreateRootScope(globalResolver Resolver) *Scope {
	return &Scope{
		GlobalResolver: globalResolver,
		Symbol:         Symbol{Kind: KindBlock},
		Counter:        new(int),
	}
}

// Channels tracks which Synnax channels a node reads from and writes to.
//
// This is used for data flow analysis to understand which channels are accessed by
// different parts of an Arc program. The maps use channel IDs as keys and channel
// names as values.
type Channels struct {
	// Read contains Synnax channels that the node reads from.
	Read set.Mapped[uint32, string] `json:"read"`
	// Write contains Synnax channels that the node writes to.
	Write set.Mapped[uint32, string] `json:"write"`
}

// Copy returns a deep copy of the Channels.
func (c Channels) Copy() Channels {
	if c.Read == nil {
		c.Read = make(set.Mapped[uint32, string])
	}
	if c.Write == nil {
		c.Write = make(set.Mapped[uint32, string])
	}
	return Channels{Read: c.Read.Copy(), Write: c.Write.Copy()}
}

// NewChannels creates a new Channels with empty read and write sets.
func NewChannels() Channels {
	return Channels{
		Read:  make(set.Mapped[uint32, string]),
		Write: make(set.Mapped[uint32, string]),
	}
}

// Scope represents a symbol scope in the hierarchical scope tree.
//
// Scopes form a tree structure where each scope can have a parent and multiple children.
// The Symbol field is embedded, making each Scope also a Symbol. Scopes can have an
// optional GlobalResolver for looking up built-in symbols, a Parent for lexical scoping,
// and Children for nested scopes.
//
// ID Assignment: Each scope with a Counter (root and functions) assigns unique IDs to
// variables, inputs, outputs, config, and stateful variables added to it or its descendants.
// Functions create new Counter instances to isolate variable IDs per function.
//
// Resolution Order: When resolving a name via Resolve(), the search proceeds in order:
//  1. Children of the current scope
//  2. Parent scope (recursively)
//  3. GlobalResolver (if non-nil)
type Scope struct {
	Symbol
	// GlobalResolver provides global built-in symbols available from any scope.
	GlobalResolver Resolver
	// Parent is the lexically enclosing scope. Nil for the root scope.
	Parent *Scope
	// Children are nested scopes within this scope.
	Children []*Scope
	// Counter is the ID counter for variable kinds. Functions create new counters.
	Counter *int
	// OnResolve is an optional callback invoked when symbols are resolved from this scope.
	OnResolve func(ctx context.Context, s *Scope) error
	// Channels tracks which Synnax channels this scope's AST node reads from and writes to.
	Channels Channels
}

// GetChildByParserRule finds a direct child scope with the given AST parser rule.
// Returns an error if no matching child is found.
func (s *Scope) GetChildByParserRule(rule antlr.ParserRuleContext) (*Scope, error) {
	res := s.FindChild(func(child *Scope) bool { return child.AST == rule })
	if res == nil {
		return nil, errors.New("could not find symbol matching parser rule")
	}
	return res, nil
}

// FindChildByName searches for a direct child scope with the given name.
// Returns nil if no matching child is found.
func (s *Scope) FindChildByName(name string) *Scope {
	return s.FindChild(func(scope *Scope) bool { return scope.Name == name })
}

// FindChild searches for a direct child scope matching the predicate.
// Returns nil if no matching child is found.
func (s *Scope) FindChild(predicate func(*Scope) bool) *Scope {
	res, _ := lo.Find(s.Children, predicate)
	return res
}

// FilterChildren returns all direct child scopes matching the predicate.
func (s *Scope) FilterChildren(predicate func(*Scope) bool) []*Scope {
	return lo.Filter(s.Children, func(item *Scope, _ int) bool {
		return predicate(item)
	})
}

// FilterChildrenByKind returns all direct child scopes of the given kind.
func (s *Scope) FilterChildrenByKind(kind Kind) []*Scope {
	return s.FilterChildren(func(scope *Scope) bool {
		return scope.Kind == kind
	})
}

// AutoName assigns a unique name to this scope by appending a numeric ID to the prefix.
// The ID is obtained from the parent scope's counter. Returns the scope for method chaining.
func (s *Scope) AutoName(prefix string) *Scope {
	idx := s.Parent.addIndex()
	s.Name = prefix + strconv.Itoa(idx)
	return s
}

// Add creates a new child scope with the given symbol and adds it to this scope's children.
//
// If the symbol has a non-empty name, Add checks for naming conflicts with existing symbols
// in the current scope and its parents. Global symbols (with nil AST) can be shadowed.
// Returns an error if a local symbol with the same name already exists.
//
// Functions (KindFunction) receive a new ID counter, while variables, inputs, outputs, config,
// and stateful variables receive unique IDs from the nearest ancestor counter.
func (s *Scope) Add(ctx context.Context, sym Symbol) (*Scope, error) {
	if sym.Name != "" {
		// Don't return error on global symbol shadowing. Global symbols have an
		// empty AST.
		existing, err := s.Resolve(ctx, sym.Name)
		if err == nil && existing.AST != nil {
			tok := existing.AST.GetStart()
			return nil, errors.Newf(
				"name %s conflicts with existing symbol at line %d, col %d",
				sym.Name,
				tok.GetLine(),
				tok.GetColumn(),
			)
		}
	}
	child := &Scope{Parent: s, Symbol: sym}
	if sym.Kind == KindFunction || sym.Kind == KindSequence {
		child.Counter = new(int)
	}
	if sym.Kind == KindVariable ||
		sym.Kind == KindStatefulVariable ||
		sym.Kind == KindInput ||
		sym.Kind == KindConfig ||
		sym.Kind == KindOutput {
		child.ID = s.addIndex()
	}
	s.Children = append(s.Children, child)
	return child, nil
}

func (s *Scope) addIndex() int {
	if s.Counter != nil {
		v := *s.Counter
		*s.Counter++
		return v
	}
	return s.Parent.addIndex()
}

// Root returns the root scope by traversing up the parent chain.
func (s *Scope) Root() *Scope {
	if s.Parent == nil {
		return s
	}
	return s.Parent.Root()
}

// Resolve looks up a symbol by name using lexical scoping rules.
//
// The search proceeds in order: direct children of this scope, the GlobalResolver
// (if present), and then the parent scope (recursively). If OnResolve is set, it
// is invoked with the resolved scope before returning.
//
// Returns an error if the symbol is not found in any scope.
func (s *Scope) Resolve(ctx context.Context, name string) (*Scope, error) {
	if child := s.FindChildByName(name); child != nil {
		if s.OnResolve != nil {
			return child, s.OnResolve(ctx, child)
		}
		return child, nil
	}
	if s.GlobalResolver != nil {
		if sym, err := s.GlobalResolver.Resolve(ctx, name); err == nil {
			scope := &Scope{Symbol: sym}
			if s.OnResolve != nil {
				return scope, s.OnResolve(ctx, scope)
			}
			return scope, nil
		}
	}
	if s.Parent != nil {
		scope, err := s.Parent.Resolve(ctx, name)
		if err != nil {
			return nil, err
		}
		if s.OnResolve != nil {
			return scope, s.OnResolve(ctx, scope)
		}
		return scope, nil
	}
	return nil, errors.Newf("undefined symbol: %s", name)
}

// ResolvePrefix returns all symbols whose names start with the given prefix.
// It searches children, GlobalResolver, and parent scope, deduplicating results.
func (s *Scope) ResolvePrefix(ctx context.Context, prefix string) ([]*Scope, error) {
	seen := make(map[string]bool)
	var scopes []*Scope
	for _, child := range s.Children {
		if strings.HasPrefix(child.Name, prefix) && !seen[child.Name] {
			scopes = append(scopes, child)
			seen[child.Name] = true
		}
	}
	if s.GlobalResolver != nil {
		symbols, err := s.GlobalResolver.ResolvePrefix(ctx, prefix)
		if err == nil {
			for _, sym := range symbols {
				if !seen[sym.Name] {
					scopes = append(scopes, &Scope{Symbol: sym})
					seen[sym.Name] = true
				}
			}
		}
	}
	if s.Parent != nil {
		parentScopes, err := s.Parent.ResolvePrefix(ctx, prefix)
		if err == nil {
			for _, scope := range parentScopes {
				if !seen[scope.Name] {
					scopes = append(scopes, scope)
					seen[scope.Name] = true
				}
			}
		}
	}
	return scopes, nil
}

// String returns a human-readable string representation of the scope tree.
func (s *Scope) String() string { return s.stringWithIndent("") }

// ClosestAncestorOfKind searches up the scope tree for the nearest ancestor of the given kind.
// Returns the current scope if it matches the kind. Returns an error if no ancestor is found.
func (s *Scope) ClosestAncestorOfKind(kind Kind) (*Scope, error) {
	if s.Kind == kind {
		return s, nil
	}
	if s.Parent == nil {
		return nil, errors.Wrap(query.NotFound, "undefined symbol")
	}
	return s.Parent.ClosestAncestorOfKind(kind)
}

// FirstChildOfKind returns the first direct child scope of the given kind.
// Returns an error if no matching child is found.
func (s *Scope) FirstChildOfKind(kind Kind) (*Scope, error) {
	for _, child := range s.Children {
		if child.Kind == kind {
			return child, nil
		}
	}
	return nil, errors.Wrap(query.NotFound, "undefined symbol")
}

func (s *Scope) stringWithIndent(indent string) string {
	builder := strings.Builder{}
	if s.Name != "" {
		builder.WriteString(indent)
		builder.WriteString("name: ")
		builder.WriteString(s.Name)
		builder.WriteString("\n")
	}
	builder.WriteString(indent)
	builder.WriteString("kind: ")
	builder.WriteString(s.Kind.String())
	builder.WriteString("\n")
	if s.Type.Kind != types.KindInvalid {
		builder.WriteString(indent)
		builder.WriteString("type: ")
		builder.WriteString(s.Type.String())
		builder.WriteString("\n")
	}
	if len(s.Children) > 0 {
		builder.WriteString(indent)
		builder.WriteString("children: ")
		builder.WriteString("\n")
		childIndent := indent + "  "
		for _, child := range s.Children {
			builder.WriteString(child.stringWithIndent(childIndent))
			builder.WriteString(childIndent)
			builder.WriteString("---")
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

// AccumulateReadChannels initializes channel tracking for this scope.
// It sets up an OnResolve callback that records any channel references
// in the Channels.Read map. This is used by functions and expressions
// to track their channel dependencies.
func (s *Scope) AccumulateReadChannels() {
	s.Channels = NewChannels()
	s.OnResolve = func(_ context.Context, resolved *Scope) error {
		if resolved.Kind == KindChannel || resolved.Type.Kind == types.KindChan {
			s.Channels.Read[uint32(resolved.ID)] = resolved.Name
		}
		return nil
	}
}
