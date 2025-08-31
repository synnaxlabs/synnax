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
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

type Type interface {
	fmt.Stringer
}

type Kind int

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
	globalResolver Resolver
	Symbol         *Symbol
	Parent         *Scope
	Children       []*Scope
}

type Symbol struct {
	Name       string
	Kind       Kind
	Type       Type
	ParserRule antlr.ParserRuleContext
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
		if s.globalResolver != nil {
			_, err := s.globalResolver.Resolve(name)
			if errors.Is(query.NotFound, err) {
				return nil
			}
			return errors.Newf("name %s conflicts with global symbol", name)
		}
		return nil
	}
	return s.Parent.checkForNameConflicts(name)
}

func (s *Scope) AddBlock() *Scope {
	child := &Scope{Parent: s}
	s.Children = append(s.Children, child)
	return child
}

func (s *Scope) AddSymbol(
	name string,
	kind Kind,
	t Type,
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
	s.Children = append(s.Children, child)
	return child, nil
}

func (s *Scope) Get(name string) (*Scope, error) {
	for _, child := range s.Children {
		if child.Symbol != nil && child.Symbol.Name == name {
			return child, nil
		}
	}
	if s.Parent == nil {
		return nil, errors.Newf("undefined symbol: %s", name)
	}
	return s.Parent.Get(name)
}
