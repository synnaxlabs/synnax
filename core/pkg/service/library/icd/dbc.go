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
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/library/icd/internal/dbc"
	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

const (
	attrCycleTime   = "GenMsgCycleTime"
	attrFrameFormat = "VFrameFormat"
	attrBusType     = "BusType"
	attrProtocol    = "ProtocolType"
	busTypeFD       = "CAN FD"
	protocolJ1939   = "J1939"
	formatJ1939     = "J1939PG"
	formatFDSuffix  = "CAN_FD"
	// independentSignals names the pseudo-message that holds unassigned signals.
	independentSignals = "VECTOR__INDEPENDENT_SIG_MSG"
	// maxMultiplexValues bounds the values the ranges of one SG_MUL_VAL_ line hold.
	maxMultiplexValues = 1 << 16
	// maxCANLength is the length of the longest classic CAN frame.
	maxCANLength = 8
	// minPDU2Format is the smallest PDU format of a J1939 group whose PDU specific
	// byte is a group extension, not a destination address.
	minPDU2Format = 240
	// pdu1Mask and pdu2Mask keep the bits of a J1939 identifier that hold the
	// parameter group number of a PDU1 and a PDU2 group.
	pdu1Mask = 0x03FF0000
	pdu2Mask = 0x03FFFF00
)

// parseDBC parses a DBC file into one message entry per BO_ definition, with a CAN
// identifier and a binary field per signal, followed by one enum entry per value
// table and per signal whose value descriptions match no value table. Extended
// multiplexing (m<N>M markers and SG_MUL_VAL_ ranges) is supported. A message is CAN
// FD when its VFrameFormat says so, when it is longer than 8 bytes, or, in a file with
// no VFrameFormat, when the BusType is CAN FD. A J1939 group matches frames by its
// parameter group number. A signal keeps its [min|max] range unless both are zero.
// Every entry and field gets a new key. Errors are scoped to the "data" path and name
// the line.
func parseDBC(data []byte) ([]versions.Entry, error) {
	defs, err := dbc.Parse(data)
	if err != nil {
		return nil, dataErrorf("%s", err)
	}
	f, err := newDBCFile(defs)
	if err != nil {
		return nil, err
	}
	return f.entries()
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

func newMuxSpec(d *dbc.MultiplexedValuesDef) (muxSpec, error) {
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
		if r.To-r.From >= uint64(maxMultiplexValues-len(values)) {
			return muxSpec{}, dataErrorf(
				"line %d: multiplexor ranges hold more than %d values",
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
	tables        []*dbc.ValueTableDef
	attrs         map[dbc.Identifier]*dbc.AttributeDef
	networkAttrs  set.Set[dbc.Identifier]
	defaults      map[dbc.Identifier]*dbc.AttributeDefaultValueDef
	values        map[attrRef]*dbc.AttributeValueForObjectDef
	networkValues map[dbc.Identifier]*dbc.AttributeValueForObjectDef
	valueTypes    map[signalRef]*dbc.SignalValueTypeDef
	choices       map[signalRef]*dbc.ValueDescriptionsDef
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
		choices:       make(map[signalRef]*dbc.ValueDescriptionsDef),
	}
	for _, def := range defs {
		switch d := def.(type) {
		case *dbc.MessageDef:
			if d.Name != independentSignals {
				f.messages = append(f.messages, d)
			}
		case *dbc.ValueTableDef:
			f.tables = append(f.tables, d)
		case *dbc.AttributeDef:
			switch d.ObjectType {
			case dbc.ObjectTypeMessage:
				f.attrs[d.Name] = d
			case dbc.ObjectTypeUnspecified:
				f.networkAttrs.Add(d.Name)
			}
		case *dbc.AttributeDefaultValueDef:
			f.defaults[d.AttributeName] = d
		case *dbc.AttributeValueForObjectDef:
			switch d.ObjectType {
			case dbc.ObjectTypeMessage:
				f.values[attrRef{name: d.AttributeName, message: d.MessageID}] = d
			case dbc.ObjectTypeUnspecified:
				f.networkValues[d.AttributeName] = d
			}
		case *dbc.SignalValueTypeDef:
			f.valueTypes[signalRef{message: d.MessageID, signal: string(d.SignalName)}] = d
		case *dbc.ValueDescriptionsDef:
			if d.ObjectType == dbc.ObjectTypeSignal {
				f.choices[signalRef{message: d.MessageID, signal: string(d.SignalName)}] = d
			}
		case *dbc.MultiplexedValuesDef:
			ref := signalRef{message: d.MessageID, signal: string(d.SignalName)}
			if _, dup := f.mux[ref]; dup {
				return f, dataErrorf(
					"line %d: duplicate SG_MUL_VAL_ for signal %s",
					d.Pos.Line,
					ref.signal,
				)
			}
			spec, err := newMuxSpec(d)
			if err != nil {
				return f, err
			}
			f.mux[ref] = spec
		}
	}
	return f, nil
}

func (f dbcFile) entries() ([]versions.Entry, error) {
	var (
		entries = make([]versions.Entry, 0, len(f.messages))
		enums   = make([]versions.Entry, 0, len(f.tables))
		tables  = make([]versions.EnumEntry, 0, len(f.tables))
		used    = make(set.Set[signalRef])
	)
	for _, t := range f.tables {
		values, err := enumValues(
			"value table "+string(t.TableName),
			t.ValueDescriptions,
		)
		if err != nil {
			return nil, err
		}
		e := versions.EnumEntry{
			Key:    uuid.New(),
			Name:   string(t.TableName),
			Values: values,
		}
		tables = append(tables, e)
		enums = append(enums, versions.Entry{Variant: e})
	}
	for _, m := range f.messages {
		msg, signalEnums, err := f.message(m, tables, used)
		if err != nil {
			return nil, err
		}
		entries = append(entries, versions.Entry{Variant: msg})
		enums = append(enums, signalEnums...)
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
	return append(entries, enums...), nil
}

func (f dbcFile) message(
	m *dbc.MessageDef,
	tables []versions.EnumEntry,
	used set.Set[signalRef],
) (versions.MessageEntry, []versions.Entry, error) {
	if m.Size > math.MaxUint16 {
		return versions.MessageEntry{}, nil, dataErrorf(
			"line %d: message %s is longer than %d bytes",
			m.Pos.Line,
			m.Name,
			math.MaxUint16,
		)
	}
	period, err := f.period(m)
	if err != nil {
		return versions.MessageEntry{}, nil, err
	}
	payload := versions.BinaryPayload{
		Identifier: &versions.Identifier{Variant: f.identifier(m)},
		Length:     new(uint16(m.Size)),
		Fields:     make([]versions.BinaryField, 0, len(m.Signals)),
	}
	keys := make(map[string]versions.FieldKey, len(m.Signals))
	var switches []string
	for _, s := range m.Signals {
		if _, ok := keys[string(s.Name)]; ok {
			return versions.MessageEntry{}, nil, dataErrorf(
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
	var enums []versions.Entry
	for _, s := range m.Signals {
		ref := signalRef{message: m.MessageID, signal: string(s.Name)}
		field, err := f.field(s, ref, keys[ref.signal])
		if err != nil {
			return versions.MessageEntry{}, nil, err
		}
		if vd, ok := f.choices[ref]; ok {
			values, err := enumValues("signal "+ref.signal, vd.ValueDescriptions)
			if err != nil {
				return versions.MessageEntry{}, nil, err
			}
			key, ok := matchTable(tables, values)
			if !ok {
				e := versions.EnumEntry{
					Key:    uuid.New(),
					Name:   string(m.Name) + "." + ref.signal,
					Values: values,
				}
				enums = append(enums, versions.Entry{Variant: e})
				key = e.Key
			}
			field.Enumeration = &key
		}
		if s.IsMultiplexed {
			mux, values, err := f.multiplexor(m, s, switches, keys, used)
			if err != nil {
				return versions.MessageEntry{}, nil, err
			}
			field.Multiplexor = new(keys[mux])
			field.MultiplexValues = values
		}
		payload.Fields = append(payload.Fields, field)
	}
	return versions.MessageEntry{
		Key:     uuid.New(),
		Name:    string(m.Name),
		Payload: versions.Payload{Variant: payload},
		Period:  period,
	}, enums, nil
}

func (f dbcFile) field(
	s dbc.SignalDef,
	ref signalRef,
	key versions.FieldKey,
) (versions.BinaryField, error) {
	if s.StartBit > math.MaxUint16 || s.Size > math.MaxUint8 {
		return versions.BinaryField{}, dataErrorf(
			"line %d: signal %s does not fit a message",
			s.Pos.Line,
			s.Name,
		)
	}
	isFloat := false
	if vt, ok := f.valueTypes[ref]; ok {
		var bits uint64
		switch vt.SignalValueType {
		case dbc.SignalValueTypeFloat32:
			bits = 32
		case dbc.SignalValueTypeFloat64:
			bits = 64
		}
		if bits != 0 && s.Size != bits {
			return versions.BinaryField{}, dataErrorf(
				"line %d: signal %s is a %d-bit float but has %d bits",
				vt.Pos.Line,
				s.Name,
				bits,
				s.Size,
			)
		}
		isFloat = bits != 0
	}
	byteOrder := versions.ByteOrderLittleEndian
	if s.IsBigEndian {
		byteOrder = versions.ByteOrderBigEndian
	}
	field := versions.BinaryField{
		Key:       key,
		Name:      string(s.Name),
		Scale:     s.Factor,
		Offset:    s.Offset,
		Units:     s.Unit,
		StartBit:  uint16(s.StartBit),
		BitLength: uint8(s.Size),
		ByteOrder: byteOrder,
		Signed:    s.IsSigned && !isFloat,
		Float:     isFloat,
	}
	// A range of [0|0] means the file gives none.
	if s.Minimum != 0 || s.Maximum != 0 {
		field.Min, field.Max = new(s.Minimum), new(s.Maximum)
	}
	return field, nil
}

// multiplexor returns the name of the signal that multiplexes s and the switch values
// for which s is present. An SG_MUL_VAL_ definition names both. Without one, the
// message must have exactly one other multiplexor signal, and s is present for the
// value of its m<N> marker.
func (f dbcFile) multiplexor(
	m *dbc.MessageDef,
	s dbc.SignalDef,
	switches []string,
	keys map[string]versions.FieldKey,
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

// period returns the GenMsgCycleTime of the message, or nil when it is absent or zero.
func (f dbcFile) period(m *dbc.MessageDef) (*telem.TimeSpan, error) {
	def, ok := f.attrs[attrCycleTime]
	if !ok {
		return nil, nil
	}
	var ms float64
	if v, ok := f.values[attrRef{name: attrCycleTime, message: m.MessageID}]; ok {
		ms = numeric(def, v.IntValue, v.FloatValue)
	} else if d, ok := f.defaults[attrCycleTime]; ok {
		ms = numeric(def, d.DefaultIntValue, d.DefaultFloatValue)
	}
	if ms < 0 {
		return nil, dataErrorf(
			"line %d: message %s has a negative cycle time",
			m.Pos.Line,
			m.Name,
		)
	}
	if ms == 0 {
		return nil, nil
	}
	span, ok := msToSpan(ms)
	if !ok {
		return nil, dataErrorf(
			"line %d: message %s has a cycle time out of range: %g ms",
			m.Pos.Line,
			m.Name,
			ms,
		)
	}
	return &span, nil
}

func numeric(def *dbc.AttributeDef, i int64, f float64) float64 {
	if def.Type == dbc.AttributeValueTypeFloat {
		return f
	}
	return float64(i)
}

// identifier returns the CAN identifier of the message.
func (f dbcFile) identifier(m *dbc.MessageDef) versions.CanIdentifier {
	format := f.frameFormat(m.MessageID)
	id := versions.CanIdentifier{
		ID:       m.MessageID.ToCAN(),
		Extended: m.MessageID.IsExtended(),
		Fd: strings.HasSuffix(format, formatFDSuffix) || m.Size > maxCANLength ||
			format == "" && f.networkString(attrBusType) == busTypeFD,
	}
	// A J1939 parameter group always has an extended identifier.
	if id.Extended && (format == formatJ1939 ||
		format == "" && f.networkString(attrProtocol) == protocolJ1939) {
		id.Mask = new(groupMask(id.ID))
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
	def, ok := f.attrs[attrFrameFormat]
	if !ok {
		return ""
	}
	var name string
	var index int64
	if v, ok := f.values[attrRef{name: attrFrameFormat, message: id}]; ok {
		name, index = v.StringValue, v.IntValue
	} else if d, ok := f.defaults[attrFrameFormat]; ok {
		name, index = d.DefaultStringValue, d.DefaultIntValue
	} else {
		return ""
	}
	if def.Type == dbc.AttributeValueTypeEnum {
		return name
	}
	switch index {
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

// enumValues converts the value descriptions of owner, a named signal or value table,
// to enum values. A name that repeats gets its value appended, since enum value names
// must be unique.
func enumValues(
	owner string,
	descs []dbc.ValueDescriptionDef,
) ([]versions.EnumValue, error) {
	values := make([]versions.EnumValue, 0, len(descs))
	names := make(set.Set[string], len(descs))
	for _, d := range descs {
		if d.Value != math.Trunc(d.Value) ||
			d.Value < math.MinInt32 ||
			d.Value > math.MaxInt32 {
			return nil, dataErrorf(
				"line %d: value %v of %s is not an int32",
				d.Pos.Line,
				d.Value,
				owner,
			)
		}
		v := int32(d.Value)
		name := d.Description
		if names.Contains(name) {
			name += " (" + strconv.Itoa(int(v)) + ")"
		}
		names.Add(name)
		values = append(values, versions.EnumValue{Value: v, Name: name})
	}
	return values, nil
}

// matchTable returns the key of the value table that holds exactly the given values.
func matchTable(
	tables []versions.EnumEntry,
	values []versions.EnumValue,
) (versions.EntryKey, bool) {
	want := valueMap(values)
	for _, t := range tables {
		if maps.Equal(valueMap(t.Values), want) {
			return t.Key, true
		}
	}
	return uuid.Nil(), false
}

func valueMap(values []versions.EnumValue) map[int32]string {
	m := make(map[int32]string, len(values))
	for _, v := range values {
		m[v.Value] = v.Name
	}
	return m
}
