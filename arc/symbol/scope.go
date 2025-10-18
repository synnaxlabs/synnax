package symbol

import (
	"context"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// CreateRootScope creates a new scope representing the root scope of a program.
func CreateRootScope(globalResolver Resolver) *Scope {
	return &Scope{
		GlobalResolver: globalResolver,
		Symbol:         Symbol{Kind: KindBlock},
		Counter:        new(int),
	}
}

// Channels tracks which Synnax channels a node reads from and writes to.
type Channels struct {
	Read  set.Set[uint32] `json:"read"`
	Write set.Set[uint32] `json:"write"`
}

func (c Channels) Copy() Channels {
	return Channels{
		Read:  c.Read.Copy(),
		Write: c.Write.Copy(),
	}
}

func NewChannels() Channels {
	return Channels{
		Read:  make(set.Set[uint32]),
		Write: make(set.Set[uint32]),
	}
}

// OverrideChannels creates a Channels from other, ensuring non-nil maps.
func OverrideChannels(other Channels) Channels {
	return Channels{
		Read:  lo.Ternary(other.Read != nil, other.Read, make(set.Set[uint32])),
		Write: lo.Ternary(other.Write != nil, other.Write, make(set.Set[uint32])),
	}
}

type Scope struct {
	Symbol
	GlobalResolver Resolver
	Parent         *Scope
	Children       []*Scope
	Counter        *int
	OnResolve      func(ctx context.Context, s *Scope) error
	Channels       Channels
}

func (s *Scope) GetChildByParserRule(rule antlr.ParserRuleContext) (*Scope, error) {
	res := s.FindChild(func(child *Scope) bool { return child.AST == rule })
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

func (s *Scope) FilterChildrenByKind(kind Kind) []*Scope {
	return s.FilterChildren(func(scope *Scope) bool {
		return scope.Kind == kind
	})
}

func (s *Scope) AutoName(prefix string) *Scope {
	idx := s.Parent.addIndex()
	s.Name = prefix + strconv.Itoa(idx)
	return s
}

func (s *Scope) Add(ctx context.Context, sym Symbol) (*Scope, error) {
	if sym.Name != "" {
		if existing, err := s.Resolve(ctx, sym.Name); err == nil {
			if existing.AST == nil {
				return nil, errors.Newf("name %s conflicts with existing symbol", sym.Name)
			}
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
	if sym.Kind == KindFunction {
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

func (s *Scope) Root() *Scope {
	if s.Parent == nil {
		return s
	}
	return s.Parent.Root()
}

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

func (s *Scope) String() string {
	return s.stringWithIndent("")
}

func (s *Scope) ClosestAncestorOfKind(kind Kind) (*Scope, error) {
	if s.Kind == kind {
		return s, nil
	}
	if s.Parent == nil {
		return nil, errors.Newf("undefined symbol")
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
