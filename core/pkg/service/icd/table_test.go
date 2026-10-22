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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/xuri/excelize/v2"
)

// withoutKeys returns entries with every entry and field key cleared.
func withoutKeys(entries []library.Entry) []library.Entry {
	out := make([]library.Entry, len(entries))
	for i, e := range entries {
		m := e.Variant.(library.MessageEntry)
		m.Key = uuid.Nil()
		m.Payload.UpdateFieldBases(func(b *library.BaseField) { b.Key = uuid.Nil() })
		out[i] = library.Entry{Variant: m}
	}
	return out
}

func tableMessage(
	name string,
	id *library.Identifier,
	length *uint16,
	period *telem.TimeSpan,
	fields ...library.BinaryField,
) library.Entry {
	return library.Entry{Variant: library.MessageEntry{
		Name: name,
		Payload: library.Payload{Variant: library.BinaryPayload{
			Identifier: id,
			Length:     length,
			Fields:     fields,
		}},
		Period: period,
	}}
}

func canID(id uint32, extended bool) *library.Identifier {
	return &library.Identifier{
		Variant: library.CanIdentifier{ID: id, Extended: extended},
	}
}

// tableEntries are the entries of tableRows.
var tableEntries = []library.Entry{
	tableMessage(
		"Engine",
		canID(0x100, false),
		new(uint16(8)),
		new(100*telem.Millisecond),
		library.BinaryField{
			Name: "rpm", Scale: 0.25, Units: "rpm",
			BitLength: 16,
			ByteOrder: library.ByteOrderLittleEndian,
		},
		library.BinaryField{
			Name:      "temp",
			Scale:     1,
			Offset:    -40,
			Units:     "degC",
			StartBit:  16,
			BitLength: 8,
			ByteOrder: library.ByteOrderLittleEndian,
			Signed:    true,
		},
	),
	tableMessage(
		"Battery",
		canID(0x18FEF100, true),
		nil,
		nil,
		library.BinaryField{
			Name: "voltage", Scale: 0.01, Units: "V",
			StartBit:  7,
			BitLength: 16,
			ByteOrder: library.ByteOrderBigEndian,
		},
	),
	tableMessage(
		"Raw",
		nil,
		nil,
		nil,
		library.BinaryField{
			Name: "value", Scale: 1,
			BitLength: 32,
			ByteOrder: library.ByteOrderLittleEndian,
			Float:     true,
		},
	),
}

var tableRows = [][]string{
	{
		"Message", "ID", "Extended", "Length", "Period_MS", "Field", "Start_Bit",
		"Bit_Length", "Byte_Order", "Signed", "Float", "Scale", "Offset", "Units",
	},
	{
		"Engine", "0x100", "", "8", "100", "rpm", "0", "16", "little_endian", "false",
		"false", "0.25", "0", "rpm",
	},
	{"", "", "", "", "", "", "", "", "", "", "", "", "", ""},
	{
		"Engine", "256", "", "", "", "temp", "16", "8", "intel", "true", "", "1", "-40",
		"degC",
	},
	{
		"Battery", "419361024", "", "", "", "voltage", "7", "16", "Motorola", "", "",
		"0.01", "", "V",
	},
	{"Raw", "", "", "", "", "value", "0", "32", "", "", "yes"},
}

func toCSV(rows [][]string) []byte {
	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = strings.Join(r, ",")
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func toXLSX(sheets ...[][]string) []byte {
	GinkgoHelper()
	f := excelize.NewFile()
	DeferCleanup(f.Close)
	for i, rows := range sheets {
		name := "Sheet1"
		if i > 0 {
			name = "Other"
			MustSucceed(f.NewSheet(name))
		}
		for j, r := range rows {
			cells := make([]any, len(r))
			for k, c := range r {
				cells[k] = c
			}
			cell := MustSucceed(excelize.CoordinatesToCellName(1, j+1))
			Expect(f.SetSheetRow(name, cell, &cells)).To(Succeed())
		}
	}
	return MustSucceed(f.WriteToBuffer()).Bytes()
}

var _ = Describe("Parse CSV", func() {
	It("Should group the rows of each message into one message entry", func() {
		entries := MustSucceed(icd.Parse(icd.FormatCSV, toCSV(tableRows)))
		Expect(withoutKeys(entries)).To(Equal(tableEntries))
	})

	It("Should give every message and field a key", func() {
		entries := MustSucceed(icd.Parse(icd.FormatCSV, toCSV(tableRows)))
		m := messagesOf(entries)[0]
		Expect(m.Key).ToNot(Equal(uuid.Nil()))
		Expect(binaryFields(m)[0].Key).ToNot(Equal(uuid.Nil()))
	})

	It("Should skip a leading byte order mark", func() {
		entries := MustSucceed(
			icd.Parse(icd.FormatCSV, append([]byte("\ufeff"), toCSV(tableRows)...)),
		)
		Expect(withoutKeys(entries)).To(Equal(tableEntries))
	})

	It("Should report the line of malformed CSV", func() {
		Expect(
			icd.Parse(
				icd.FormatCSV,
				[]byte("message,field,start_bit,bit_length\na,\"b,0,8\n"),
			),
		).
			Error().
			To(MatchError(HavePrefix("data: line 2: ")))
	})

	header := "message,id,extended,length,period_ms,field,start_bit,bit_length," +
		"byte_order,signed,float\n"
	DescribeTable("Should reject an invalid table", func(csv, message string) {
		Expect(icd.Parse(icd.FormatCSV, []byte(csv))).Error().To(MatchError(
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
		Entry("a missing message", header+",,,,,a,0,8\n",
			"row 2, column message: is required"),
		Entry("a missing field", header+"m,,,,,,0,8\n",
			"row 2, column field: is required"),
		Entry("a missing start bit", header+"m,,,,,a,,8\n",
			"row 2, column start_bit: is required"),
		Entry("an invalid bit length", header+"m,,,,,a,0,300\n",
			`row 2, column bit_length: invalid integer "300"`),
		Entry("an invalid id", header+"m,0xZZ,,,,a,0,8\n",
			`row 2, column id: invalid id "0xZZ"`),
		Entry("an invalid length", header+"m,,,-1,,a,0,8\n",
			`row 2, column length: invalid length "-1"`),
		Entry("an invalid period", header+"m,,,,0,a,0,8\n",
			`row 2, column period_ms: invalid period "0"`),
		Entry("an invalid byte order", header+"m,,,,,a,0,8,middle\n",
			`row 2, column byte_order: invalid byte order "middle"`),
		Entry("an invalid boolean", header+"m,,,,,a,0,8,,maybe\n",
			`row 2, column signed: invalid boolean "maybe"`),
		Entry("an id that conflicts with an earlier row",
			header+"m,0x100,,,,a,0,8\nm,0x101,,,,b,8,8\n",
			"row 3, column id: conflicts with the value in row 2"),
		Entry("an extended flag without an id", header+"m,,true,,,a,0,8\n",
			"row 2, column extended: requires an id"),
	)
})

var _ = Describe("Parse XLSX", func() {
	It("Should parse the table in the first sheet", func() {
		other := [][]string{{"Color"}, {"red"}}
		entries := MustSucceed(icd.Parse(icd.FormatXLSX, toXLSX(tableRows, other)))
		Expect(withoutKeys(entries)).To(Equal(tableEntries))
	})

	It("Should report the row of an invalid cell", func() {
		rows := [][]string{
			{"message", "field", "start_bit", "bit_length"},
			{"m", "a", "0", "8"},
			{"m", "b", "x", "8"},
		}
		Expect(icd.Parse(icd.FormatXLSX, toXLSX(rows))).Error().To(MatchError(
			`data: row 3, column start_bit: invalid integer "x": validation error`,
		))
	})

	It("Should reject data that is not a workbook", func() {
		Expect(icd.Parse(icd.FormatXLSX, []byte("message,field"))).Error().
			To(MatchError(HavePrefix("data: invalid workbook: ")))
	})
})
