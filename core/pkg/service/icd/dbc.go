// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"maps"
	"math"
	"slices"
	"strings"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/icd/internal/dbc"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

const (
	attrFrameFormat = "VFrameFormat"
	attrBRS         = "CANFD_BRS"
	attrBusType     = "BusType"
	attrProtocol    = "ProtocolType"
	attrCycleTime   = "GenMsgCycleTime"
	busTypeFD       = "CAN FD"
	protocolJ1939   = "J1939"
	formatJ1939     = "J1939PG"
	formatFDSuffix  = "CAN_FD"
	// The long name attributes hold names past the 32 characters a DBC name allows.
	attrMessageLongName = "SystemMessageLongSymbol"
	attrSignalLongName  = "SystemSignalLongSymbol"
	// independentSignals names the pseudo-message that holds unassigned signals.
	independentSignals = "VECTOR__INDEPENDENT_SIG_MSG"
	// maxMultiplexValues bounds the switch values the SG_MUL_VAL_ lines of a file hold.
	maxMultiplexValues = 1 << 16
	// minPDU2Format is the smallest PDU format of a J1939 group whose PDU specific
	// byte is a group extension, not a destination address.
	minPDU2Format = 240
	// pdu1Mask and pdu2Mask keep the bits of a J1939 identifier that hold the
	// parameter group number of a PDU1 and a PDU2 group.
	pdu1Mask = 0x03FF0000
	pdu2Mask = 0x03FFFF00
	// sourceMask and destinationMask keep the source address and the PDU1 destination
	// address of a J1939 identifier.
	sourceMask      = 0x000000FF
	destinationMask = 0x0000FF00
)

// parseDBC parses a DBC file into one message per BO_ definition, with a CAN identifier
// and a binary field per signal. Value tables and value descriptions are skipped.
// Extended multiplexing (m<N>M markers and SG_MUL_VAL_ ranges) is supported. A message
// is CAN FD when its VFrameFormat says so, when it is longer than 8 bytes, or, in a
// file with no VFrameFormat, when the BusType is CAN FD. A CAN FD message switches
// bitrates when its CANFD_BRS is 1. A J1939 group matches frames by its parameter group
// number, and also by its addresses when other messages share the group. A signal keeps
// its [min|max] range unless both are zero. A message takes its GenMsgCycleTime as its
// cycle time unless that is zero. Messages and signals take their long names when the
// file gives them. Every message and field gets a new key. Errors are scoped to the
// "data" path and name the line.
func parseDBC(data []byte) ([]Message, error) {
	defs, err := dbc.Parse(data)
	if err != nil {
		return nil, dataErrorf("%s", err)
	}
	f, err := newDBCFile(defs)
	if err != nil {
		return nil, err
	}
	return f.toMessages()
}

type signalRef struct {
	message dbc.MessageID
	signal  string
}

// muxSpec is one SG_MUL_VAL_ definition.
type muxSpec struct {
	// line is the line of the definition.
	line int
	// switchName is the signal that multiplexes the defined signal.
	switchName string
	// values are the sorted, unique switch values the ranges hold.
	values []int32
}

// newMuxSpec parses d. used counts the switch values of the earlier SG_MUL_VAL_ lines.
func newMuxSpec(d *dbc.MultiplexedValuesDef, used int) (muxSpec, error) {
	line := d.Pos.Line
	var values []int32
	for _, r := range d.Ranges {
		if r.To < r.From {
			return muxSpec{}, dataErrorf(
				"line %d: invalid multiplexor range \"%d-%d\"",
				line,
				r.From,
				r.To,
			)
		}
		if r.To > math.MaxInt32 {
			return muxSpec{}, dataErrorf(
				"line %d: multiplexor range \"%d-%d\" of signal %s exceeds the int32 "+
					"range",
				line,
				r.From,
				r.To,
				d.SignalName,
			)
		}
		if r.To-r.From >= uint64(maxMultiplexValues-used-len(values)) {
			return muxSpec{}, dataErrorf(
				"line %d: the multiplexor ranges of the file hold more than %d values",
				line,
				maxMultiplexValues,
			)
		}
		for v := r.From; v <= r.To; v++ {
			values = append(values, int32(v))
		}
	}
	slices.Sort(values)
	return muxSpec{
		line:       line,
		switchName: string(d.SwitchName),
		values:     slices.Compact(values),
	}, nil
}

type attrRef struct {
	name    dbc.Identifier
	message dbc.MessageID
}

// dbcFile indexes the definitions of a parsed DBC file.
type dbcFile struct {
	mux           map[signalRef]muxSpec
	messages      []*dbc.MessageDef
	attrs         map[dbc.Identifier]*dbc.AttributeDef
	networkAttrs  set.Set[dbc.Identifier]
	defaults      map[dbc.Identifier]*dbc.AttributeDefaultValueDef
	values        map[attrRef]*dbc.AttributeValueForObjectDef
	networkValues map[dbc.Identifier]*dbc.AttributeValueForObjectDef
	valueTypes    map[signalRef]*dbc.SignalValueTypeDef
	messageNames  map[dbc.MessageID]string
	signalNames   map[signalRef]string
	// groupMasks holds the identifier mask of each J1939 message.
	groupMasks map[dbc.MessageID]uint32
}

func newDBCFile(defs []dbc.Def) (dbcFile, error) {
	f := dbcFile{
		mux:           make(map[signalRef]muxSpec),
		attrs:         make(map[dbc.Identifier]*dbc.AttributeDef),
		networkAttrs:  make(set.Set[dbc.Identifier]),
		defaults:      make(map[dbc.Identifier]*dbc.AttributeDefaultValueDef),
		values:        make(map[attrRef]*dbc.AttributeValueForObjectDef),
		networkValues: make(map[dbc.Identifier]*dbc.AttributeValueForObjectDef),
		valueTypes:    make(map[signalRef]*dbc.SignalValueTypeDef),
		messageNames:  make(map[dbc.MessageID]string),
		signalNames:   make(map[signalRef]string),
	}
	var muxValues int
	for _, def := range defs {
		switch d := def.(type) {
		case *dbc.MessageDef:
			if d.Name != independentSignals {
				f.messages = append(f.messages, d)
			}
		case *dbc.AttributeDef:
			switch d.ObjectType {
			case dbc.ObjectTypeMessage:
				f.attrs[d.Name] = d
			case dbc.ObjectTypeUnspecified:
				f.networkAttrs.Add(d.Name)
			case dbc.ObjectTypeNetworkNode,
				dbc.ObjectTypeSignal,
				dbc.ObjectTypeEnvironmentVariable:
			}
		case *dbc.AttributeDefaultValueDef:
			f.defaults[d.AttributeName] = d
		case *dbc.AttributeValueForObjectDef:
			f.addAttrValue(d)
		case *dbc.SignalValueTypeDef:
			f.valueTypes[signalRef{message: d.MessageID, signal: string(d.SignalName)}] = d
		case *dbc.MultiplexedValuesDef:
			added, err := f.addMux(d, muxValues)
			if err != nil {
				return dbcFile{}, err
			}
			muxValues += added
		}
	}
	f.groupMasks = f.maskGroups()
	return f, nil
}

// addAttrValue adds d, the value of an attribute of a message, a signal, or the
// network, to f.
func (f dbcFile) addAttrValue(d *dbc.AttributeValueForObjectDef) {
	switch d.ObjectType {
	case dbc.ObjectTypeMessage:
		f.values[attrRef{name: d.AttributeName, message: d.MessageID}] = d
		if d.AttributeName == attrMessageLongName && d.StringValue != "" {
			f.messageNames[d.MessageID] = d.StringValue
		}
	case dbc.ObjectTypeSignal:
		if d.AttributeName == attrSignalLongName && d.StringValue != "" {
			ref := signalRef{message: d.MessageID, signal: string(d.SignalName)}
			f.signalNames[ref] = d.StringValue
		}
	case dbc.ObjectTypeUnspecified:
		f.networkValues[d.AttributeName] = d
	case dbc.ObjectTypeNetworkNode, dbc.ObjectTypeEnvironmentVariable:
	}
}

// addMux adds the multiplexor spec of d to f. muxValues is the number of switch
// values in the specs added before. addMux returns the number of switch values of d.
func (f dbcFile) addMux(d *dbc.MultiplexedValuesDef, muxValues int) (int, error) {
	ref := signalRef{message: d.MessageID, signal: string(d.SignalName)}
	if _, dup := f.mux[ref]; dup {
		return 0, dataErrorf(
			"line %d: duplicate SG_MUL_VAL_ for signal %s",
			d.Pos.Line,
			ref.signal,
		)
	}
	spec, err := newMuxSpec(d, muxValues)
	if err != nil {
		return 0, err
	}
	f.mux[ref] = spec
	return len(spec.values), nil
}

type maskedID struct{ id, mask uint32 }

// maskGroups returns the identifier mask of each J1939 message. Messages that share a
// parameter group match by their source address too, and PDU1 messages that also share
// a sender match by their destination address.
func (f dbcFile) maskGroups() map[dbc.MessageID]uint32 {
	masks := make(map[dbc.MessageID]uint32)
	for _, m := range f.messages {
		if f.isJ1939(m.MessageID) {
			masks[m.MessageID] = groupMask(m.MessageID.ToCAN())
		}
	}
	for _, address := range []uint32{sourceMask, destinationMask} {
		counts := make(map[maskedID]int, len(masks))
		for id, mask := range masks {
			counts[maskedID{id: id.ToCAN() & mask, mask: mask}]++
		}
		for id, mask := range masks {
			if counts[maskedID{id: id.ToCAN() & mask, mask: mask}] > 1 {
				masks[id] = mask | address
			}
		}
	}
	return masks
}

// isJ1939 returns true when the message is a J1939 parameter group, which always has
// an extended identifier.
func (f dbcFile) isJ1939(id dbc.MessageID) bool {
	format := f.frameFormat(id)
	return id.IsExtended() && (format == formatJ1939 ||
		format == "" && f.networkString(attrProtocol) == protocolJ1939)
}

func (f dbcFile) toMessages() ([]Message, error) {
	messages := make([]Message, 0, len(f.messages))
	used := make(set.Set[signalRef])
	for _, m := range f.messages {
		msg, err := f.message(m, used)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	refs := slices.SortedFunc(maps.Keys(f.mux), func(a, b signalRef) int {
		return f.mux[a].line - f.mux[b].line
	})
	for _, ref := range refs {
		if !used.Contains(ref) {
			return nil, dataErrorf(
				"line %d: no message with id %d has a multiplexed signal %s",
				f.mux[ref].line,
				ref.message,
				ref.signal,
			)
		}
	}
	return messages, nil
}

func (f dbcFile) message(
	m *dbc.MessageDef,
	used set.Set[signalRef],
) (Message, error) {
	if err := validateMessage(m); err != nil {
		return Message{}, err
	}
	payload := BinaryPayload{
		Identifier: &Identifier{Variant: f.identifier(m)},
		Length:     new(uint16(m.Size)),
		Fields:     make([]BinaryField, 0, len(m.Signals)),
	}
	name := string(m.Name)
	if long, ok := f.messageNames[m.MessageID]; ok {
		name = long
	}
	cycleTime, err := f.cycleTime(m)
	if err != nil {
		return Message{}, err
	}
	keys, switches, err := signalKeys(m)
	if err != nil {
		return Message{}, err
	}
	for _, s := range m.Signals {
		ref := signalRef{message: m.MessageID, signal: string(s.Name)}
		field, err := f.field(m, s, ref, keys[ref.signal])
		if err != nil {
			return Message{}, err
		}
		// Some tools mark a multiplexed switch with M alone and name its multiplexor
		// in an SG_MUL_VAL_ line.
		_, named := f.mux[ref]
		if s.IsMultiplexed || (s.IsMultiplexerSwitch && named) {
			mux, values, err := f.multiplexor(m, s, switches, keys, used)
			if err != nil {
				return Message{}, err
			}
			base := field.Base()
			base.Multiplexor = new(keys[mux])
			base.MultiplexValues = values
			field.SetBase(base)
		}
		payload.Fields = append(payload.Fields, field)
	}
	return Message{
		Key:       uuid.New(),
		Name:      name,
		Payload:   Payload{Variant: payload},
		CycleTime: cycleTime,
	}, nil
}

// cycleTime returns the GenMsgCycleTime of m, or nil when it is absent or zero.
func (f dbcFile) cycleTime(m *dbc.MessageDef) (*telem.TimeSpan, error) {
	v, ok := f.attr(attrCycleTime, m.MessageID)
	if !ok {
		return nil, nil
	}
	ms := v.number()
	if ms == 0 {
		return nil, nil
	}
	span := ms * float64(telem.Millisecond)
	if !(span >= float64(telem.Nanosecond) && span < float64(telem.TimeSpanMax)) {
		return nil, dataErrorf(
			"line %d: message %s has a cycle time out of range: %g ms",
			v.line,
			m.Name,
			ms,
		)
	}
	return new(telem.TimeSpan(span)), nil
}

// validateMessage checks that the identifier and the size of m fit an ICD message.
func validateMessage(m *dbc.MessageDef) error {
	if err := m.MessageID.Validate(); err != nil {
		return dataErrorf("line %d: %s in message %s", m.Pos.Line, err, m.Name)
	}
	if m.Size > math.MaxUint16 {
		return dataErrorf(
			"line %d: message %s is longer than %d bytes",
			m.Pos.Line,
			m.Name,
			math.MaxUint16,
		)
	}
	return nil
}

// signalKeys returns a new field key for each signal of m, by name, and the names of
// its multiplexor switches.
func signalKeys(m *dbc.MessageDef) (map[string]FieldKey, []string, error) {
	keys := make(map[string]FieldKey, len(m.Signals))
	var switches []string
	for _, s := range m.Signals {
		if _, ok := keys[string(s.Name)]; ok {
			return nil, nil, dataErrorf(
				"line %d: message %s has more than one signal named %s",
				s.Pos.Line,
				m.Name,
				s.Name,
			)
		}
		keys[string(s.Name)] = uuid.New()
		if s.IsMultiplexerSwitch {
			switches = append(switches, string(s.Name))
		}
	}
	return keys, switches, nil
}

func (f dbcFile) field(
	m *dbc.MessageDef,
	s dbc.SignalDef,
	ref signalRef,
	key FieldKey,
) (BinaryField, error) {
	if err := validateSignal(s); err != nil {
		return BinaryField{}, err
	}
	isFloat, err := f.isFloat(s, ref)
	if err != nil {
		return BinaryField{}, err
	}
	byteOrder := ByteOrderLittleEndian
	if s.IsBigEndian {
		byteOrder = ByteOrderBigEndian
	}
	name := string(s.Name)
	if long, ok := f.signalNames[ref]; ok {
		name = long
	}
	base := BinaryBaseField{
		Key:       key,
		Name:      name,
		Scale:     s.Factor,
		Offset:    s.Offset,
		Units:     s.Unit,
		StartBit:  uint16(s.StartBit),
		BitLength: uint8(s.Size),
		ByteOrder: byteOrder,
	}
	if lastByte(base) >= int(m.Size) {
		return BinaryField{}, dataErrorf(
			"line %d: signal %s extends past the %d-byte message %s",
			s.Pos.Line,
			s.Name,
			m.Size,
			m.Name,
		)
	}
	// A range of [0|0] means the file gives none.
	if s.Minimum != 0 || s.Maximum != 0 {
		base.Min, base.Max = new(s.Minimum), new(s.Maximum)
	}
	switch {
	case isFloat:
		return BinaryField{Variant: FloatBinaryField{BinaryBaseField: base}}, nil
	case s.IsSigned:
		return BinaryField{Variant: IntegerBinaryField{BinaryBaseField: base}}, nil
	default:
		return BinaryField{Variant: UnsignedBinaryField{BinaryBaseField: base}}, nil
	}
}

// validateSignal checks that the start bit and the size of s fit a binary field.
func validateSignal(s dbc.SignalDef) error {
	if s.StartBit > math.MaxUint16 {
		return dataErrorf(
			"line %d: signal %s does not fit a message",
			s.Pos.Line,
			s.Name,
		)
	}
	if s.Size < 1 || s.Size > maxBitLength {
		return dataErrorf(
			"line %d: signal %s has %d bits, not 1 to %d",
			s.Pos.Line,
			s.Name,
			s.Size,
			maxBitLength,
		)
	}
	return nil
}

// isFloat reports whether the value type of s, the signal at ref, is a float. A float
// value type must match the size of s.
func (f dbcFile) isFloat(s dbc.SignalDef, ref signalRef) (bool, error) {
	vt, ok := f.valueTypes[ref]
	if !ok {
		return false, nil
	}
	var bits uint64
	switch vt.SignalValueType {
	case dbc.SignalValueTypeFloat32:
		bits = 32
	case dbc.SignalValueTypeFloat64:
		bits = 64
	case dbc.SignalValueTypeInt:
	}
	if bits != 0 && s.Size != bits {
		return false, dataErrorf(
			"line %d: signal %s is a %d-bit float but has %d bits",
			vt.Pos.Line,
			s.Name,
			bits,
			s.Size,
		)
	}
	return bits != 0, nil
}

// multiplexor returns the name of the signal that multiplexes s and the switch values
// for which s is present. An SG_MUL_VAL_ definition names both. Without one, the
// message must have exactly one other multiplexor signal, and s is present for the
// value of its m<N> marker.
func (f dbcFile) multiplexor(
	m *dbc.MessageDef,
	s dbc.SignalDef,
	switches []string,
	keys map[string]FieldKey,
	used set.Set[signalRef],
) (string, []int32, error) {
	ref := signalRef{message: m.MessageID, signal: string(s.Name)}
	if spec, ok := f.mux[ref]; ok {
		used.Add(ref)
		if _, ok := keys[spec.switchName]; !ok {
			return "", nil, dataErrorf(
				"line %d: multiplexor %s is not a signal of message %s",
				spec.line,
				spec.switchName,
				m.Name,
			)
		}
		return spec.switchName, spec.values, nil
	}
	others := slices.DeleteFunc(slices.Clone(switches), func(name string) bool {
		return name == ref.signal
	})
	if len(others) != 1 {
		return "", nil, dataErrorf(
			"line %d: signal %s needs an SG_MUL_VAL_ definition to name its multiplexor",
			s.Pos.Line,
			s.Name,
		)
	}
	if s.MultiplexerSwitch > math.MaxInt32 {
		return "", nil, dataErrorf(
			"line %d: multiplexor value of signal %s exceeds the int32 range",
			s.Pos.Line,
			s.Name,
		)
	}
	return others[0], []int32{int32(s.MultiplexerSwitch)}, nil
}

// attrValue is the value of a message attribute. An ENUM attribute holds the name of
// its value in str.
type attrValue struct {
	typ dbc.AttributeValueType
	str string
	i   int64
	f   float64
	// line is the line of the definition that gives the value.
	line int
}

func (v attrValue) number() float64 {
	if v.typ == dbc.AttributeValueTypeFloat {
		return v.f
	}
	return float64(v.i)
}

// attr returns the value the message gives the named attribute, or the attribute's
// default. It returns false when the file defines the attribute without either.
func (f dbcFile) attr(name dbc.Identifier, id dbc.MessageID) (attrValue, bool) {
	def, ok := f.attrs[name]
	if !ok {
		return attrValue{}, false
	}
	if v, ok := f.values[attrRef{name: name, message: id}]; ok {
		return attrValue{
			typ:  def.Type,
			str:  v.StringValue,
			i:    v.IntValue,
			f:    v.FloatValue,
			line: v.Pos.Line,
		}, true
	}
	if d, ok := f.defaults[name]; ok {
		return attrValue{
			typ:  def.Type,
			str:  d.DefaultStringValue,
			i:    d.DefaultIntValue,
			f:    d.DefaultFloatValue,
			line: d.Pos.Line,
		}, true
	}
	return attrValue{}, false
}

// identifier returns the CAN identifier of the message.
func (f dbcFile) identifier(m *dbc.MessageDef) CanIdentifier {
	format := f.frameFormat(m.MessageID)
	id := CanIdentifier{
		ID:       m.MessageID.ToCAN(),
		Extended: m.MessageID.IsExtended(),
		Fd: strings.HasSuffix(format, formatFDSuffix) || m.Size > maxCANLength ||
			format == "" && f.networkString(attrBusType) == busTypeFD,
	}
	if v, ok := f.attr(attrBRS, m.MessageID); ok && id.Fd {
		id.BitrateSwitched = v.str == "1" ||
			v.typ != dbc.AttributeValueTypeEnum && v.number() == 1
	}
	if mask, ok := f.groupMasks[m.MessageID]; ok {
		id.Mask = new(mask)
	}
	return id
}

// groupMask returns the mask that matches a J1939 identifier by its parameter group
// number. The mask drops the priority and the source address, and for a PDU1 group
// also the destination address.
func groupMask(id uint32) uint32 {
	if pduFormat := id >> 16 & 0xFF; pduFormat < minPDU2Format {
		return pdu1Mask
	}
	return pdu2Mask
}

// frameFormat returns the name of the message's VFrameFormat, or "" when the file
// gives it none. An INT attribute holds the index of the format in Vector's list.
func (f dbcFile) frameFormat(id dbc.MessageID) string {
	v, ok := f.attr(attrFrameFormat, id)
	if !ok {
		return ""
	}
	if v.typ == dbc.AttributeValueTypeEnum {
		return v.str
	}
	switch v.i {
	case 0:
		return "StandardCAN"
	case 1:
		return "ExtendedCAN"
	case 3:
		return formatJ1939
	case 14:
		return "StandardCAN_FD"
	case 15:
		return "ExtendedCAN_FD"
	}
	return "reserved"
}

// networkString returns the value of the network attribute with the given name, or its
// default when the file sets none.
func (f dbcFile) networkString(name dbc.Identifier) string {
	if !f.networkAttrs.Contains(name) {
		return ""
	}
	if v, ok := f.networkValues[name]; ok {
		return v.StringValue
	}
	if d, ok := f.defaults[name]; ok {
		return d.DefaultStringValue
	}
	return ""
}
