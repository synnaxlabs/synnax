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
	"encoding/csv"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/xuri/excelize/v2"
)

// The columns of a field table.
const (
	columnMessage   = "message"
	columnID        = "id"
	columnExtended  = "extended"
	columnLength    = "length"
	columnPeriodMS  = "period_ms"
	columnField     = "field"
	columnStartBit  = "start_bit"
	columnBitLength = "bit_length"
	columnByteOrder = "byte_order"
	columnSigned    = "signed"
	columnFloat     = "float"
	columnScale     = "scale"
	columnOffset    = "offset"
	columnUnits     = "units"
)

const (
	maxStandardID = 0x7FF
	// maxUnzipSize bounds the uncompressed size of an XLSX workbook.
	maxUnzipSize = 64 << 20
	utf8BOM      = "\ufeff"
)

func parseCSV(data []byte) ([]versions.Entry, error) {
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte(utf8BOM))))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		var perr *csv.ParseError
		if errors.As(err, &perr) {
			return nil, dataErrorf("line %d: %s", perr.StartLine, perr.Err)
		}
		return nil, err
	}
	return parseTable(rows)
}

func parseXLSX(data []byte) (entries []versions.Entry, err error) {
	f, err := excelize.OpenReader(
		bytes.NewReader(data),
		excelize.Options{UnzipSizeLimit: maxUnzipSize},
	)
	if err != nil {
		return nil, dataErrorf("invalid workbook: %s", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, dataErrorf("workbook has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, dataErrorf("invalid workbook: %s", err)
	}
	return parseTable(rows)
}

// attr is a message-level value that every row of the message may repeat.
type attr[T comparable] struct {
	value T
	row   int
	set   bool
}

func (a *attr[T]) merge(column string, row int, v T) error {
	if !a.set {
		*a = attr[T]{value: v, row: row, set: true}
		return nil
	}
	if a.value != v {
		return rowErrorf(row, column, "conflicts with the value in row %d", a.row)
	}
	return nil
}

type tableMessage struct {
	entry    versions.MessageEntry
	id       attr[uint32]
	extended attr[bool]
	length   attr[uint16]
	period   attr[telem.TimeSpan]
}

// table reads the rows of a field table. Row numbers count from 1 at the header.
type table struct {
	columns  map[string]int
	messages []*tableMessage
	byName   map[string]*tableMessage
}

func parseTable(rows [][]string) ([]versions.Entry, error) {
	if len(rows) == 0 {
		return nil, dataErrorf("missing header row")
	}
	t := table{
		columns: make(map[string]int, len(rows[0])),
		byName:  make(map[string]*tableMessage),
	}
	if err := t.readHeader(rows[0]); err != nil {
		return nil, err
	}
	for i, cells := range rows[1:] {
		if err := t.readRow(i+2, cells); err != nil {
			return nil, err
		}
	}
	entries := make([]versions.Entry, 0, len(t.messages))
	for _, m := range t.messages {
		if m.id.set {
			extended := m.id.value > maxStandardID
			if m.extended.set {
				extended = m.extended.value
			}
			m.entry.Identifier = &versions.Identifier{Variant: versions.CanIdentifier{
				ID:       m.id.value,
				Extended: extended,
			}}
		} else if m.extended.set {
			return nil, rowErrorf(m.extended.row, columnExtended, "requires an id")
		}
		if m.length.set {
			m.entry.Length = new(m.length.value)
		}
		if m.period.set {
			m.entry.Period = new(m.period.value)
		}
		entries = append(entries, versions.Entry{Variant: m.entry})
	}
	return entries, nil
}

func (t *table) readHeader(cells []string) error {
	known := []string{
		columnMessage, columnID, columnExtended, columnLength, columnPeriodMS,
		columnField, columnStartBit, columnBitLength, columnByteOrder, columnSigned,
		columnFloat, columnScale, columnOffset, columnUnits,
	}
	for i, cell := range cells {
		name := strings.ToLower(strings.TrimSpace(cell))
		if name == "" {
			continue
		}
		if !slices.Contains(known, name) {
			return dataErrorf("row 1: unknown column %q", strings.TrimSpace(cell))
		}
		if _, dup := t.columns[name]; dup {
			return dataErrorf("row 1: duplicate column %q", name)
		}
		t.columns[name] = i
	}
	for _, name := range []string{
		columnMessage, columnField, columnStartBit, columnBitLength,
	} {
		if _, ok := t.columns[name]; !ok {
			return dataErrorf("row 1: missing required column %q", name)
		}
	}
	return nil
}

// cell returns the trimmed value of the named column in cells, or "" when the column
// or cell is absent.
func (t *table) cell(cells []string, column string) string {
	i, ok := t.columns[column]
	if !ok || i >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[i])
}

func (t *table) readRow(row int, cells []string) error {
	if strings.TrimSpace(strings.Join(cells, "")) == "" {
		return nil
	}
	name := t.cell(cells, columnMessage)
	if name == "" {
		return rowErrorf(row, columnMessage, "is required")
	}
	m, ok := t.byName[name]
	if !ok {
		m = &tableMessage{entry: versions.MessageEntry{
			Key:    uuid.New(),
			Name:   name,
			Format: versions.FormatBinary,
		}}
		t.messages = append(t.messages, m)
		t.byName[name] = m
	}
	if err := t.readMessage(row, cells, m); err != nil {
		return err
	}
	field, err := t.readField(row, cells)
	if err != nil {
		return err
	}
	m.entry.Fields = append(m.entry.Fields, versions.Field{Variant: field})
	return nil
}

func (t *table) readMessage(row int, cells []string, m *tableMessage) error {
	if s := t.cell(cells, columnID); s != "" {
		id, err := parseID(s)
		if err != nil {
			return rowErrorf(row, columnID, "invalid id %q", s)
		}
		if err = m.id.merge(columnID, row, id); err != nil {
			return err
		}
	}
	if s := t.cell(cells, columnExtended); s != "" {
		v, err := parseBool(row, columnExtended, s)
		if err != nil {
			return err
		}
		if err = m.extended.merge(columnExtended, row, v); err != nil {
			return err
		}
	}
	if s := t.cell(cells, columnLength); s != "" {
		v, err := strconv.ParseUint(s, 10, 16)
		if err != nil {
			return rowErrorf(row, columnLength, "invalid length %q", s)
		}
		if err = m.length.merge(columnLength, row, uint16(v)); err != nil {
			return err
		}
	}
	if s := t.cell(cells, columnPeriodMS); s != "" {
		ms, err := strconv.ParseFloat(s, 64)
		if err != nil || ms <= 0 {
			return rowErrorf(row, columnPeriodMS, "invalid period %q", s)
		}
		period := telem.TimeSpan(ms * float64(telem.Millisecond))
		if err = m.period.merge(columnPeriodMS, row, period); err != nil {
			return err
		}
	}
	return nil
}

func (t *table) readField(row int, cells []string) (versions.BinaryField, error) {
	f := versions.BinaryField{
		Key:       uuid.New(),
		Name:      t.cell(cells, columnField),
		Scale:     1,
		Units:     t.cell(cells, columnUnits),
		ByteOrder: versions.ByteOrderLittleEndian,
	}
	if f.Name == "" {
		return f, rowErrorf(row, columnField, "is required")
	}
	startBit, err := t.uint(row, cells, columnStartBit, 16)
	if err != nil {
		return f, err
	}
	bitLength, err := t.uint(row, cells, columnBitLength, 8)
	if err != nil {
		return f, err
	}
	f.StartBit, f.BitLength = uint16(startBit), uint8(bitLength)
	switch s := strings.ToLower(t.cell(cells, columnByteOrder)); s {
	case "", "little_endian", "intel":
	case "big_endian", "motorola":
		f.ByteOrder = versions.ByteOrderBigEndian
	default:
		return f, rowErrorf(row, columnByteOrder, "invalid byte order %q", s)
	}
	if s := t.cell(cells, columnSigned); s != "" {
		if f.Signed, err = parseBool(row, columnSigned, s); err != nil {
			return f, err
		}
	}
	if s := t.cell(cells, columnFloat); s != "" {
		if f.Float, err = parseBool(row, columnFloat, s); err != nil {
			return f, err
		}
	}
	if s := t.cell(cells, columnScale); s != "" {
		if f.Scale, err = strconv.ParseFloat(s, 64); err != nil {
			return f, rowErrorf(row, columnScale, "invalid number %q", s)
		}
	}
	if s := t.cell(cells, columnOffset); s != "" {
		if f.Offset, err = strconv.ParseFloat(s, 64); err != nil {
			return f, rowErrorf(row, columnOffset, "invalid number %q", s)
		}
	}
	return f, nil
}

func (t *table) uint(row int, cells []string, column string, bits int) (uint64, error) {
	s := t.cell(cells, column)
	if s == "" {
		return 0, rowErrorf(row, column, "is required")
	}
	v, err := strconv.ParseUint(s, 10, bits)
	if err != nil {
		return 0, rowErrorf(row, column, "invalid integer %q", s)
	}
	return v, nil
}

// parseID parses a CAN identifier in hexadecimal with a 0x prefix, or in decimal.
func parseID(s string) (uint32, error) {
	base := 10
	if lower := strings.ToLower(s); strings.HasPrefix(lower, "0x") {
		s, base = s[2:], 16
	}
	v, err := strconv.ParseUint(s, base, 32)
	return uint32(v), err
}

func parseBool(row int, column, s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "t", "yes", "y", "1":
		return true, nil
	case "false", "f", "no", "n", "0":
		return false, nil
	}
	return false, rowErrorf(row, column, "invalid boolean %q", s)
}

func rowErrorf(row int, column, format string, args ...any) error {
	return dataErrorf(
		"row %d, column %s: "+format,
		append([]any{row, column}, args...)...,
	)
}
