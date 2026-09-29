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
	"bytes"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.einride.tech/can/pkg/dbc"
)

const (
	attrCycleTime   = "GenMsgCycleTime"
	attrFrameFormat = "VFrameFormat"
	// independentSignals names the pseudo-message that holds unassigned signals.
	independentSignals = "VECTOR__INDEPENDENT_SIG_MSG"
	// maxMultiplexValues bounds the values the ranges of one SG_MUL_VAL_ line hold.
	maxMultiplexValues = 1 << 16
	// keywordMuxValues starts an extended multiplexing definition.
	keywordMuxValues dbc.Keyword = "SG_MUL_VAL_"
)

var (
	// extendedSwitch matches a signal marked m<N>M, which the DBC parser rejects. Its
	// first submatch is the trailing M.
	extendedSwitch = regexp.MustCompile(`^\s*SG_\s+\w+\s+m\d+(M)\s*:`)
	// keyword matches the keyword that starts a definition.
	keyword       = regexp.MustCompile(`^\s*([A-Z][A-Z0-9_]*)(?:[\s:]|$)`)
	muxValuesLine = regexp.MustCompile(
		`^\s*SG_MUL_VAL_\s+(\d+)\s+(\w+)\s+(\w+)\s+(.+?)\s*;\s*$`,
	)
	muxRange = regexp.MustCompile(`^(\d+)\s*-\s*(\d+)$`)
)

// ParseDBC parses a DBC file into one message entry per BO_ definition, with a CAN
// identifier and a binary field per signal, followed by one enum entry per value
// table and per signal whose value descriptions match no value table. Extended
// multiplexing (m<N>M markers and SG_MUL_VAL_ ranges) is supported. Every entry and
// field gets a new key. Errors are scoped to the "data" path and name the line.
func ParseDBC(data []byte) ([]versions.Entry, error) {
	text, ext, err := extractExtensions(data)
	if err != nil {
		return nil, err
	}
	p := dbc.NewParser("", text)
	if perr := p.Parse(); perr != nil {
		pos := perr.Position()
		return nil, dataErrorf(
			"line %d, column %d: %s",
			pos.Line,
			pos.Column,
			perr.Reason(),
		)
	}
	return newDBCFile(p.Defs(), ext).entries()
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

// extensions holds the extended multiplexing syntax the DBC parser does not read.
type extensions struct {
	// switchLines are the lines of signals marked m<N>M.
	switchLines set.Set[int]
	mux         map[signalRef]muxSpec
}

// extractExtensions reads the extended multiplexing syntax from data and returns a copy
// the DBC parser accepts: each m<N>M marker becomes m<N>, and each definition the
// parser does not read, SG_MUL_VAL_ included, is blanked. The parser skips such
// definitions with a bug that can also skip the line after them. The copy keeps every
// byte offset, so parse errors name the right line.
func extractExtensions(data []byte) ([]byte, extensions, error) {
	var (
		text = bytes.Clone(data)
		ext  = extensions{
			switchLines: make(set.Set[int]),
			mux:         make(map[signalRef]muxSpec),
		}
		quoted  = false
		symbols = false
		start   = 0
	)
	for n := 1; start <= len(text); n++ {
		end := bytes.IndexByte(text[start:], '\n')
		if end < 0 {
			end = len(text)
		} else {
			end += start
		}
		line := text[start:end]
		start = end + 1
		wasQuoted := quoted
		quoted = quoted != (countQuotes(line)%2 == 1)
		if wasQuoted {
			continue
		}
		var kw dbc.Keyword
		if m := keyword.FindSubmatch(line); m != nil {
			kw = dbc.Keyword(m[1])
		}
		// The NS_ symbol list is tab-indented keywords, which must stay in place.
		if kw == dbc.KeywordNewSymbols {
			symbols = true
			continue
		}
		if symbols && (len(bytes.TrimSpace(line)) == 0 || line[0] == '\t') {
			continue
		}
		symbols = false
		if err := ext.read(n, kw, line); err != nil {
			return nil, extensions{}, err
		}
	}
	return text, ext, nil
}

// countQuotes counts the unescaped double quotes in line, so that lines inside a
// multi-line comment string are not read as definitions.
func countQuotes(line []byte) int {
	n := 0
	for i, c := range line {
		if c == '"' && (i == 0 || line[i-1] != '\\') {
			n++
		}
	}
	return n
}

func (e extensions) read(n int, kw dbc.Keyword, line []byte) error {
	if m := extendedSwitch.FindSubmatchIndex(line); m != nil {
		line[m[2]] = ' '
		e.switchLines.Add(n)
		return nil
	}
	if kw == keywordMuxValues {
		if err := e.readMuxValues(n, line); err != nil {
			return err
		}
	}
	if kw != "" && !parsed(kw) {
		for i := range line {
			line[i] = ' '
		}
	}
	return nil
}

// parsed reports whether the DBC parser reads definitions that start with kw.
func parsed(kw dbc.Keyword) bool {
	switch kw {
	case dbc.KeywordVersion,
		dbc.KeywordBitTiming,
		dbc.KeywordNewSymbols,
		dbc.KeywordNodes,
		dbc.KeywordMessage,
		dbc.KeywordSignal,
		dbc.KeywordEnvironmentVariable,
		dbc.KeywordComment,
		dbc.KeywordAttribute,
		dbc.KeywordAttributeDefault,
		dbc.KeywordAttributeValue,
		dbc.KeywordValueDescriptions,
		dbc.KeywordValueTable,
		dbc.KeywordSignalValueType,
		dbc.KeywordMessageTransmitters,
		dbc.KeywordEnvironmentVariableData:
		return true
	}
	return false
}

func (e extensions) readMuxValues(n int, line []byte) error {
	m := muxValuesLine.FindSubmatch(line)
	if m == nil {
		return dataErrorf("line %d: invalid SG_MUL_VAL_ definition", n)
	}
	id, err := strconv.ParseUint(string(m[1]), 10, 32)
	if err != nil {
		return dataErrorf("line %d: invalid message id %s", n, m[1])
	}
	ref := signalRef{message: dbc.MessageID(id), signal: string(m[2])}
	var values []int32
	for r := range strings.SplitSeq(string(m[4]), ",") {
		r = strings.TrimSpace(r)
		bounds := muxRange.FindStringSubmatch(r)
		if bounds == nil {
			return dataErrorf("line %d: invalid multiplexor range %q", n, r)
		}
		// The pattern admits only digits, so a parse error is always a range error.
		lo, loErr := strconv.ParseInt(bounds[1], 10, 32)
		hi, hiErr := strconv.ParseInt(bounds[2], 10, 32)
		if loErr != nil || hiErr != nil {
			return dataErrorf(
				"line %d: multiplexor range %q of signal %s exceeds the int32 range",
				n,
				r,
				ref.signal,
			)
		}
		if hi < lo {
			return dataErrorf("line %d: invalid multiplexor range %q", n, r)
		}
		if hi-lo >= int64(maxMultiplexValues-len(values)) {
			return dataErrorf(
				"line %d: multiplexor ranges hold more than %d values",
				n,
				maxMultiplexValues,
			)
		}
		for v := lo; v <= hi; v++ {
			values = append(values, int32(v))
		}
	}
	slices.Sort(values)
	if _, dup := e.mux[ref]; dup {
		return dataErrorf("line %d: duplicate SG_MUL_VAL_ for signal %s", n, ref.signal)
	}
	e.mux[ref] = muxSpec{
		line:       n,
		switchName: string(m[3]),
		values:     slices.Compact(values),
	}
	return nil
}

type attrRef struct {
	name    dbc.Identifier
	message dbc.MessageID
}

// dbcFile indexes the definitions of a parsed DBC file.
type dbcFile struct {
	ext        extensions
	messages   []*dbc.MessageDef
	tables     []*dbc.ValueTableDef
	attrs      map[dbc.Identifier]*dbc.AttributeDef
	defaults   map[dbc.Identifier]*dbc.AttributeDefaultValueDef
	values     map[attrRef]*dbc.AttributeValueForObjectDef
	valueTypes map[signalRef]*dbc.SignalValueTypeDef
	choices    map[signalRef]*dbc.ValueDescriptionsDef
}

func newDBCFile(defs []dbc.Def, ext extensions) dbcFile {
	f := dbcFile{
		ext:        ext,
		attrs:      make(map[dbc.Identifier]*dbc.AttributeDef),
		defaults:   make(map[dbc.Identifier]*dbc.AttributeDefaultValueDef),
		values:     make(map[attrRef]*dbc.AttributeValueForObjectDef),
		valueTypes: make(map[signalRef]*dbc.SignalValueTypeDef),
		choices:    make(map[signalRef]*dbc.ValueDescriptionsDef),
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
			if d.ObjectType == dbc.ObjectTypeMessage {
				f.attrs[d.Name] = d
			}
		case *dbc.AttributeDefaultValueDef:
			f.defaults[d.AttributeName] = d
		case *dbc.AttributeValueForObjectDef:
			if d.ObjectType == dbc.ObjectTypeMessage {
				f.values[attrRef{name: d.AttributeName, message: d.MessageID}] = d
			}
		case *dbc.SignalValueTypeDef:
			f.valueTypes[signalRef{message: d.MessageID, signal: string(d.SignalName)}] = d
		case *dbc.ValueDescriptionsDef:
			if d.ObjectType == dbc.ObjectTypeSignal {
				f.choices[signalRef{message: d.MessageID, signal: string(d.SignalName)}] = d
			}
		}
	}
	return f
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
	refs := slices.SortedFunc(maps.Keys(f.ext.mux), func(a, b signalRef) int {
		return f.ext.mux[a].line - f.ext.mux[b].line
	})
	for _, ref := range refs {
		if !used.Contains(ref) {
			return nil, dataErrorf(
				"line %d: no message with id %d has a multiplexed signal %s",
				f.ext.mux[ref].line,
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
	msg := versions.MessageEntry{
		Key:  uuid.New(),
		Name: string(m.Name),
		Identifier: &versions.Identifier{Variant: versions.CanIdentifier{
			ID:       m.MessageID.ToCAN(),
			Extended: m.MessageID.IsExtended(),
			Fd:       f.fd(m.MessageID),
		}},
		Format: versions.FormatBinary,
		Length: new(uint16(m.Size)),
		Period: period,
		Fields: make([]versions.Field, 0, len(m.Signals)),
	}
	keys := make(map[string]versions.FieldKey, len(m.Signals))
	var switches []string
	for _, s := range m.Signals {
		keys[string(s.Name)] = uuid.New()
		if s.IsMultiplexerSwitch || f.ext.switchLines.Contains(s.Pos.Line) {
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
		msg.Fields = append(msg.Fields, versions.Field{Variant: field})
	}
	return msg, enums, nil
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
	return versions.BinaryField{
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
	}, nil
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
	if spec, ok := f.ext.mux[ref]; ok {
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
	return new(telem.TimeSpan(ms * float64(telem.Millisecond))), nil
}

func numeric(def *dbc.AttributeDef, i int64, f float64) float64 {
	if def.Type == dbc.AttributeValueTypeFloat {
		return f
	}
	return float64(i)
}

// fd reports whether the VFrameFormat of the message is a CAN FD format. An INT
// attribute holds the index of the format in Vector's list, where 14 and 15 are the FD
// formats.
func (f dbcFile) fd(id dbc.MessageID) bool {
	def, ok := f.attrs[attrFrameFormat]
	if !ok {
		return false
	}
	name, index := "", int64(0)
	if v, ok := f.values[attrRef{name: attrFrameFormat, message: id}]; ok {
		name, index = v.StringValue, v.IntValue
	} else if d, ok := f.defaults[attrFrameFormat]; ok {
		name, index = d.DefaultStringValue, d.DefaultIntValue
	}
	if def.Type == dbc.AttributeValueTypeEnum {
		return strings.HasSuffix(name, "CAN_FD")
	}
	return index == 14 || index == 15
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
