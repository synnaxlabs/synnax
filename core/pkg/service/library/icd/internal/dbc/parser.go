// Copyright 2019 Einride AB. Modified by Synnax Labs, Inc.
//
// Use of this source code is governed by the MIT license in the LICENSE file in this
// directory.

// Package dbc parses Vector DBC files into their definitions. It derives from
// go.einride.tech/can/pkg/dbc v0.17.0, adds extended multiplexing, and skips an unknown
// definition without skipping the line after it.
package dbc

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"text/scanner"
	"unicode/utf8"
)

const defaultScannerMode = scanner.ScanIdents | scanner.ScanFloats

const (
	defaultWhitespace  = scanner.GoWhitespace
	significantNewline = defaultWhitespace & ^uint64(1<<'\n')
	significantTab     = defaultWhitespace & ^uint64(1<<'\t')
)

// Parse returns the definitions of a DBC file in file order. A definition it does not
// know becomes an UnknownDef that spans the rest of its line. It returns an *Error when
// the file does not parse.
func Parse(data []byte) (defs []Def, err error) {
	p := newParser(data)
	defer func() {
		if r := recover(); r != nil {
			perr, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			defs, err = nil, perr
		}
	}()
	p.parse()
	return p.defs, nil
}

type token struct {
	typ rune
	pos scanner.Position
	txt string
}

type parser struct {
	sc           scanner.Scanner
	curr         token
	lookahead    token
	hasLookahead bool
	defs         []Def
}

func newParser(data []byte) *parser {
	p := &parser{}
	p.sc.Init(bytes.NewReader(data))
	p.sc.Mode = defaultScannerMode
	p.sc.Whitespace = defaultWhitespace
	p.sc.Error = func(sc *scanner.Scanner, msg string) { p.failf(sc.Pos(), "%s", msg) }
	return p
}

func (p *parser) parse() {
	for p.peekToken().typ != scanner.EOF {
		var def Def
		switch p.peekKeyword() {
		case KeywordVersion:
			def = &VersionDef{}
		case KeywordBitTiming:
			def = &BitTimingDef{}
		case KeywordNewSymbols:
			def = &NewSymbolsDef{}
		case KeywordNodes:
			def = &NodesDef{}
		case KeywordMessage:
			def = &MessageDef{}
		case KeywordSignal:
			def = &SignalDef{}
		case KeywordEnvironmentVariable:
			def = &EnvironmentVariableDef{}
		case KeywordComment:
			def = &CommentDef{}
		case KeywordAttribute:
			def = &AttributeDef{}
		case KeywordAttributeDefault:
			def = &AttributeDefaultValueDef{}
		case KeywordAttributeValue:
			def = &AttributeValueForObjectDef{}
		case KeywordValueDescriptions:
			def = &ValueDescriptionsDef{}
		case KeywordValueTable:
			def = &ValueTableDef{}
		case KeywordSignalValueType:
			def = &SignalValueTypeDef{}
		case KeywordMessageTransmitters:
			def = &MessageTransmittersDef{}
		case KeywordEnvironmentVariableData:
			def = &EnvironmentVariableDataDef{}
		case KeywordMultiplexedValues:
			def = &MultiplexedValuesDef{}
		default:
			def = &UnknownDef{}
		}
		def.parseFrom(p)
		p.defs = append(p.defs, def)
	}
}

func (p *parser) failf(pos scanner.Position, format string, a ...any) {
	panic(&Error{Pos: pos, Reason: fmt.Sprintf(format, a...)})
}

func (p *parser) useWhitespace(whitespace uint64) {
	p.sc.Whitespace = whitespace
}

func (p *parser) nextRune() rune {
	if p.hasLookahead {
		if utf8.RuneCountInString(p.lookahead.txt) > 1 {
			p.failf(
				p.lookahead.pos,
				"cannot get next rune when lookahead contains a token",
			)
		}
		p.hasLookahead = false
		r, _ := utf8.DecodeRuneInString(p.lookahead.txt)
		return r
	}
	return p.sc.Next()
}

func (p *parser) peekRune() rune {
	if p.hasLookahead {
		if utf8.RuneCountInString(p.lookahead.txt) > 1 {
			p.failf(
				p.lookahead.pos,
				"cannot peek next rune when lookahead contains a token",
			)
		}
		r, _ := utf8.DecodeRuneInString(p.lookahead.txt)
		return r
	}
	return p.sc.Peek()
}

// discardLine consumes tokens up to and including the next newline.
func (p *parser) discardLine() {
	p.useWhitespace(significantNewline)
	defer p.useWhitespace(defaultWhitespace)
	for {
		if typ := p.nextToken().typ; typ == '\n' || typ == scanner.EOF {
			return
		}
	}
}

func (p *parser) nextToken() token {
	if p.hasLookahead {
		p.hasLookahead = false
		p.curr = p.lookahead
		return p.lookahead
	}
	p.curr = token{typ: p.sc.Scan(), pos: p.sc.Position, txt: p.sc.TokenText()}
	return p.curr
}

func (p *parser) peekToken() token {
	if p.hasLookahead {
		return p.lookahead
	}
	p.hasLookahead = true
	p.lookahead = token{typ: p.sc.Scan(), pos: p.sc.Position, txt: p.sc.TokenText()}
	return p.lookahead
}

// string parses a string that may contain newlines.
func (p *parser) string() string {
	tok := p.nextToken()
	if tok.typ != '"' {
		p.failf(tok.pos, `expected token "`)
	}
	var b strings.Builder
	for {
		switch r := p.nextRune(); r {
		case scanner.EOF:
			p.failf(tok.pos, "unterminated string")
		case '"':
			return b.String()
		case '\n':
			b.WriteRune(' ')
		case '\\':
			if p.peekRune() == '"' {
				_ = p.nextRune()
				b.WriteString(`\"`)
				continue
			}
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
}

func (p *parser) identifier() Identifier {
	tok := p.nextToken()
	if tok.typ != scanner.Ident {
		p.failf(tok.pos, "expected ident")
	}
	id := Identifier(tok.txt)
	if err := id.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return id
}

func (p *parser) stringIdentifier() Identifier {
	tok := p.peekToken()
	id := Identifier(p.string())
	if err := id.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return id
}

func (p *parser) keyword(kw Keyword) token {
	if p.peekKeyword() != kw {
		p.failf(p.peekToken().pos, "expected keyword: %v", kw)
	}
	return p.nextToken()
}

func (p *parser) peekKeyword() Keyword {
	tok := p.peekToken()
	if tok.typ != scanner.Ident {
		p.failf(tok.pos, "expected ident")
	}
	return Keyword(tok.txt)
}

func (p *parser) token(typ rune) {
	if tok := p.nextToken(); tok.typ != typ {
		p.failf(
			tok.pos,
			"expected token: %v, found: %v (%v)",
			scanner.TokenString(typ),
			scanner.TokenString(tok.typ),
			tok.txt,
		)
	}
}

func (p *parser) optionalToken(typ rune) {
	if p.peekToken().typ == typ {
		p.token(typ)
	}
}

// enumValue parses an enum attribute value given as a string or, as some files do, as
// an index into values.
func (p *parser) enumValue(values []string) string {
	tok := p.peekToken()
	if tok.typ == scanner.Int {
		i := p.uint()
		if i >= uint64(len(values)) {
			p.failf(tok.pos, "enum index out of bounds")
		}
		return values[i]
	}
	return p.string()
}

func (p *parser) float() float64 {
	isNegative := false
	if p.peekToken().typ == '-' {
		p.token('-')
		isNegative = true
	}
	tok := p.nextToken()
	if tok.typ != scanner.Int && tok.typ != scanner.Float {
		p.failf(tok.pos, "expected int or float")
	}
	f, err := strconv.ParseFloat(tok.txt, 64)
	if err != nil {
		p.failf(tok.pos, "invalid float")
	}
	if isNegative {
		f *= -1
	}
	return f
}

func (p *parser) int() int64 {
	isNegative := false
	if p.peekToken().typ == '-' {
		p.token('-')
		isNegative = true
	}
	tok := p.nextToken()
	if tok.typ != scanner.Int && tok.typ != scanner.Float {
		p.failf(tok.pos, "expected int or float")
	}
	f, err := strconv.ParseFloat(tok.txt, 64)
	if err != nil {
		p.failf(tok.pos, "invalid int")
	}
	i := int64(f)
	if f > math.MaxInt64 {
		i = math.MaxInt64
	} else if f < math.MinInt64 {
		i = math.MinInt64
	}
	if isNegative {
		i *= -1
	}
	return i
}

func (p *parser) uint() uint64 {
	tok := p.nextToken()
	if tok.typ != scanner.Int {
		p.failf(tok.pos, "expected int")
	}
	i, err := strconv.ParseUint(tok.txt, 10, 64)
	if err != nil {
		p.failf(tok.pos, "invalid uint")
	}
	return i
}

func (p *parser) intInRange(rangeMin, rangeMax int) int {
	isNegative := false
	if p.peekToken().typ == '-' {
		p.token('-')
		isNegative = true
	}
	tok := p.nextToken()
	i, err := strconv.Atoi(tok.txt)
	if err != nil {
		p.failf(tok.pos, "invalid int")
	}
	if isNegative {
		i *= -1
	}
	if i < rangeMin || i > rangeMax {
		p.failf(tok.pos, "invalid value")
	}
	return i
}

func (p *parser) optionalUint() uint64 {
	if p.peekToken().typ != scanner.Int {
		return 0
	}
	return p.uint()
}

func (p *parser) anyOf(tokenTypes ...rune) rune {
	tok := p.nextToken()
	if slices.Contains(tokenTypes, tok.typ) {
		return tok.typ
	}
	p.failf(tok.pos, "unexpected token")
	return 0
}

func (p *parser) optionalObjectType() ObjectType {
	tok := p.peekToken()
	if tok.typ != scanner.Ident {
		return ObjectTypeUnspecified
	}
	objectType := ObjectType(p.identifier())
	if err := objectType.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return objectType
}

func (p *parser) messageID() MessageID {
	tok := p.peekToken()
	u := p.uint()
	if u > math.MaxUint32 {
		p.failf(tok.pos, "invalid message ID: %v", u)
	}
	messageID := MessageID(u)
	if err := messageID.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return messageID
}

func (p *parser) signalValueType() SignalValueType {
	tok := p.peekToken()
	signalValueType := SignalValueType(p.uint())
	if err := signalValueType.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return signalValueType
}

func (p *parser) environmentVariableType() EnvironmentVariableType {
	tok := p.peekToken()
	environmentVariableType := EnvironmentVariableType(p.uint())
	if err := environmentVariableType.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return environmentVariableType
}

func (p *parser) attributeValueType() AttributeValueType {
	tok := p.peekToken()
	attributeValueType := AttributeValueType(p.identifier())
	if err := attributeValueType.validate(); err != nil {
		p.failf(tok.pos, "%s", err)
	}
	return attributeValueType
}

func (p *parser) accessType() AccessType {
	tok := p.peekToken()
	accessType := AccessType(p.identifier())
	if err := accessType.validate(); err != nil {
		p.failf(tok.pos, "invalid access type")
	}
	return accessType
}
