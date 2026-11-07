// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package dbc_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd/internal/dbc"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

// FuzzParse checks that Parse returns definitions or an *Error for any input, and never
// panics.
func FuzzParse(f *testing.F) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "testdata", "*.dbc"))
	if err != nil {
		f.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		defs, err := dbc.Parse(data)
		if err == nil {
			return
		}
		var perr *dbc.Error
		if !errors.As(err, &perr) || defs != nil {
			t.Fatalf("Parse returned %d definitions and error %v", len(defs), err)
		}
	})
}

// parseOne parses src and returns its only definition, which must have type D.
func parseOne[D dbc.Def](src string) D {
	GinkgoHelper()
	defs := MustSucceed(dbc.Parse([]byte(src)))
	Expect(defs).To(HaveLen(1))
	return defs[0].(D)
}

func signalOf(marker string) dbc.SignalDef {
	GinkgoHelper()
	msg := parseOne[*dbc.MessageDef](
		"BO_ 256 M: 8 Ecu\n SG_ S " + marker + " : 0|8@1+ (1,0) [0|0] \"\" Ecu\n",
	)
	return msg.Signals[0]
}

var _ = Describe("Parse", func() {
	It("Should parse a message and its signals", func() {
		msg := parseOne[*dbc.MessageDef](`BO_ 2147484416 Engine: 8 Ecu
 SG_ Rpm : 7|16@0- (0.25,-10) [-100|100] "rpm" Gateway,Display
`)
		Expect(msg.MessageID.IsExtended()).To(BeTrue())
		Expect(msg.MessageID.ToCAN()).To(Equal(uint32(0x300)))
		Expect(msg.Name).To(Equal(dbc.Identifier("Engine")))
		Expect(msg.Size).To(Equal(uint64(8)))
		Expect(msg.Transmitter).To(Equal(dbc.Identifier("Ecu")))
		Expect(msg.Signals).To(HaveLen(1))
		s := msg.Signals[0]
		Expect(s.Pos.Line).To(Equal(2))
		s.Pos = msg.Signals[0].Pos
		Expect(s).To(Equal(dbc.SignalDef{
			Pos:         s.Pos,
			Name:        "Rpm",
			StartBit:    7,
			Size:        16,
			IsBigEndian: true,
			IsSigned:    true,
			Factor:      0.25,
			Offset:      -10,
			Minimum:     -100,
			Maximum:     100,
			Unit:        "rpm",
			Receivers:   []dbc.Identifier{"Gateway", "Display"},
		}))
	})

	DescribeTable("Should report whether a message ID is valid",
		func(id dbc.MessageID, message string) {
			if message == "" {
				Expect(id.Validate()).To(Succeed())
				return
			}
			Expect(id.Validate()).To(MatchError(message))
		},
		Entry("the largest standard ID", dbc.MessageID(0x7FF), ""),
		Entry("the largest extended ID", dbc.MessageID(0x9FFFFFFF), ""),
		Entry(
			"a standard ID past 11 bits",
			dbc.MessageID(0x800),
			"standard ID 0x800 has more than 11 bits",
		),
		Entry(
			"an extended ID past 29 bits",
			dbc.MessageID(0xFFFFFFFF),
			"extended ID 0x7FFFFFFF has more than 29 bits",
		),
	)

	It("Should parse a message ID that is not a valid CAN ID", func() {
		msg := parseOne[*dbc.MessageDef](
			"BO_ 1073741824 VECTOR__INDEPENDENT_SIG_MSG: 0 X\n",
		)
		Expect(msg.MessageID).To(Equal(dbc.MessageID(0x40000000)))
	})

	DescribeTable("Should read the multiplexing marker of a signal",
		func(marker string, isSwitch, isMultiplexed bool, value uint64) {
			s := signalOf(marker)
			Expect(s.IsMultiplexerSwitch).To(Equal(isSwitch))
			Expect(s.IsMultiplexed).To(Equal(isMultiplexed))
			Expect(s.MultiplexerSwitch).To(Equal(value))
		},
		Entry("a multiplexer switch", "M", true, false, uint64(0)),
		Entry("a multiplexed signal", "m3", false, true, uint64(3)),
		Entry("an extended multiplexer switch", "m12M", true, true, uint64(12)),
		Entry("a lone m, which some files write for M", "m", true, false, uint64(0)),
	)

	DescribeTable("Should reject an invalid multiplexing marker",
		func(marker, reason string) {
			Expect(dbc.Parse([]byte(
				"BO_ 256 M: 8 Ecu\n SG_ S " + marker + " : 0|8@1+ (1,0) [0|0] \"\" Ecu\n",
			))).Error().To(MatchError("line 2, column 8: " + reason))
		},
		Entry("an unknown marker", "x", "expected multiplexer"),
		Entry("a number", "1", "expected multiplexer"),
		Entry("a marker without a value", "mM", "invalid multiplexer value"),
		Entry("a marker with a bad value", "m1x", "invalid multiplexer value"),
	)

	It("Should parse the switch values of a multiplexed signal", func() {
		def := parseOne[*dbc.MultiplexedValuesDef](
			"SG_MUL_VAL_ 2147484416 Current Channel 4-5, 8-9;\n",
		)
		Expect(def.MessageID).To(Equal(dbc.MessageID(2147484416)))
		Expect(def.SignalName).To(Equal(dbc.Identifier("Current")))
		Expect(def.SwitchName).To(Equal(dbc.Identifier("Channel")))
		Expect(def.Ranges).To(Equal([]dbc.MultiplexRange{
			{From: 4, To: 5},
			{From: 8, To: 9},
		}))
	})

	DescribeTable(
		"Should reject a malformed SG_MUL_VAL_ definition",
		func(src, message string) {
			Expect(dbc.Parse([]byte(src))).Error().To(MatchError(message))
		},
		Entry(
			"a missing switch",
			"SG_MUL_VAL_ 256 A;\n",
			"line 1, column 18: expected a name",
		),
		Entry(
			"a range without an upper bound",
			"SG_MUL_VAL_ 256 A S 0;\n",
			"line 1, column 22: expected token: \"-\", found: \";\" (;)",
		),
		Entry(
			"a missing semicolon",
			"SG_MUL_VAL_ 256 A S 0-1\n",
			"line 2, column 1: expected token: \";\", found: EOF ()",
		),
	)

	DescribeTable("Should parse the definition after an unknown one",
		func(tokens int) {
			unknown := "SIG_GROUP_" + strings.Repeat(" x", tokens)
			defs := MustSucceed(dbc.Parse([]byte(
				unknown + "\nBA_DEF_ BO_ \"GenMsgCycleTime\" INT 0 100;\n",
			)))
			Expect(defs).To(HaveLen(2))
			Expect(defs[0]).To(HaveField("Keyword", dbc.Keyword("SIG_GROUP_")))
			Expect(defs[1]).To(HaveField("Name", dbc.Identifier("GenMsgCycleTime")))
		},
		Entry("with no tokens", 0),
		Entry("with one token", 1),
		Entry("with two tokens", 2),
		Entry("with three tokens", 3),
	)

	It("Should parse an unknown definition on the last line", func() {
		def := parseOne[*dbc.UnknownDef]("SGTYPE_ x y")
		Expect(def.Keyword).To(Equal(dbc.Keyword("SGTYPE_")))
	})

	It("Should parse attributes and their values by type", func() {
		defs := MustSucceed(dbc.Parse([]byte(`BA_DEF_ BO_ "Period" INT 0 100;
BA_DEF_ SG_ "Gain" FLOAT 0 1;
BA_DEF_ "Owner" STRING;
BA_DEF_ BO_ "Format" ENUM "Std","FD";
BA_DEF_DEF_ "Period" 10;
BA_DEF_DEF_ "Format" "Std";
BA_ "Period" BO_ 256 20;
BA_ "Gain" SG_ 256 A 0.5;
BA_ "Owner" "Team";
BA_ "Format" BO_ 256 1;
`)))
		Expect(defs).To(HaveLen(10))
		Expect(defs[3]).To(HaveField("EnumValues", []string{"Std", "FD"}))
		Expect(defs[4]).To(HaveField("DefaultIntValue", int64(10)))
		Expect(defs[5]).To(HaveField("DefaultStringValue", "Std"))
		Expect(defs[6]).To(HaveField("IntValue", int64(20)))
		Expect(defs[7]).To(HaveField("FloatValue", 0.5))
		Expect(defs[7]).To(HaveField("SignalName", dbc.Identifier("A")))
		Expect(defs[8]).To(HaveField("StringValue", "Team"))
		Expect(defs[9]).To(HaveField("StringValue", "FD"))
	})

	It("Should parse value tables and value descriptions", func() {
		defs := MustSucceed(dbc.Parse([]byte(`VAL_TABLE_ Gear 0 "P" 1 "D" ;
VAL_ 256 Mode -1 "Fault" 0 "Off" ;
SIG_VALTYPE_ 256 Temp : 1;
`)))
		Expect(defs).To(HaveLen(3))
		table := defs[0].(*dbc.ValueTableDef)
		Expect(table.TableName).To(Equal(dbc.Identifier("Gear")))
		Expect(table.ValueDescriptions).To(HaveLen(2))
		Expect(table.ValueDescriptions[1]).To(HaveField("Description", "D"))
		vals := defs[1].(*dbc.ValueDescriptionsDef)
		Expect(vals.ObjectType).To(Equal(dbc.ObjectTypeSignal))
		Expect(vals.ValueDescriptions[0]).To(HaveField("Value", -1.0))
		Expect(defs[2]).To(HaveField("SignalValueType", dbc.SignalValueTypeFloat32))
	})

	It("Should parse a comment that spans lines", func() {
		def := parseOne[*dbc.CommentDef](
			"CM_ SG_ 256 A \"Spans\nlines \\\"here\\\"\";\n",
		)
		Expect(def.ObjectType).To(Equal(dbc.ObjectTypeSignal))
		Expect(def.SignalName).To(Equal(dbc.Identifier("A")))
		Expect(def.Comment).To(Equal(`Spans lines \"here\"`))
	})

	It("Should parse the header definitions", func() {
		defs := MustSucceed(
			dbc.Parse([]byte("VERSION \"1.0\"\n\nNS_ :\n\tSG_MUL_VAL_\n" +
				"\tBA_\n\nBS_: 500\n\nBU_: Ecu Gateway\n")),
		)
		Expect(defs).To(HaveLen(4))
		Expect(defs[0]).To(HaveField("Version", "1.0"))
		Expect(defs[1]).To(HaveField(
			"Symbols",
			[]dbc.Keyword{dbc.KeywordMultiplexedValues, dbc.KeywordAttributeValue},
		))
		Expect(defs[2]).To(HaveField("BaudRate", uint64(500)))
		Expect(defs[3]).To(HaveField("NodeNames", []dbc.Identifier{"Ecu", "Gateway"}))
	})

	It("Should parse the bit timing registers", func() {
		def := parseOne[*dbc.BitTimingDef]("BS_: 500:12,34\n")
		Expect(def.BaudRate).To(Equal(uint64(500)))
		Expect(def.BTR1).To(Equal(uint64(12)))
		Expect(def.BTR2).To(Equal(uint64(34)))
	})

	It("Should parse new symbols indented with spaces", func() {
		defs := MustSucceed(dbc.Parse([]byte(
			"NS_ :\n    CM_\n    BA_DEF_\n    BA_\n\nBS_:\n",
		)))
		Expect(defs).To(HaveLen(2))
		Expect(defs[0]).To(HaveField("Symbols", []dbc.Keyword{
			dbc.KeywordComment,
			dbc.KeywordAttribute,
			dbc.KeywordAttributeValue,
		}))
		Expect(defs[1]).To(BeAssignableToTypeOf(&dbc.BitTimingDef{}))
	})

	It("Should skip line comments", func() {
		defs := MustSucceed(dbc.Parse([]byte(
			"// A comment.\r\nVERSION \"2.0\" // Another comment.\r\n\r\nBS_:\r\n",
		)))
		Expect(defs).To(HaveLen(2))
		Expect(defs[0]).To(HaveField("Version", "2.0"))
	})

	It("Should keep a comment marker inside a string", func() {
		def := parseOne[*dbc.CommentDef]("CM_ \"See http://example.com\";\n")
		Expect(def.Comment).To(Equal("See http://example.com"))
	})

	It("Should read a file that is not UTF-8 as Windows-1252", func() {
		def := parseOne[*dbc.CommentDef]("CM_ \"Range \xb10.5 \xb0C, cost \x80\";\n")
		Expect(def.Comment).To(Equal("Range ±0.5 °C, cost €"))
	})

	It("Should read numbers with a plus sign", func() {
		s := parseOne[*dbc.MessageDef](
			"BO_ 256 M: 8 Ecu\n SG_ S : 8|8@1- (+1,-128) [-128|+127] \"\" Ecu\n",
		).Signals[0]
		Expect(s.Factor).To(Equal(1.0))
		Expect(s.Minimum).To(Equal(-128.0))
		Expect(s.Maximum).To(Equal(127.0))
	})

	It("Should accept an attribute name that is not an identifier", func() {
		defs := MustSucceed(dbc.Parse([]byte(`BA_DEF_ SG_ "NWM-Wakeup" ENUM "No","Yes";
BA_DEF_DEF_ "NWM-Wakeup" "No";
BA_ "NWM-Wakeup" SG_ 256 A 1;
`)))
		Expect(defs).To(HaveLen(3))
		Expect(defs[0]).To(HaveField("Name", dbc.Identifier("NWM-Wakeup")))
		Expect(defs[2]).To(HaveField("StringValue", "Yes"))
	})

	DescribeTable("Should parse the access type of an environment variable",
		func(accessType dbc.AccessType) {
			def := parseOne[*dbc.EnvironmentVariableDef](
				"EV_ Mode: 0 [0|0] \"\" 0 2 " + string(accessType) + " Vector__XXX;\n",
			)
			Expect(def.AccessType).To(Equal(accessType))
		},
		Entry("unrestricted", dbc.AccessTypeUnrestricted),
		Entry("read and write", dbc.AccessTypeReadWrite),
		Entry("unrestricted, for a string", dbc.AccessTypeUnrestrictedString),
		Entry("read, for a string", dbc.AccessTypeReadString),
		Entry("write, for a string", dbc.AccessTypeWriteString),
		Entry("read and write, for a string", dbc.AccessTypeReadWriteString),
	)

	It("Should parse environment variables and transmitters", func() {
		defs := MustSucceed(dbc.Parse([]byte(
			`EV_ Speed: 1 [0|100] "km/h" 0 1 DUMMY_NODE_VECTOR1 Ecu;
ENVVAR_DATA_ Speed: 4;
BO_TX_BU_ 256 : Ecu,Gateway;
`,
		)))
		Expect(defs).To(HaveLen(3))
		Expect(defs[0]).To(HaveField("Type", dbc.EnvironmentVariableTypeFloat))
		Expect(defs[0]).To(HaveField("AccessType", dbc.AccessTypeRead))
		Expect(defs[1]).To(HaveField("DataSize", uint64(4)))
		Expect(
			defs[2],
		).To(HaveField("Transmitters", []dbc.Identifier{"Ecu", "Gateway"}))
	})

	DescribeTable(
		"Should reject an invalid definition",
		func(src, message string) {
			Expect(dbc.Parse([]byte(src))).Error().To(MatchError(message))
		},
		Entry(
			"a definition that does not start with a keyword",
			"VERSION \"\"\n256\n",
			"line 2, column 1: expected a keyword",
		),
		Entry(
			"a name that is a number",
			"BO_ 1275 2017_5: 8 Ecu\n",
			"line 1, column 10: a name must start with a letter or an underscore",
		),
		Entry(
			"a comment with an ID but no object type",
			"CM_ 145 \"Pedal\";\n",
			"line 1, column 5: expected a string",
		),
		Entry(
			"a signal comment with no signal name",
			"CM_ SG_ 304 \"Pedal\";\n",
			"line 1, column 13: expected a name",
		),
		Entry(
			"a name that starts with a digit",
			"BO_ 256 M: 8 Ecu\n SG_ 0_COUNT : 0|8@1+ (1,0) [0|0] \"\" Ecu\n",
			"line 2, column 6: a name must start with a letter or an underscore",
		),
		Entry(
			"a value description with no closing semicolon",
			"VAL_ 256 A 1 \"On\" 0 \"Off\"\nVAL_ 256 B 1 \"On\" ;\n",
			"line 1, column 1: VAL_ has no closing ';'",
		),
		Entry(
			"a value table with no closing semicolon",
			"VAL_TABLE_ T 1 \"On\"\nBO_ 256 M: 8 Ecu\n",
			"line 1, column 1: VAL_TABLE_ has no closing ';'",
		),
		Entry(
			"an unknown access type",
			"EV_ Mode: 0 [0|0] \"\" 0 2 DUMMY_NODE_VECTOR4 Vector__XXX;\n",
			"line 1, column 26: invalid access type",
		),
		Entry(
			"a message ID past 32 bits",
			"BO_ 4294967296 M: 8 Ecu\n",
			"line 1, column 5: invalid message ID: 4294967296",
		),
		Entry(
			"an unknown signal value type",
			"SIG_VALTYPE_ 256 A : 3;\n",
			"line 1, column 22: invalid signal value type: 3",
		),
		Entry(
			"an unknown attribute value type",
			"BA_DEF_ \"A\" BOOL;\n",
			"line 1, column 13: invalid attribute value type: BOOL",
		),
		Entry(
			"an unterminated string",
			"VERSION \"1.0",
			"line 1, column 9: unterminated string",
		),
	)

	It("Should return an *Error that names where parsing failed", func() {
		Expect(dbc.Parse([]byte("VERSION 1"))).Error().To(SatisfyAll(
			BeAssignableToTypeOf(&dbc.Error{}),
			HaveField("Pos.Line", 1),
			HaveField("Pos.Column", 9),
			HaveField("Reason", "expected a string"),
		))
	})
})
