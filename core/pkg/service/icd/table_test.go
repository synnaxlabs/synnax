// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"archive/zip"
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/xuri/excelize/v2"
)

// withoutKeys returns messages with every message and field key cleared.
func withoutKeys(messages []icd.Message) []icd.Message {
	out := make([]icd.Message, len(messages))
	for i, m := range messages {
		m.Key = uuid.Nil()
		m.Payload.UpdateFieldBases(func(b *icd.BaseField) { b.Key = uuid.Nil() })
		out[i] = m
	}
	return out
}

func tableMessage(
	name string,
	id *icd.Identifier,
	length *uint16,
	fields ...icd.BinaryField,
) icd.Message {
	return icd.Message{
		Name: name,
		Payload: icd.Payload{Variant: icd.BinaryPayload{
			Identifier: id,
			Length:     length,
			Fields:     fields,
		}},
	}
}

func canID(id uint32, extended bool) *icd.Identifier {
	return &icd.Identifier{
		Variant: icd.CanIdentifier{ID: id, Extended: extended},
	}
}

// tableMessages are the messages of tableRows.
var tableMessages = []icd.Message{
	tableMessage(
		"Engine",
		canID(0x100, false),
		new(uint16(8)),
		icd.BinaryField{
			Name: "rpm", Scale: 0.25, Units: "rpm",
			BitLength: 16,
			ByteOrder: icd.ByteOrderLittleEndian,
		},
		icd.BinaryField{
			Name:      "temp",
			Scale:     1,
			Offset:    -40,
			Units:     "degC",
			Min:       new(-40.0),
			Max:       new(87.5),
			StartBit:  16,
			BitLength: 8,
			ByteOrder: icd.ByteOrderLittleEndian,
			Signed:    true,
		},
	),
	tableMessage(
		"Battery",
		canID(0x18FEF100, true),
		nil,
		icd.BinaryField{
			Name: "voltage", Scale: 0.01, Units: "V",
			StartBit:  7,
			BitLength: 16,
			ByteOrder: icd.ByteOrderBigEndian,
		},
	),
	tableMessage(
		"Raw",
		nil,
		nil,
		icd.BinaryField{
			Name: "value", Scale: 1,
			BitLength: 32,
			ByteOrder: icd.ByteOrderLittleEndian,
			Float:     true,
		},
	),
}

var tableRows = [][]string{
	{
		"Message",
		"ID",
		"Extended",
		"Length",
		"Field",
		"Start_Bit",
		"Bit_Length",
		"Byte_Order",
		"Signed",
		"Float",
		"Scale",
		"Offset",
		"Units",
		"Min",
		"Max",
	},
	{
		"Engine", "0x100", "", "8", "rpm", "0", "16", "little_endian", "false", "false",
		"0.25", "0", "rpm",
	},
	{"", "", "", "", "", "", "", "", "", "", "", "", ""},
	{
		"Engine", "256", "", "", "temp", "16", "8", "intel", "true", "", "1", "-40",
		"degC", "-40", "87.5",
	},
	{
		"Battery",
		"419361024",
		"",
		"",
		"voltage",
		"7",
		"16",
		"Motorola",
		"",
		"",
		"0.01",
		"",
		"V",
	},
	{"Raw", "", "", "", "value", "0", "32", "", "", "yes"},
}

func toCSV(rows [][]string) []byte {
	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = strings.Join(r, ",")
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

// cellValue returns s as a spreadsheet stores it: a number, a boolean, or text.
func cellValue(s string) any {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}
	return s
}

// writeWorkbook returns a workbook with one sheet per element of sheets, whose cells
// hold the typed value of each string.
func writeWorkbook(sheets ...[][]string) (*excelize.File, error) {
	f := excelize.NewFile()
	for i, rows := range sheets {
		name := "Sheet1"
		if i > 0 {
			name = "Other"
			if _, err := f.NewSheet(name); err != nil {
				return nil, errors.Join(err, f.Close())
			}
		}
		for j, r := range rows {
			cells := make([]any, len(r))
			for k, c := range r {
				cells[k] = cellValue(c)
			}
			cell, err := excelize.CoordinatesToCellName(1, j+1)
			if err == nil {
				err = f.SetSheetRow(name, cell, &cells)
			}
			if err != nil {
				return nil, errors.Join(err, f.Close())
			}
		}
	}
	return f, nil
}

func newWorkbook(sheets ...[][]string) *excelize.File {
	GinkgoHelper()
	f := MustSucceed(writeWorkbook(sheets...))
	DeferCleanup(f.Close)
	return f
}

// encodeWorkbook returns the bytes of a workbook written by writeWorkbook.
func encodeWorkbook(sheets ...[][]string) ([]byte, error) {
	f, err := writeWorkbook(sheets...)
	if err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err = errors.Join(err, f.Close()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func toXLSX(sheets ...[][]string) []byte {
	GinkgoHelper()
	return MustSucceed(encodeWorkbook(sheets...))
}

const firstSheet = "xl/worksheets/sheet1.xml"

// mapSheet returns the workbook data with the XML of its first sheet replaced by the
// result of fn.
func mapSheet(data []byte, fn func(sheet []byte) []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range r.File {
		src, err := f.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(src)
		if err = errors.Join(err, src.Close()); err != nil {
			return nil, err
		}
		if f.Name == firstSheet {
			content = fn(content)
		}
		dst, err := w.Create(f.Name)
		if err != nil {
			return nil, err
		}
		if _, err = dst.Write(content); err != nil {
			return nil, err
		}
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// replaceInSheet returns the workbook data with old replaced by new in the XML of its
// first sheet.
func replaceInSheet(data []byte, old, new string) []byte {
	GinkgoHelper()
	return MustSucceed(mapSheet(data, func(sheet []byte) []byte {
		Expect(string(sheet)).To(ContainSubstring(old))
		return []byte(strings.Replace(string(sheet), old, new, 1))
	}))
}

// FuzzImportCSV checks that any input either imports with resolved references or fails
// with a validation error, and never panics.
func FuzzImportCSV(f *testing.F) {
	f.Add(toCSV(tableRows))
	f.Add(toCSV(tableRows[:1]))
	f.Add(
		[]byte(
			"message,field,start_bit,bit_length,id,extended\nm,a,0,8,0x1FFFFFFF,1\n",
		),
	)
	f.Fuzz(fuzzImport(f, icd.FormatCSV))
}

// FuzzImportXLSX checks that any input either imports with resolved references or fails
// with a validation error, and never panics.
func FuzzImportXLSX(f *testing.F) {
	for _, rows := range [][][]string{tableRows, tableRows[:1]} {
		data, err := encodeWorkbook(rows)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(fuzzImport(f, icd.FormatXLSX))
}

// FuzzImportXLSXSheet is FuzzImportXLSX for a valid workbook whose first sheet holds
// any XML, which reaches the cell reader that most whole-file inputs never get past.
func FuzzImportXLSXSheet(f *testing.F) {
	workbook, err := encodeWorkbook(tableRows)
	if err != nil {
		f.Fatal(err)
	}
	if _, err = mapSheet(workbook, func(sheet []byte) []byte {
		f.Add(sheet)
		return sheet
	}); err != nil {
		f.Fatal(err)
	}
	importXLSX := fuzzImport(f, icd.FormatXLSX)
	f.Fuzz(func(t *testing.T, sheet []byte) {
		data, err := mapSheet(workbook, func([]byte) []byte { return sheet })
		if err != nil {
			t.Fatal(err)
		}
		importXLSX(t, data)
	})
}

var _ = Describe("Import CSV", func() {
	It("Should group the rows of each message into one message", func(ctx SpecContext) {
		messages := MustSucceed(importFile(ctx, icd.FormatCSV, toCSV(tableRows)))
		Expect(withoutKeys(messages)).To(Equal(tableMessages))
	})

	It("Should give every message and field a key", func(ctx SpecContext) {
		messages := MustSucceed(importFile(ctx, icd.FormatCSV, toCSV(tableRows)))
		m := messages[0]
		Expect(m.Key).ToNot(Equal(uuid.Nil()))
		Expect(binaryFields(m)[0].Key).ToNot(Equal(uuid.Nil()))
	})

	It("Should skip a leading byte order mark", func(ctx SpecContext) {
		messages := MustSucceed(
			importFile(
				ctx,
				icd.FormatCSV,
				append([]byte("\ufeff"), toCSV(tableRows)...),
			),
		)
		Expect(withoutKeys(messages)).To(Equal(tableMessages))
	})

	It("Should report the line of malformed CSV", func(ctx SpecContext) {
		Expect(
			importFile(
				ctx,
				icd.FormatCSV,
				[]byte("message,field,start_bit,bit_length\na,\"b,0,8\n"),
			),
		).
			Error().
			To(MatchError(HavePrefix("data: line 2: ")))
	})

	header := "message,id,extended,length,field,start_bit,bit_length,byte_order," +
		"signed,float\n"
	DescribeTable(
		"Should reject an invalid table",
		func(ctx SpecContext, csv, message string) {
			Expect(importFile(ctx, icd.FormatCSV, []byte(csv))).Error().To(MatchError(
				"data: " + message + ": validation error",
			))
		},
		Entry("an empty file", "", "missing header row"),
		Entry("an unknown column", "message,field,start_bit,bit_length,Color\n",
			`row 1: unknown column "Color"`),
		Entry("a duplicate column", "message,field,start_bit,bit_length,Field\n",
			`row 1: duplicate column "field"`),
		Entry("a missing required column", "message,field,start_bit\n",
			`row 1: missing required column "bit_length"`),
		Entry("a missing message", header+",,,,a,0,8\n",
			"row 2, column message: is required"),
		Entry("a missing field", header+"m,,,,,0,8\n",
			"row 2, column field: is required"),
		Entry("a missing start bit", header+"m,,,,a,,8\n",
			"row 2, column start_bit: is required"),
		Entry("an invalid bit length", header+"m,,,,a,0,300\n",
			`row 2, column bit_length: invalid integer "300"`),
		Entry("an invalid id", header+"m,0xZZ,,,a,0,8\n",
			`row 2, column id: invalid id "0xZZ"`),
		Entry("an invalid length", header+"m,,,-1,a,0,8\n",
			`row 2, column length: invalid length "-1"`),
		Entry(
			"an invalid minimum",
			"message,field,start_bit,bit_length,min\nm,a,0,8,low\n",
			`row 2, column min: invalid number "low"`,
		),
		Entry(
			"an invalid maximum",
			"message,field,start_bit,bit_length,max\nm,a,0,8,hi\n",
			`row 2, column max: invalid number "hi"`,
		),
		Entry(
			"a scale that is not a number",
			"message,field,start_bit,bit_length,scale\nm,a,0,8,NaN\n",
			`row 2, column scale: invalid number "NaN"`,
		),
		Entry(
			"an infinite offset",
			"message,field,start_bit,bit_length,offset\nm,a,0,8,-Inf\n",
			`row 2, column offset: invalid number "-Inf"`,
		),
		Entry(
			"a maximum past the float64 range",
			"message,field,start_bit,bit_length,max\nm,a,0,8,1e400\n",
			`row 2, column max: invalid number "1e400"`,
		),
		Entry("an invalid byte order", header+"m,,,,a,0,8,middle\n",
			`row 2, column byte_order: invalid byte order "middle"`),
		Entry("an invalid boolean", header+"m,,,,a,0,8,,maybe\n",
			`row 2, column signed: invalid boolean "maybe"`),
		Entry("an id that conflicts with an earlier row",
			header+"m,0x100,,,a,0,8\nm,0x101,,,b,8,8\n",
			"row 3, column id: conflicts with the value in row 2"),
		Entry("an extended flag without an id", header+"m,,true,,a,0,8\n",
			"row 2, column extended: requires an id"),
	)
})

var _ = Describe("Import XLSX", func() {
	It("Should parse the table in the first sheet", func(ctx SpecContext) {
		other := [][]string{{"Color"}, {"red"}}
		messages := MustSucceed(
			importFile(ctx, icd.FormatXLSX, toXLSX(tableRows, other)),
		)
		Expect(withoutKeys(messages)).To(Equal(tableMessages))
	})

	It(
		"Should read the value of a numeric cell, not its formatted text",
		func(ctx SpecContext) {
			f := newWorkbook([][]string{
				{"message", "field", "start_bit", "bit_length", "scale", "offset"},
				{"m", "a", "8", "16", "0.001", "0.125"},
			})
			style := MustSucceed(f.NewStyle(&excelize.Style{CustomNumFmt: new("0.0")}))
			Expect(f.SetCellStyle("Sheet1", "C2", "F2", style)).To(Succeed())
			messages := MustSucceed(
				importFile(ctx, icd.FormatXLSX, MustSucceed(f.WriteToBuffer()).Bytes()),
			)
			a := binaryFields(messages[0])[0]
			Expect(a.StartBit).To(Equal(uint16(8)))
			Expect(a.Scale).To(Equal(0.001))
			Expect(a.Offset).To(Equal(0.125))
		},
	)

	It("Should report a row whose cell reference is invalid", func(ctx SpecContext) {
		rows := [][]string{
			{"message", "field", "start_bit", "bit_length"},
			{"m", "a", "0", "8"},
			{"m", "b", "8", "8"},
		}
		data := replaceInSheet(toXLSX(rows), `r="B3"`, `r="3B"`)
		Expect(importFile(ctx, icd.FormatXLSX, data)).Error().
			To(MatchError(HavePrefix("data: row 3: ")))
	})

	It("Should report the row of an invalid cell", func(ctx SpecContext) {
		rows := [][]string{
			{"message", "field", "start_bit", "bit_length"},
			{"m", "a", "0", "8"},
			{"m", "b", "x", "8"},
		}
		Expect(importFile(ctx, icd.FormatXLSX, toXLSX(rows))).Error().To(MatchError(
			`data: row 3, column start_bit: invalid integer "x": validation error`,
		))
	})

	It("Should reject data that is not a workbook", func(ctx SpecContext) {
		Expect(importFile(ctx, icd.FormatXLSX, []byte("message,field"))).Error().
			To(MatchError(HavePrefix("data: invalid workbook: ")))
	})
})
