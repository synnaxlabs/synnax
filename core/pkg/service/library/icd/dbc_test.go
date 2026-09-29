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
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// expectedFile is the shape testdata/generate.py writes from the cantools decoding of
// a DBC file.
type expectedFile struct {
	Messages    []expectedMessage              `json:"messages"`
	ValueTables map[string][]library.EnumValue `json:"value_tables"`
}

type expectedMessage struct {
	Name     string          `json:"name"`
	ID       uint32          `json:"id"`
	Extended bool            `json:"extended"`
	FD       bool            `json:"fd"`
	Length   uint16          `json:"length"`
	PeriodMS *int64          `json:"period_ms"`
	Fields   []expectedField `json:"fields"`
}

type expectedField struct {
	Name            string              `json:"name"`
	StartBit        uint16              `json:"start_bit"`
	BitLength       uint8               `json:"bit_length"`
	ByteOrder       library.ByteOrder   `json:"byte_order"`
	Signed          bool                `json:"signed"`
	Float           bool                `json:"float"`
	Scale           float64             `json:"scale"`
	Offset          float64             `json:"offset"`
	Units           string              `json:"units"`
	Choices         []library.EnumValue `json:"choices"`
	Multiplexor     *string             `json:"multiplexor"`
	MultiplexValues []int32             `json:"multiplex_values"`
}

func messagesOf(entries []library.Entry) []library.MessageEntry {
	var messages []library.MessageEntry
	for _, e := range entries {
		if m, ok := e.Variant.(library.MessageEntry); ok {
			messages = append(messages, m)
		}
	}
	return messages
}

func messageNamed(entries []library.Entry, name string) library.MessageEntry {
	GinkgoHelper()
	for _, m := range messagesOf(entries) {
		if m.Name == name {
			return m
		}
	}
	Fail("no message named " + name)
	return library.MessageEntry{}
}

func enumWhere(
	entries []library.Entry,
	match func(library.EnumEntry) bool,
) library.EnumEntry {
	GinkgoHelper()
	for _, e := range entries {
		if en, ok := e.Variant.(library.EnumEntry); ok && match(en) {
			return en
		}
	}
	Fail("no matching enum entry")
	return library.EnumEntry{}
}

func enumNamed(entries []library.Entry, name string) library.EnumEntry {
	GinkgoHelper()
	return enumWhere(entries, func(e library.EnumEntry) bool { return e.Name == name })
}

func enumWithKey(entries []library.Entry, key library.EntryKey) library.EnumEntry {
	GinkgoHelper()
	return enumWhere(entries, func(e library.EnumEntry) bool { return e.Key == key })
}

func binaryFields(m library.MessageEntry) []library.BinaryField {
	fields := make([]library.BinaryField, len(m.Fields))
	for i, f := range m.Fields {
		fields[i] = f.Variant.(library.BinaryField)
	}
	return fields
}

func fieldNamed(m library.MessageEntry, name string) library.BinaryField {
	GinkgoHelper()
	for _, f := range binaryFields(m) {
		if f.Name == name {
			return f
		}
	}
	Fail("no field named " + name)
	return library.BinaryField{}
}

// toExpected converts parsed messages to the shape of the cantools decoding.
func toExpected(entries []library.Entry) []expectedMessage {
	GinkgoHelper()
	var out []expectedMessage
	for _, m := range messagesOf(entries) {
		id := m.Identifier.Variant.(library.CanIdentifier)
		em := expectedMessage{
			Name:     m.Name,
			ID:       id.ID,
			Extended: id.Extended,
			FD:       id.Fd,
			Length:   *m.Length,
		}
		if m.Period != nil {
			em.PeriodMS = new(int64(*m.Period / telem.Millisecond))
		}
		fields := binaryFields(m)
		names := make(map[library.FieldKey]string, len(fields))
		for _, f := range fields {
			names[f.Key] = f.Name
		}
		for _, f := range fields {
			ef := expectedField{
				Name:            f.Name,
				StartBit:        f.StartBit,
				BitLength:       f.BitLength,
				ByteOrder:       f.ByteOrder,
				Signed:          f.Signed,
				Float:           f.Float,
				Scale:           f.Scale,
				Offset:          f.Offset,
				Units:           f.Units,
				MultiplexValues: append([]int32{}, f.MultiplexValues...),
			}
			if f.Enumeration != nil {
				ef.Choices = enumWithKey(entries, *f.Enumeration).Values
			}
			if f.Multiplexor != nil {
				ef.Multiplexor = new(names[*f.Multiplexor])
			}
			em.Fields = append(em.Fields, ef)
		}
		out = append(out, em)
	}
	return out
}

func readFixture(name string) ([]byte, expectedFile) {
	GinkgoHelper()
	data := MustSucceed(os.ReadFile(filepath.Join("testdata", name+".dbc")))
	raw := MustSucceed(os.ReadFile(filepath.Join("testdata", name+".expected.json")))
	var want expectedFile
	Expect(json.Unmarshal(raw, &want)).To(Succeed())
	return data, want
}

const dbcHeader = `VERSION ""

NS_ :

BS_:

BU_: Ecu

`

var _ = Describe("ParseDBC", func() {
	DescribeTable("Should decode each message as cantools does", func(name string) {
		data, want := readFixture(name)
		entries := MustSucceed(icd.ParseDBC(data))
		Expect(toExpected(entries)).To(Equal(want.Messages))
		for table, values := range want.ValueTables {
			Expect(enumNamed(entries, table).Values).To(Equal(values))
		}
	},
		Entry("ids, byte orders, floats, units, value tables, and periods", "basic"),
		Entry("simple multiplexing", "multiplexed"),
		Entry("extended multiplexing with SG_MUL_VAL_ ranges", "extended_multiplexed"),
	)

	It("Should read a file with CRLF line endings", func() {
		data, want := readFixture("extended_multiplexed")
		crlf := bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
		Expect(toExpected(MustSucceed(icd.ParseDBC(crlf)))).To(Equal(want.Messages))
	})

	It("Should reference a value table whose values match a signal's", func() {
		data, _ := readFixture("basic")
		entries := MustSucceed(icd.ParseDBC(data))
		gear := fieldNamed(messageNamed(entries, "Engine"), "Gear")
		Expect(*gear.Enumeration).To(Equal(enumNamed(entries, "GearTable").Key))
		state := fieldNamed(messageNamed(entries, "Battery"), "State")
		Expect(*state.Enumeration).To(Equal(enumNamed(entries, "Battery.State").Key))
	})

	It("Should give every entry and field a distinct key", func() {
		data, _ := readFixture("basic")
		entries := MustSucceed(icd.ParseDBC(data))
		keys := make(set.Set[library.EntryKey])
		for _, e := range entries {
			switch v := e.Variant.(type) {
			case library.EnumEntry:
				keys.Add(v.Key)
			case library.MessageEntry:
				keys.Add(v.Key)
				for _, f := range binaryFields(v) {
					keys.Add(f.Key)
				}
			}
		}
		Expect(entries).To(HaveLen(7))
		Expect(keys).To(HaveLen(7 + 13))
	})

	It("Should read a VFrameFormat defined as an INT attribute", func() {
		data := []byte(dbcHeader + `BO_ 256 Fast: 64 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ BO_ "VFrameFormat" INT 0 15;
BA_DEF_DEF_ "VFrameFormat" 0;
BA_ "VFrameFormat" BO_ 256 14;
`)
		m := messageNamed(MustSucceed(icd.ParseDBC(data)), "Fast")
		Expect(m.Identifier.Variant.(library.CanIdentifier).Fd).To(BeTrue())
	})

	It("Should read the lines that follow an unknown definition", func() {
		data := []byte(dbcHeader + `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

SIG_GROUP_ 256 Group 1 : A;
BA_DEF_ BO_ "GenMsgCycleTime" INT 0 65535;
BA_DEF_DEF_ "GenMsgCycleTime" 0;
BA_ "GenMsgCycleTime" BO_ 256 20;
`)
		m := messageNamed(MustSucceed(icd.ParseDBC(data)), "Engine")
		Expect(m.Period).To(Equal(new(20 * telem.Millisecond)))
	})

	It("Should append the value to a repeated value description", func() {
		data := []byte(dbcHeader + `BO_ 256 Engine: 8 Ecu
 SG_ Mode : 0|8@1+ (1,0) [0|255] "" Ecu

VAL_ 256 Mode 0 "Off" 1 "Reserved" 2 "Reserved" ;
`)
		entries := MustSucceed(icd.ParseDBC(data))
		Expect(enumNamed(entries, "Engine.Mode").Values).To(Equal([]library.EnumValue{
			{Value: 0, Name: "Off"},
			{Value: 1, Name: "Reserved"},
			{Value: 2, Name: "Reserved (2)"},
		}))
	})

	It("Should not read definitions inside a multi-line comment", func() {
		data := []byte(dbcHeader + `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

CM_ BO_ 256 "Spans lines,
SG_MUL_VAL_ 256 A B 0-0;
and ends here.";
`)
		Expect(binaryFields(messageNamed(MustSucceed(icd.ParseDBC(data)), "Engine"))).
			To(HaveLen(1))
	})

	DescribeTable("Should reject an invalid file", func(body, message string) {
		err := MatchError(SatisfyAll(
			HavePrefix("data: "+message),
			HaveSuffix(": validation error"),
		))
		Expect(icd.ParseDBC([]byte(dbcHeader + body))).Error().To(err)
	},
		Entry("a syntax error", `BO_ 256 Engine 8 Ecu
`, "line 9, column 18: expected token"),
		Entry("a syntax error after a blanked SG_MUL_VAL_ line", `BO_ 256 Engine: 8 Ecu
 SG_ S M : 0|8@1+ (1,0) [0|255] "" Ecu
 SG_ A m0 : 8|8@1+ (1,0) [0|255] "" Ecu
SG_MUL_VAL_ 256 A S 0-0;
BO_ x
`, "line 13, column 5"),
		Entry("a malformed SG_MUL_VAL_ line", `SG_MUL_VAL_ 256 A;
`, "line 9: invalid SG_MUL_VAL_ definition"),
		Entry("an inverted multiplexor range", `SG_MUL_VAL_ 256 A S 3-1;
`, `line 9: invalid multiplexor range "3-1"`),
		Entry("a multiplexor range past int32", `SG_MUL_VAL_ 256 A S 0-4294967296;
`, `line 9: multiplexor range "0-4294967296" of signal A exceeds the int32 range`),
		Entry("multiplexor ranges that are too large", `SG_MUL_VAL_ 256 A S 0-70000;
`, "line 9: multiplexor ranges hold more than 65536 values"),
		Entry("a repeated SG_MUL_VAL_ line", `SG_MUL_VAL_ 256 A S 0-0;
SG_MUL_VAL_ 256 A S 1-1;
`, "line 10: duplicate SG_MUL_VAL_ for signal A"),
		Entry(
			"an SG_MUL_VAL_ line for a signal no message multiplexes",
			`BO_ 256 Engine: 8 Ecu
 SG_ S M : 0|8@1+ (1,0) [0|255] "" Ecu
 SG_ A : 8|8@1+ (1,0) [0|255] "" Ecu

SG_MUL_VAL_ 256 A S 0-0;
`,
			"line 13: no message with id 256 has a multiplexed signal A",
		),
		Entry("an SG_MUL_VAL_ multiplexor outside the message", `BO_ 256 Engine: 8 Ecu
 SG_ S M : 0|8@1+ (1,0) [0|255] "" Ecu
 SG_ A m0 : 8|8@1+ (1,0) [0|255] "" Ecu

SG_MUL_VAL_ 256 A Missing 0-0;
`, "line 13: multiplexor Missing is not a signal of message Engine"),
		Entry("an ambiguous multiplexed signal", `BO_ 256 Engine: 8 Ecu
 SG_ S M : 0|4@1+ (1,0) [0|15] "" Ecu
 SG_ T m0M : 4|4@1+ (1,0) [0|15] "" Ecu
 SG_ A m1 : 8|8@1+ (1,0) [0|255] "" Ecu
`, "line 12: signal A needs an SG_MUL_VAL_ definition to name its multiplexor"),
		Entry("a multiplexed signal without a multiplexor", `BO_ 256 Engine: 8 Ecu
 SG_ A m1 : 8|8@1+ (1,0) [0|255] "" Ecu
`, "line 10: signal A needs an SG_MUL_VAL_ definition to name its multiplexor"),
		Entry("a float whose size does not match its type", `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|16@1- (1,0) [0|0] "" Ecu

SIG_VALTYPE_ 256 A : 1;
`, "line 12: signal A is a 32-bit float but has 16 bits"),
		Entry("a value description past int32", `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|40@1+ (1,0) [0|0] "" Ecu

VAL_ 256 A 4294967296 "Big" ;
`, "line 12: value 4.294967296e+09 of signal A is not an int32"),
		Entry("a negative cycle time", `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ BO_ "GenMsgCycleTime" INT -100 65535;
BA_ "GenMsgCycleTime" BO_ 256 -5;
`, "line 9: message Engine has a negative cycle time"),
	)
})
