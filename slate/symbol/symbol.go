// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

type Kind int

//go:generate stringer -type=Kind
const (
	KindVariable Kind = iota
	KindStatefulVariable
	KindParam
	KindFunction
	KindTask
	KindChannel
	KindConfigParam
	KindBlock
)

type Symbol struct {
	Name       string
	Kind       Kind
	Type       types.Type
	ParserRule antlr.ParserRuleContext
	ID         int
}

// CreateRoot creates a new scope representing the root scope of a program.
func CreateRoot(globalResolver Resolver) *Scope {
	return &Scope{
		GlobalResolver: globalResolver,
		Symbol:         Symbol{Kind: KindBlock},
		Counter:        new(int),
	}
}

type Scope struct {
	Symbol
	GlobalResolver Resolver
	Parent         *Scope
	Children       []*Scope
	Counter        *int
	OnResolve      func(s *Scope) error
}

func (s *Scope) GetChildByParserRule(rule antlr.ParserRuleContext) (*Scope, error) {
	res := s.FindChild(func(child *Scope) bool { return child.ParserRule == rule })
	if res == nil {
		return nil, errors.New("could not find symbol matching parser rule")
	}
	return res, nil
}

func (s *Scope) FindChildByName(name string) *Scope {
	return s.FindChild(func(scope *Scope) bool { return scope.Name == name })
}

func (s *Scope) FindChild(predicate func(*Scope) bool) *Scope {
	res, _ := lo.Find(s.Children, predicate)
	return res
}

func (s *Scope) FilterChildren(predicate func(*Scope) bool) []*Scope {
	return lo.Filter(s.Children, func(item *Scope, _ int) bool {
		return predicate(item)
	})
}

func (s *Scope) AutoName(prefix string) *Scope {
	idx := s.Parent.addIndex()
	s.Name = prefix + strconv.Itoa(idx)
	return s
}

func (s *Scope) Add(
	name string,
	kind Kind,
	t types.Type,
	parserRule antlr.ParserRuleContext,
) (*Scope, error) {
	if name != "" {
		if sym, err := s.Resolve(name); err == nil {
			if sym.ParserRule == nil {
				return nil, errors.Newf("name %s conflicts with existing symbol", name)
			}
			tok := sym.ParserRule.GetStart()
			return nil, errors.Newf(
				"name %s conflicts with existing symbol at line %d, col %d",
				name,
				tok.GetLine(),
				tok.GetColumn(),
			)
		}
	}
	child := &Scope{Parent: s, Symbol: Symbol{
		Name:       name,
		Kind:       kind,
		ParserRule: parserRule,
		Type:       t,
	}}
	if kind == KindFunction || kind == KindTask {
		child.Counter = new(int)
	}
	if kind == KindVariable || kind == KindStatefulVariable || kind == KindParam {
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

func (s *Scope) Root() *Scope {
	if s.Parent == nil {
		return s
	}
	return s.Parent.Root()
}

func (s *Scope) Resolve(name string) (*Scope, error) {
	if child := s.FindChildByName(name); child != nil {
		if s.OnResolve != nil {
			return child, s.OnResolve(child)
		}
		return child, nil
	}
	if s.GlobalResolver != nil {
		if sym, err := s.GlobalResolver.Resolve(name); err == nil {
			scope := &Scope{Symbol: sym}
			if s.OnResolve != nil {
				return scope, s.OnResolve(scope)
			}
			return scope, nil
		}
	}
	if s.Parent != nil {
		scope, err := s.Parent.Resolve(name)
		if err != nil {
			return nil, err
		}
		if s.OnResolve != nil {
			return scope, s.OnResolve(scope)
		}
		return scope, nil
	}
	return nil, errors.Newf("undefined symbol: %s", name)
}

func (s *Scope) String() string {
	return s.stringWithIndent("")
}

func (s *Scope) ClosestAncestorOfKind(kind Kind) (*Scope, error) {
	if s.Parent == nil {
		return nil, errors.Newf("undefined symbol")
	}
	if s.Kind == kind {
		return s, nil
	}
	return s.Parent.ClosestAncestorOfKind(kind)
}

func (s *Scope) FirstChildOfKind(kind Kind) (*Scope, error) {
	for _, child := range s.Children {
		if child.Kind == kind {
			return child, nil
		}
	}
	return nil, errors.Newf("undefined symbol")
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
	if s.Type != nil {
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
