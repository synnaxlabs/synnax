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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
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

type Scope struct {
	GlobalResolver Resolver
	Symbol         *Symbol
	Parent         *Scope
	Children       []*Scope
	Counter        *int
}

func (s *Scope) Blocks() []*Scope {
	return lo.Filter(s.Children, func(item *Scope, _ int) bool {
		return item.Symbol.Kind == KindBlock
	})
}

type Symbol struct {
	Name       string
	Kind       Kind
	Type       types.Type
	ParserRule antlr.ParserRuleContext
	ID         int
}

func (s *Scope) checkForNameConflicts(name string) error {
	for _, child := range s.Children {
		if child.Symbol != nil && child.Symbol.Name == name {
			tok := child.Symbol.ParserRule.GetStart()
			return errors.Newf(
				"name %s conflicts with existing symbol at line %d, col %d",
				name,
				tok.GetLine(),
				tok.GetColumn(),
			)
		}
	}
	if s.Parent == nil {
		if s.GlobalResolver != nil {
			_, err := s.GlobalResolver.Resolve(name)
			if errors.Is(err, query.NotFound) {
				return nil
			}
			return errors.Newf("name %s conflicts with global symbol", name)
		}
		return nil
	}
	return s.Parent.checkForNameConflicts(name)
}

func (s *Scope) AddBlock(rule antlr.ParserRuleContext) *Scope {
	child := &Scope{
		Parent: s,
		Symbol: &Symbol{Kind: KindBlock, ParserRule: rule},
	}
	s.Children = append(s.Children, child)
	return child
}

func (s *Scope) FindByParserRule(rule antlr.ParserRuleContext) (*Scope, error) {
	r, ok := lo.Find(s.Children, func(item *Scope) bool {
		return item.Symbol != nil && item.Symbol.ParserRule == rule
	})
	if !ok {
		return nil, errors.Newf("could not find")
	}
	return r, nil
}

func (s *Scope) AddSymbol(
	name string,
	kind Kind,
	t types.Type,
	parserRule antlr.ParserRuleContext,
) (*Scope, error) {
	if err := s.checkForNameConflicts(name); err != nil {
		return nil, err
	}
	child := &Scope{Parent: s, Symbol: &Symbol{
		Name:       name,
		Kind:       kind,
		ParserRule: parserRule,
		Type:       t,
	}}
	if kind == KindFunction || kind == KindTask {
		child.Counter = new(int)
	}
	if kind == KindVariable || kind == KindStatefulVariable || kind == KindParam {
		child.Symbol.ID = s.addIndex()
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

func (s *Scope) Get(name string) (*Scope, error) {
	for _, child := range s.Children {
		if child.Symbol != nil && child.Symbol.Name == name {
			return child, nil
		}
	}
	if s.Parent == nil {
		if s.GlobalResolver != nil {
			if s, err := s.GlobalResolver.Resolve(name); err == nil {
				return &Scope{Symbol: s}, nil
			}
		}
		return nil, errors.Newf("undefined symbol: %s", name)
	}
	return s.Parent.Get(name)
}

func (s *Scope) String() string {
	return s.stringWithIndent("")
}

func (s *Scope) ClosestAncestorOfKind(kind Kind) (*Scope, error) {
	if s.Parent == nil {
		return nil, errors.Newf("undefined symbol")
	}
	if s.Symbol != nil && s.Symbol.Kind == kind {
		return s, nil
	}
	return s.Parent.ClosestAncestorOfKind(kind)
}

func (s *Scope) FirstChildOfKind(kind Kind) (*Scope, error) {
	for _, child := range s.Children {
		if child.Symbol != nil && child.Symbol.Kind == kind {
			return child, nil
		}
	}
	return nil, errors.Newf("undefined symbol")
}

func (s *Scope) stringWithIndent(indent string) string {
	builder := strings.Builder{}

	if s.Symbol != nil {
		builder.WriteString(indent)
		builder.WriteString("name: ")
		builder.WriteString(s.Symbol.Name)
		builder.WriteString("\n")
		builder.WriteString(indent)
		builder.WriteString("kind: ")
		builder.WriteString(s.Symbol.Kind.String())
		builder.WriteString("\n")
		builder.WriteString(indent)
		if s.Symbol.Type != nil {
			builder.WriteString("type: ")
			builder.WriteString(s.Symbol.Type.String())
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString(indent)
		builder.WriteString("block\n")
	}

	childIndent := indent + "  "
	for _, child := range s.Children {
		builder.WriteString(child.stringWithIndent(childIndent))
	}

	return builder.String()
}
