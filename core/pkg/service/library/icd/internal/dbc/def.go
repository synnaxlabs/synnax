// Copyright 2019 Einride AB. Modified by Synnax Labs, Inc.
//
// Use of this source code is governed by the MIT license in the LICENSE file in this
// directory.

package dbc

import (
	"strconv"
	"strings"
	"text/scanner"
)

// Def is one definition in a DBC file.
type Def interface {
	// Position returns where the definition starts.
	Position() scanner.Position
	parseFrom(*parser)
}

// VersionDef is the version of a DBC file.
type VersionDef struct {
	Pos     scanner.Position
	Version string
}

var _ Def = (*VersionDef)(nil)

func (d *VersionDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordVersion).pos
	d.Version = p.string()
}

// Position implements Def.
func (d *VersionDef) Position() scanner.Position { return d.Pos }

// NewSymbolsDef lists the keywords a DBC file may use.
type NewSymbolsDef struct {
	Pos     scanner.Position
	Symbols []Keyword
}

var _ Def = (*NewSymbolsDef)(nil)

func (d *NewSymbolsDef) parseFrom(p *parser) {
	p.useWhitespace(significantTab)
	defer p.useWhitespace(defaultWhitespace)
	d.Pos = p.keyword(KeywordNewSymbols).pos
	p.token(':')
	for p.peekToken().typ == '\t' {
		p.token('\t')
		d.Symbols = append(d.Symbols, Keyword(p.identifier()))
	}
}

// Position implements Def.
func (d *NewSymbolsDef) Position() scanner.Position { return d.Pos }

// BitTimingDef is the obsolete baud rate and BTR register settings of a network.
type BitTimingDef struct {
	Pos      scanner.Position
	BaudRate uint64
	BTR1     uint64
	BTR2     uint64
}

var _ Def = (*BitTimingDef)(nil)

func (d *BitTimingDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordBitTiming).pos
	p.token(':')
	d.BaudRate = p.optionalUint()
	if p.peekToken().typ == ':' {
		d.BTR1 = p.optionalUint()
	}
	if p.peekToken().typ == ',' {
		d.BTR2 = p.optionalUint()
	}
}

// Position implements Def.
func (d *BitTimingDef) Position() scanner.Position { return d.Pos }

// NodesDef names the nodes on the network.
type NodesDef struct {
	Pos       scanner.Position
	NodeNames []Identifier
}

var _ Def = (*NodesDef)(nil)

func (d *NodesDef) parseFrom(p *parser) {
	p.useWhitespace(significantNewline)
	defer p.useWhitespace(defaultWhitespace)
	d.Pos = p.keyword(KeywordNodes).pos
	p.token(':')
	for p.peekToken().typ == scanner.Ident {
		d.NodeNames = append(d.NodeNames, p.identifier())
	}
	if p.peekToken().typ != scanner.EOF {
		p.token('\n')
	}
}

// Position implements Def.
func (d *NodesDef) Position() scanner.Position { return d.Pos }

// ValueDescriptionDef names one raw value.
type ValueDescriptionDef struct {
	Pos         scanner.Position
	Value       float64
	Description string
}

var _ Def = (*ValueDescriptionDef)(nil)

func (d *ValueDescriptionDef) parseFrom(p *parser) {
	d.Pos = p.peekToken().pos
	d.Value = p.float()
	d.Description = p.string()
}

// Position implements Def.
func (d *ValueDescriptionDef) Position() scanner.Position { return d.Pos }

// ValueTableDef is a named, global set of value descriptions.
type ValueTableDef struct {
	Pos               scanner.Position
	TableName         Identifier
	ValueDescriptions []ValueDescriptionDef
}

var _ Def = (*ValueTableDef)(nil)

func (d *ValueTableDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordValueTable).pos
	d.TableName = p.identifier()
	for p.peekToken().typ != ';' {
		var desc ValueDescriptionDef
		desc.parseFrom(p)
		d.ValueDescriptions = append(d.ValueDescriptions, desc)
	}
	p.token(';')
}

// Position implements Def.
func (d *ValueTableDef) Position() scanner.Position { return d.Pos }

// MessageDef is a frame on the network and the signals it carries.
type MessageDef struct {
	Pos       scanner.Position
	MessageID MessageID
	Name      Identifier
	// Size is the length of the frame in bytes.
	Size        uint64
	Transmitter Identifier
	Signals     []SignalDef
}

var _ Def = (*MessageDef)(nil)

func (d *MessageDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordMessage).pos
	d.MessageID = p.messageID()
	d.Name = p.identifier()
	p.token(':')
	d.Size = p.uint()
	d.Transmitter = p.identifier()
	for p.peekToken().typ != scanner.EOF && p.peekKeyword() == KeywordSignal {
		var s SignalDef
		s.parseFrom(p)
		d.Signals = append(d.Signals, s)
	}
}

// Position implements Def.
func (d *MessageDef) Position() scanner.Position { return d.Pos }

// SignalDef is a signal within a message.
type SignalDef struct {
	Pos  scanner.Position
	Name Identifier
	// StartBit is the least significant bit of a little-endian signal, or the most
	// significant bit of a big-endian signal.
	StartBit    uint64
	Size        uint64
	IsBigEndian bool
	IsSigned    bool
	// IsMultiplexerSwitch is true for a signal marked M or m<N>M.
	IsMultiplexerSwitch bool
	// IsMultiplexed is true for a signal marked m<N> or m<N>M.
	IsMultiplexed bool
	// MultiplexerSwitch is N for a signal marked m<N> or m<N>M.
	MultiplexerSwitch uint64
	Offset            float64
	Factor            float64
	Minimum           float64
	Maximum           float64
	Unit              string
	Receivers         []Identifier
}

var _ Def = (*SignalDef)(nil)

func (d *SignalDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordSignal).pos
	d.Name = p.identifier()
	if p.peekToken().typ != ':' {
		d.parseMultiplexing(p)
	}
	p.token(':')
	d.StartBit = p.uint()
	p.token('|')
	d.Size = p.uint()
	p.token('@')
	d.IsBigEndian = p.intInRange(0, 1) == 0
	d.IsSigned = p.anyOf('-', '+') == '-'
	p.token('(')
	d.Factor = p.float()
	p.token(',')
	d.Offset = p.float()
	p.token(')')
	p.token('[')
	d.Minimum = p.float()
	p.token('|')
	d.Maximum = p.float()
	p.token(']')
	d.Unit = p.string()
	d.Receivers = append(d.Receivers, p.identifier())
	for p.peekToken().typ == ',' {
		p.token(',')
		d.Receivers = append(d.Receivers, p.identifier())
	}
}

// parseMultiplexing reads the M, m<N>, or m<N>M marker of a signal.
func (d *SignalDef) parseMultiplexing(p *parser) {
	tok := p.nextToken()
	if tok.typ != scanner.Ident {
		p.failf(tok.pos, "expected ident")
	}
	if tok.txt == "M" {
		d.IsMultiplexerSwitch = true
		return
	}
	if tok.txt[0] != 'm' || len(tok.txt) < 2 {
		p.failf(tok.pos, "expected multiplexer")
	}
	value, isSwitch := strings.CutSuffix(tok.txt[1:], "M")
	i, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		p.failf(tok.pos, "invalid multiplexer value")
	}
	d.IsMultiplexed = true
	d.IsMultiplexerSwitch = isSwitch
	d.MultiplexerSwitch = i
}

// Position implements Def.
func (d *SignalDef) Position() scanner.Position { return d.Pos }

// MultiplexRange is an inclusive range of multiplexer switch values.
type MultiplexRange struct {
	From uint64
	To   uint64
}

// MultiplexedValuesDef names the switch of a multiplexed signal and the switch values
// for which the signal is present.
type MultiplexedValuesDef struct {
	Pos        scanner.Position
	MessageID  MessageID
	SignalName Identifier
	SwitchName Identifier
	Ranges     []MultiplexRange
}

var _ Def = (*MultiplexedValuesDef)(nil)

func (d *MultiplexedValuesDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordMultiplexedValues).pos
	d.MessageID = p.messageID()
	d.SignalName = p.identifier()
	d.SwitchName = p.identifier()
	for {
		var r MultiplexRange
		r.From = p.uint()
		p.token('-')
		r.To = p.uint()
		d.Ranges = append(d.Ranges, r)
		if p.peekToken().typ != ',' {
			break
		}
		p.token(',')
	}
	p.token(';')
}

// Position implements Def.
func (d *MultiplexedValuesDef) Position() scanner.Position { return d.Pos }

// SignalValueTypeDef sets the extended value type of a signal.
type SignalValueTypeDef struct {
	Pos             scanner.Position
	MessageID       MessageID
	SignalName      Identifier
	SignalValueType SignalValueType
}

var _ Def = (*SignalValueTypeDef)(nil)

func (d *SignalValueTypeDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordSignalValueType).pos
	d.MessageID = p.messageID()
	d.SignalName = p.identifier()
	// Not in the format, but some files hold a colon here.
	p.optionalToken(':')
	d.SignalValueType = p.signalValueType()
	p.token(';')
}

// Position implements Def.
func (d *SignalValueTypeDef) Position() scanner.Position { return d.Pos }

// MessageTransmittersDef names more than one transmitter of a message.
type MessageTransmittersDef struct {
	Pos          scanner.Position
	MessageID    MessageID
	Transmitters []Identifier
}

var _ Def = (*MessageTransmittersDef)(nil)

func (d *MessageTransmittersDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordMessageTransmitters).pos
	d.MessageID = p.messageID()
	p.token(':')
	for p.peekToken().typ != ';' {
		d.Transmitters = append(d.Transmitters, p.identifier())
		// Not in the format, but some files separate transmitters with commas.
		p.optionalToken(',')
	}
	p.token(';')
}

// Position implements Def.
func (d *MessageTransmittersDef) Position() scanner.Position { return d.Pos }

// ValueDescriptionsDef names raw values of a signal or an environment variable.
type ValueDescriptionsDef struct {
	Pos                     scanner.Position
	ObjectType              ObjectType
	MessageID               MessageID
	SignalName              Identifier
	EnvironmentVariableName Identifier
	ValueDescriptions       []ValueDescriptionDef
}

var _ Def = (*ValueDescriptionsDef)(nil)

func (d *ValueDescriptionsDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordValueDescriptions).pos
	if p.peekToken().typ == scanner.Ident {
		d.ObjectType = ObjectTypeEnvironmentVariable
		d.EnvironmentVariableName = p.identifier()
	} else {
		d.ObjectType = ObjectTypeSignal
		d.MessageID = p.messageID()
		d.SignalName = p.identifier()
	}
	for p.peekToken().typ != ';' {
		var desc ValueDescriptionDef
		desc.parseFrom(p)
		d.ValueDescriptions = append(d.ValueDescriptions, desc)
	}
	p.token(';')
}

// Position implements Def.
func (d *ValueDescriptionsDef) Position() scanner.Position { return d.Pos }

// EnvironmentVariableDef is an environment variable of a bus simulation.
type EnvironmentVariableDef struct {
	Pos          scanner.Position
	Name         Identifier
	Type         EnvironmentVariableType
	Minimum      float64
	Maximum      float64
	Unit         string
	InitialValue float64
	ID           uint64
	AccessType   AccessType
	AccessNodes  []Identifier
}

var _ Def = (*EnvironmentVariableDef)(nil)

func (d *EnvironmentVariableDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordEnvironmentVariable).pos
	d.Name = p.identifier()
	p.token(':')
	d.Type = p.environmentVariableType()
	p.token('[')
	d.Minimum = p.float()
	p.token('|')
	d.Maximum = p.float()
	p.token(']')
	d.Unit = p.string()
	d.InitialValue = p.float()
	d.ID = p.uint()
	d.AccessType = p.accessType()
	d.AccessNodes = append(d.AccessNodes, p.identifier())
	for p.peekToken().typ == ',' {
		p.token(',')
		d.AccessNodes = append(d.AccessNodes, p.identifier())
	}
	p.token(';')
}

// Position implements Def.
func (d *EnvironmentVariableDef) Position() scanner.Position { return d.Pos }

// EnvironmentVariableDataDef makes an environment variable hold DataSize bytes.
type EnvironmentVariableDataDef struct {
	Pos                     scanner.Position
	EnvironmentVariableName Identifier
	DataSize                uint64
}

var _ Def = (*EnvironmentVariableDataDef)(nil)

func (d *EnvironmentVariableDataDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordEnvironmentVariableData).pos
	d.EnvironmentVariableName = p.identifier()
	p.token(':')
	d.DataSize = p.uint()
	p.token(';')
}

// Position implements Def.
func (d *EnvironmentVariableDataDef) Position() scanner.Position { return d.Pos }

// CommentDef is a comment on the network or on one of its objects.
type CommentDef struct {
	Pos                     scanner.Position
	ObjectType              ObjectType
	NodeName                Identifier
	MessageID               MessageID
	SignalName              Identifier
	EnvironmentVariableName Identifier
	Comment                 string
}

var _ Def = (*CommentDef)(nil)

func (d *CommentDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordComment).pos
	d.ObjectType = p.optionalObjectType()
	switch d.ObjectType {
	case ObjectTypeNetworkNode:
		d.NodeName = p.identifier()
	case ObjectTypeMessage:
		d.MessageID = p.messageID()
	case ObjectTypeSignal:
		d.MessageID = p.messageID()
		d.SignalName = p.identifier()
	case ObjectTypeEnvironmentVariable:
		d.EnvironmentVariableName = p.identifier()
	case ObjectTypeUnspecified:
	}
	d.Comment = p.string()
	p.token(';')
}

// Position implements Def.
func (d *CommentDef) Position() scanner.Position { return d.Pos }

// AttributeDef defines a user attribute of one object type.
type AttributeDef struct {
	Pos          scanner.Position
	ObjectType   ObjectType
	Name         Identifier
	Type         AttributeValueType
	MinimumInt   int64
	MaximumInt   int64
	MinimumFloat float64
	MaximumFloat float64
	EnumValues   []string
}

var _ Def = (*AttributeDef)(nil)

func (d *AttributeDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordAttribute).pos
	d.ObjectType = p.optionalObjectType()
	d.Name = p.stringIdentifier()
	d.Type = p.attributeValueType()
	switch d.Type {
	case AttributeValueTypeInt, AttributeValueTypeHex:
		if p.peekToken().typ != ';' {
			d.MinimumInt = p.int()
			d.MaximumInt = p.int()
		}
	case AttributeValueTypeFloat:
		if p.peekToken().typ != ';' {
			d.MinimumFloat = p.float()
			d.MaximumFloat = p.float()
		}
	case AttributeValueTypeEnum:
		d.EnumValues = append(d.EnumValues, p.string())
		for p.peekToken().typ == ',' {
			p.token(',')
			d.EnumValues = append(d.EnumValues, p.string())
		}
	case AttributeValueTypeString:
	}
	p.token(';')
}

// Position implements Def.
func (d *AttributeDef) Position() scanner.Position { return d.Pos }

// AttributeDefaultValueDef is the default value of an attribute.
type AttributeDefaultValueDef struct {
	Pos                scanner.Position
	AttributeName      Identifier
	DefaultIntValue    int64
	DefaultFloatValue  float64
	DefaultStringValue string
}

var _ Def = (*AttributeDefaultValueDef)(nil)

func (d *AttributeDefaultValueDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordAttributeDefault).pos
	d.AttributeName = Identifier(p.string())
	if def := p.attribute(d.AttributeName); def != nil {
		switch def.Type {
		case AttributeValueTypeInt, AttributeValueTypeHex:
			d.DefaultIntValue = p.int()
		case AttributeValueTypeFloat:
			d.DefaultFloatValue = p.float()
		case AttributeValueTypeString:
			d.DefaultStringValue = p.string()
		case AttributeValueTypeEnum:
			d.DefaultStringValue = p.enumValue(def.EnumValues)
		}
	}
	p.token(';')
}

// Position implements Def.
func (d *AttributeDefaultValueDef) Position() scanner.Position { return d.Pos }

// AttributeValueForObjectDef is the value of an attribute for one object.
type AttributeValueForObjectDef struct {
	Pos                     scanner.Position
	AttributeName           Identifier
	ObjectType              ObjectType
	MessageID               MessageID
	SignalName              Identifier
	NodeName                Identifier
	EnvironmentVariableName Identifier
	IntValue                int64
	FloatValue              float64
	StringValue             string
}

var _ Def = (*AttributeValueForObjectDef)(nil)

func (d *AttributeValueForObjectDef) parseFrom(p *parser) {
	d.Pos = p.keyword(KeywordAttributeValue).pos
	d.AttributeName = Identifier(p.string())
	d.ObjectType = p.optionalObjectType()
	switch d.ObjectType {
	case ObjectTypeMessage:
		d.MessageID = p.messageID()
	case ObjectTypeSignal:
		d.MessageID = p.messageID()
		d.SignalName = p.identifier()
	case ObjectTypeNetworkNode:
		d.NodeName = p.identifier()
	case ObjectTypeEnvironmentVariable:
		d.EnvironmentVariableName = p.identifier()
	case ObjectTypeUnspecified:
	}
	if def := p.attribute(d.AttributeName); def != nil {
		switch def.Type {
		case AttributeValueTypeInt, AttributeValueTypeHex:
			d.IntValue = p.int()
		case AttributeValueTypeFloat:
			d.FloatValue = p.float()
		case AttributeValueTypeString:
			d.StringValue = p.string()
		case AttributeValueTypeEnum:
			d.StringValue = p.enumValue(def.EnumValues)
		}
	}
	p.token(';')
}

// Position implements Def.
func (d *AttributeValueForObjectDef) Position() scanner.Position { return d.Pos }

// attribute returns the first attribute definition named name, or nil when none
// precedes the current definition.
func (p *parser) attribute(name Identifier) *AttributeDef {
	for _, def := range p.defs {
		if a, ok := def.(*AttributeDef); ok && a.Name == name {
			return a
		}
	}
	return nil
}

// UnknownDef is a definition the parser does not read. It spans the rest of the line
// its keyword starts.
type UnknownDef struct {
	Pos     scanner.Position
	Keyword Keyword
}

var _ Def = (*UnknownDef)(nil)

func (d *UnknownDef) parseFrom(p *parser) {
	tok := p.peekToken()
	d.Pos = tok.pos
	d.Keyword = Keyword(tok.txt)
	p.discardLine()
}

// Position implements Def.
func (d *UnknownDef) Position() scanner.Position { return d.Pos }
