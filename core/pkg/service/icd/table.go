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
	"math"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/synnaxlabs/x/errors"
	"github.com/xuri/excelize/v2"
)

// The columns of a field table.
const (
	columnMessage   = "message"
	columnID        = "id"
	columnExtended  = "extended"
	columnLength    = "length"
	columnField     = "field"
	columnStartBit  = "start_bit"
	columnBitLength = "bit_length"
	columnByteOrder = "byte_order"
	columnSigned    = "signed"
	columnFloat     = "float"
	columnScale     = "scale"
	columnOffset    = "offset"
	columnUnits     = "units"
	columnMin       = "min"
	columnMax       = "max"
)

const (
	// maxUnzipSize bounds the uncompressed size of an XLSX workbook.
	maxUnzipSize = 64 << 20
	utf8BOM      = "\ufeff"
)

func parseCSV(data []byte) ([]Message, error) {
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

func parseXLSX(data []byte) (messages []Message, err error) {
	f, err := excelize.OpenReader(
		bytes.NewReader(data),
		excelize.Options{UnzipSizeLimit: maxUnzipSize},
	)
	if err != nil {
		return nil, dataErrorf("invalid workbook: %s", err)
	}
	defer func() {
		if err = errors.Join(err, f.Close()); err != nil {
			messages = nil
		}
	}()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, dataErrorf("workbook has no sheets")
	}
	rows, err := readRows(f, sheets[0])
	if err != nil {
		return nil, err
	}
	return parseTable(rows)
}

// readRows returns the raw cell values of every row of the sheet, so a number format
// never rounds a number. Unlike excelize's GetRows, it returns an error for a row it
// cannot read instead of dropping that row and every row after it.
func readRows(f *excelize.File, sheet string) (table [][]string, err error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, dataErrorf("invalid workbook: %s", err)
	}
	// Without a temporary file, Close returns rows.Error, which err already holds.
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			table, err = nil, closeErr
		}
	}()
	for row := 1; rows.Next(); row++ {
		cells, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, dataErrorf("row %d: %s", row, err)
		}
		table = append(table, cells)
	}
	if err = rows.Error(); err != nil {
		return nil, dataErrorf("invalid workbook: %s", err)
	}
	return table, nil
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
	message  Message
	payload  BinaryPayload
	id       attr[uint32]
	extended attr[bool]
	length   attr[uint16]
}

// table reads the rows of a field table. Row numbers count from 1 at the header.
type table struct {
	columns  map[string]int
	messages []*tableMessage
	byName   map[string]*tableMessage
}

func parseTable(rows [][]string) ([]Message, error) {
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
	messages := make([]Message, 0, len(t.messages))
	for _, m := range t.messages {
		if m.id.set {
			extended := m.id.value > maxStandardCANID
			if m.extended.set {
				extended = m.extended.value
			}
			m.payload.Identifier = &Identifier{Variant: CanIdentifier{
				ID:       m.id.value,
				Extended: extended,
			}}
		} else if m.extended.set {
			return nil, rowErrorf(m.extended.row, columnExtended, "requires an id")
		}
		if m.length.set {
			m.payload.Length = new(m.length.value)
		}
		m.message.Payload = Payload{Variant: m.payload}
		messages = append(messages, m.message)
	}
	return messages, nil
}

func (t *table) readHeader(cells []string) error {
	known := []string{
		columnMessage, columnID, columnExtended, columnLength, columnField,
		columnStartBit, columnBitLength, columnByteOrder, columnSigned, columnFloat,
		columnScale, columnOffset, columnUnits, columnMin, columnMax,
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
		m = &tableMessage{message: Message{Key: uuid.New(), Name: name}}
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
	m.payload.Fields = append(m.payload.Fields, field)
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
	return nil
}

func (t *table) readField(row int, cells []string) (BinaryField, error) {
	f := BinaryField{
		Key:       uuid.New(),
		Name:      t.cell(cells, columnField),
		Scale:     1,
		Units:     t.cell(cells, columnUnits),
		ByteOrder: ByteOrderLittleEndian,
	}
	if f.Name == "" {
		return BinaryField{}, rowErrorf(row, columnField, "is required")
	}
	startBit, err := t.uint(row, cells, columnStartBit, 16)
	if err != nil {
		return BinaryField{}, err
	}
	bitLength, err := t.uint(row, cells, columnBitLength, 8)
	if err != nil {
		return BinaryField{}, err
	}
	f.StartBit, f.BitLength = uint16(startBit), uint8(bitLength)
	switch s := strings.ToLower(t.cell(cells, columnByteOrder)); s {
	case "", "little_endian", "intel":
	case "big_endian", "motorola":
		f.ByteOrder = ByteOrderBigEndian
	default:
		return BinaryField{}, rowErrorf(
			row,
			columnByteOrder,
			"invalid byte order %q",
			s,
		)
	}
	if s := t.cell(cells, columnSigned); s != "" {
		if f.Signed, err = parseBool(row, columnSigned, s); err != nil {
			return BinaryField{}, err
		}
	}
	if s := t.cell(cells, columnFloat); s != "" {
		if f.Float, err = parseBool(row, columnFloat, s); err != nil {
			return BinaryField{}, err
		}
	}
	scale, err := t.number(row, cells, columnScale)
	if err != nil {
		return BinaryField{}, err
	}
	if scale != nil {
		f.Scale = *scale
	}
	offset, err := t.number(row, cells, columnOffset)
	if err != nil {
		return BinaryField{}, err
	}
	if offset != nil {
		f.Offset = *offset
	}
	if f.Min, err = t.number(row, cells, columnMin); err != nil {
		return BinaryField{}, err
	}
	if f.Max, err = t.number(row, cells, columnMax); err != nil {
		return BinaryField{}, err
	}
	return f, nil
}

// number returns the finite number in the named column, or nil when the cell is empty.
func (t *table) number(row int, cells []string, column string) (*float64, error) {
	s := t.cell(cells, column)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, rowErrorf(row, column, "invalid number %q", s)
	}
	return &v, nil
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
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
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
