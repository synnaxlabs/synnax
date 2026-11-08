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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// expectedFile is the shape testdata/generate.py writes from the cantools decoding of
// a DBC file.
type expectedFile struct {
	Messages []expectedMessage `json:"messages"`
}

type expectedMessage struct {
	Name     string          `json:"name"`
	ID       uint32          `json:"id"`
	Extended bool            `json:"extended"`
	FD       bool            `json:"fd"`
	Length   uint16          `json:"length"`
	Fields   []expectedField `json:"fields"`
}

type expectedField struct {
	Name            string        `json:"name"`
	StartBit        uint16        `json:"start_bit"`
	BitLength       uint8         `json:"bit_length"`
	ByteOrder       icd.ByteOrder `json:"byte_order"`
	Signed          bool          `json:"signed"`
	Float           bool          `json:"float"`
	Scale           float64       `json:"scale"`
	Offset          float64       `json:"offset"`
	Units           string        `json:"units"`
	Multiplexor     *string       `json:"multiplexor"`
	MultiplexValues []int32       `json:"multiplex_values"`
	Min             *float64      `json:"min"`
	Max             *float64      `json:"max"`
}

func messageNamed(messages []icd.Message, name string) icd.Message {
	GinkgoHelper()
	for _, m := range messages {
		if m.Name == name {
			return m
		}
	}
	Fail("no message named " + name)
	return icd.Message{}
}

func canIdentifier(m icd.Message) icd.CanIdentifier {
	return binaryPayloadOf(m).Identifier.Variant.(icd.CanIdentifier)
}

func binaryFields(m icd.Message) []icd.BinaryField {
	return binaryPayloadOf(m).Fields
}

func fieldNamed(m icd.Message, name string) icd.BinaryField {
	GinkgoHelper()
	for _, f := range binaryFields(m) {
		if f.Name == name {
			return f
		}
	}
	Fail("no field named " + name)
	return icd.BinaryField{}
}

// toExpected converts parsed messages to the shape of the cantools decoding.
func toExpected(messages []icd.Message) []expectedMessage {
	GinkgoHelper()
	var out []expectedMessage
	for _, m := range messages {
		p := binaryPayloadOf(m)
		id := p.Identifier.Variant.(icd.CanIdentifier)
		em := expectedMessage{
			Name:     m.Name,
			ID:       id.ID,
			Extended: id.Extended,
			FD:       id.Fd,
			Length:   *p.Length,
		}
		fields := binaryFields(m)
		names := make(map[icd.FieldKey]string, len(fields))
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
				Min:             f.Min,
				Max:             f.Max,
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

// FuzzImportDBC checks that any input either imports with resolved references or fails
// with a validation error, and never panics.
func FuzzImportDBC(f *testing.F) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.dbc"))
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
	f.Add([]byte(dbcHeader + `BO_ 2364539904 M: 24 Ecu
 SG_ A : 0|8@1+ (1,0) [0|0] "" Ecu

BA_DEF_ "BusType" STRING ;
BA_DEF_ "ProtocolType" STRING ;
BA_DEF_ BO_ "VFrameFormat" INT 0 15;
BA_ "BusType" "CAN FD";
BA_ "ProtocolType" "J1939";
BA_ "VFrameFormat" BO_ 2364539904 3;
`))
	f.Fuzz(fuzzImport(f, icd.FormatDBC))
}

const dbcHeader = `VERSION ""

NS_ :

BS_:

BU_: Ecu

`

var _ = Describe("Import DBC", func() {
	DescribeTable(
		"Should decode each message as cantools does",
		func(ctx SpecContext, name string) {
			data, want := readFixture(name)
			messages := MustSucceed(importFile(ctx, icd.FormatDBC, data))
			Expect(toExpected(messages)).To(Equal(want.Messages))
		},
		Entry("ids, byte orders, floats, and units", "basic"),
		Entry("simple multiplexing", "multiplexed"),
		Entry("extended multiplexing with SG_MUL_VAL_ ranges", "extended_multiplexed"),
	)

	It("Should read a file with CRLF line endings", func(ctx SpecContext) {
		data, want := readFixture("extended_multiplexed")
		crlf := bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
		Expect(
			toExpected(MustSucceed(importFile(ctx, icd.FormatDBC, crlf))),
		).To(Equal(want.Messages))
	})

	It("Should give every message and field a distinct key", func(ctx SpecContext) {
		data, _ := readFixture("basic")
		messages := MustSucceed(importFile(ctx, icd.FormatDBC, data))
		keys := make(set.Set[icd.MessageKey])
		for _, m := range messages {
			keys.Add(m.Key)
			for _, f := range binaryFields(m) {
				keys.Add(f.Key)
			}
		}
		Expect(messages).To(HaveLen(5))
		Expect(keys).To(HaveLen(5 + 13))
	})

	It("Should read a VFrameFormat defined as an INT attribute", func(ctx SpecContext) {
		data := []byte(dbcHeader + `BO_ 256 Fast: 64 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ BO_ "VFrameFormat" INT 0 15;
BA_DEF_DEF_ "VFrameFormat" 0;
BA_ "VFrameFormat" BO_ 256 14;
`)
		m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "Fast")
		Expect(canIdentifier(m).Fd).To(BeTrue())
	})

	DescribeTable("Should choose between CAN and CAN FD frames",
		func(ctx SpecContext, body string, fd bool) {
			data := []byte(dbcHeader + body)
			m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "M")
			Expect(canIdentifier(m).Fd).To(Equal(fd))
		},
		Entry("FD for a message longer than 8 bytes with no VFrameFormat",
			`BO_ 256 M: 24 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu
`, true),
		Entry("FD for a message longer than 8 bytes with a classic VFrameFormat",
			`BO_ 256 M: 64 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ BO_ "VFrameFormat" ENUM "StandardCAN","ExtendedCAN";
BA_DEF_DEF_ "VFrameFormat" "StandardCAN";
`, true),
		Entry("FD on a CAN FD bus with no VFrameFormat", `BO_ 256 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ "BusType" STRING ;
BA_DEF_DEF_ "BusType" "";
BA_ "BusType" "CAN FD";
`, true),
		Entry("classic on a CAN FD bus with a classic VFrameFormat", `BO_ 256 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ "BusType" STRING ;
BA_DEF_ BO_ "VFrameFormat" ENUM "StandardCAN","ExtendedCAN";
BA_DEF_DEF_ "VFrameFormat" "StandardCAN";
BA_ "BusType" "CAN FD";
`, false),
		Entry("classic on a classic bus", `BO_ 256 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ "BusType" STRING ;
BA_ "BusType" "CAN";
`, false),
	)

	DescribeTable("Should switch the bitrate of a CAN FD message by its CANFD_BRS",
		func(ctx SpecContext, body string, switched bool) {
			data := []byte(dbcHeader + `BO_ 256 M: 24 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu
BO_ 257 Classic: 8 Ecu
 SG_ B : 0|8@1+ (1,0) [0|255] "" Ecu

` + body)
			messages := MustSucceed(importFile(ctx, icd.FormatDBC, data))
			Expect(canIdentifier(messageNamed(messages, "M")).BitrateSwitched).
				To(Equal(switched))
			Expect(canIdentifier(messageNamed(messages, "Classic")).BitrateSwitched).
				To(BeFalse())
		},
		Entry("from the default of an ENUM attribute",
			`BA_DEF_ BO_ "CANFD_BRS" ENUM "0","1";
BA_DEF_DEF_ "CANFD_BRS" "1";
`, true),
		Entry("from the value an ENUM attribute gives the message",
			`BA_DEF_ BO_ "CANFD_BRS" ENUM "0","1";
BA_DEF_DEF_ "CANFD_BRS" "1";
BA_ "CANFD_BRS" BO_ 256 0;
`, false),
		Entry("from an INT attribute",
			`BA_DEF_ BO_ "CANFD_BRS" INT 0 1;
BA_DEF_DEF_ "CANFD_BRS" 0;
BA_ "CANFD_BRS" BO_ 256 1;
`, true),
		Entry("off with no CANFD_BRS", "", false),
	)

	j1939Formats := `BA_DEF_ BO_ "VFrameFormat" ENUM "StandardCAN","ExtendedCAN",
 "reserved","J1939PG";
BA_DEF_DEF_ "VFrameFormat" "J1939PG";
`
	DescribeTable("Should match a J1939 message by its parameter group number",
		func(ctx SpecContext, body string, id, mask uint32) {
			data := []byte(dbcHeader + body)
			m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "M")
			Expect(canIdentifier(m)).To(Equal(icd.CanIdentifier{
				ID:       id,
				Extended: true,
				Mask:     new(mask),
			}))
		},
		Entry("ignoring the priority and source address of a PDU2 message",
			`BO_ 2364539904 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

`+j1939Formats, uint32(0x0CF00400), uint32(0x03FFFF00)),
		Entry("ignoring the destination address of a PDU1 message",
			`BO_ 2348810243 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

`+j1939Formats, uint32(0x0C000003), uint32(0x03FF0000)),
		Entry("from a VFrameFormat defined as an INT attribute",
			`BO_ 2364539904 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ BO_ "VFrameFormat" INT 0 15;
BA_ "VFrameFormat" BO_ 2364539904 3;
`, uint32(0x0CF00400), uint32(0x03FFFF00)),
		Entry("from a ProtocolType of J1939 with no VFrameFormat",
			`BO_ 2364539904 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ "ProtocolType" STRING ;
BA_ "ProtocolType" "J1939";
`, uint32(0x0CF00400), uint32(0x03FFFF00)),
	)

	It(
		"Should match a standard message in a J1939 file exactly",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 256 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

` + j1939Formats)
			m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "M")
			Expect(canIdentifier(m)).To(Equal(icd.CanIdentifier{ID: 0x100}))
		},
	)

	It(
		"Should match a message that is not a J1939 group exactly",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 2364539904 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BA_DEF_ "ProtocolType" STRING ;
` + j1939Formats + `BA_ "ProtocolType" "J1939";
BA_ "VFrameFormat" BO_ 2364539904 1;
`)
			m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "M")
			Expect(canIdentifier(m)).To(Equal(icd.CanIdentifier{
				ID:       0x0CF00400,
				Extended: true,
			}))
		},
	)

	DescribeTable("Should match J1939 messages that share a group by their addresses",
		func(ctx SpecContext, first, second, mask uint32) {
			data := []byte(dbcHeader + fmt.Sprintf(`BO_ %d M1: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

BO_ %d M2: 8 Ecu
 SG_ B : 0|8@1+ (1,0) [0|255] "" Ecu

`, first|1<<31, second|1<<31) + j1939Formats)
			messages := MustSucceed(importFile(ctx, icd.FormatDBC, data))
			for name, id := range map[string]uint32{"M1": first, "M2": second} {
				Expect(canIdentifier(messageNamed(messages, name))).To(Equal(
					icd.CanIdentifier{ID: id, Extended: true, Mask: new(mask)},
				))
			}
		},
		Entry("by the source address of a PDU2 group from two senders",
			uint32(0x0CF00400), uint32(0x0CF00401), uint32(0x03FFFFFF)),
		Entry("by the source address of a PDU1 group from two senders",
			uint32(0x0C000003), uint32(0x0C000004), uint32(0x03FF00FF)),
		Entry("by both addresses of a PDU1 group one sender sends to two receivers",
			uint32(0x0C000003), uint32(0x0C00FF03), uint32(0x03FFFFFF)),
	)

	It(
		"Should keep the range of a signal unless both of its limits are zero",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 256 M: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|0] "" Ecu
 SG_ B : 8|8@1- (0.5,-10) [-20|50] "" Ecu
 SG_ C : 16|8@1+ (1,0) [0|255] "" Ecu
`)
			m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "M")
			a, b, c := fieldNamed(m, "A"), fieldNamed(m, "B"), fieldNamed(m, "C")
			Expect(a.Min).To(BeNil())
			Expect(a.Max).To(BeNil())
			Expect(b.Min).To(Equal(new(-20.0)))
			Expect(b.Max).To(Equal(new(50.0)))
			Expect(c.Min).To(Equal(new(0.0)))
			Expect(c.Max).To(Equal(new(255.0)))
		},
	)

	It(
		"Should read the lines that follow an unknown definition",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|32@1- (1,0) [0|0] "" Ecu

SIG_GROUP_ 256 Group 1 : A;
SIG_VALTYPE_ 256 A : 1;
`)
			m := messageNamed(
				MustSucceed(importFile(ctx, icd.FormatDBC, data)),
				"Engine",
			)
			Expect(binaryFields(m)[0].Float).To(BeTrue())
		},
	)

	It(
		"Should not read definitions inside a multi-line comment",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu

CM_ BO_ 256 "Spans lines,
SG_MUL_VAL_ 256 A B 0-0;
and ends here.";
`)
			Expect(
				binaryFields(
					messageNamed(
						MustSucceed(importFile(ctx, icd.FormatDBC, data)),
						"Engine",
					),
				),
			).
				To(HaveLen(1))
		},
	)

	It("Should name messages and signals by their long names", func(ctx SpecContext) {
		data := []byte(dbcHeader + `BO_ 256 Engine_Stat: 8 Ecu
 SG_ Mode : 0|8@1+ (1,0) [0|0] "" Ecu
 SG_ Rpm : 8|8@1+ (1,0) [0|0] "" Ecu
 SG_ Gear : 16|8@1+ (1,0) [0|0] "" Ecu
 SG_ T M : 24|8@1+ (1,0) [0|0] "" Ecu
 SG_ U m0 : 32|8@1+ (1,0) [0|0] "" Ecu

BA_DEF_ SG_ "SystemSignalLongSymbol" STRING ;
BA_DEF_ BO_ "SystemMessageLongSymbol" STRING ;
BA_DEF_DEF_ "SystemSignalLongSymbol" "";
BA_DEF_DEF_ "SystemMessageLongSymbol" "";
BA_ "SystemMessageLongSymbol" BO_ 256 "Engine_Status_From_The_Engine_Control_Unit";
BA_ "SystemSignalLongSymbol" SG_ 256 Mode "Operating_Mode_Of_The_Engine_Control_Unit";
BA_ "SystemSignalLongSymbol" SG_ 256 Gear "";
BA_ "SystemSignalLongSymbol" SG_ 256 T "Multiplexor_Of_The_Engine_Status_Message";
`)
		messages := MustSucceed(importFile(ctx, icd.FormatDBC, data))
		m := messageNamed(messages, "Engine_Status_From_The_Engine_Control_Unit")
		fieldNamed(m, "Operating_Mode_Of_The_Engine_Control_Unit")
		fieldNamed(m, "Rpm")
		fieldNamed(m, "Gear")
		t := fieldNamed(m, "Multiplexor_Of_The_Engine_Status_Message")
		Expect(fieldNamed(m, "U").Multiplexor).To(Equal(new(t.Key)))
	})

	It(
		"Should accept signals that end at the last bit of the message",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 256 M: 2 Ecu
 SG_ A : 8|8@1+ (1,0) [0|0] "" Ecu
 SG_ B : 7|8@0+ (1,0) [0|0] "" Ecu
 SG_ C : 7|16@0+ (1,0) [0|0] "" Ecu
`)
			Expect(binaryFields(messageNamed(
				MustSucceed(importFile(ctx, icd.FormatDBC, data)),
				"M",
			))).To(HaveLen(3))
		},
	)

	It(
		"Should multiplex a switch that an SG_MUL_VAL_ line names",
		func(ctx SpecContext) {
			data := []byte(dbcHeader + `BO_ 2147483904 M: 8 Ecu
 SG_ A m1 : 16|8@1+ (1,0) [0|0] "" Ecu
 SG_ T M : 8|8@1+ (1,0) [0|0] "" Ecu
 SG_ S M : 0|8@1+ (1,0) [0|0] "" Ecu

SG_MUL_VAL_ 2147483904 A T 1-1;
SG_MUL_VAL_ 2147483904 T S 0-0;
`)
			m := messageNamed(MustSucceed(importFile(ctx, icd.FormatDBC, data)), "M")
			a, t, s := fieldNamed(m, "A"), fieldNamed(m, "T"), fieldNamed(m, "S")
			Expect(a.Multiplexor).To(Equal(new(t.Key)))
			Expect(a.MultiplexValues).To(Equal([]int32{1}))
			Expect(t.Multiplexor).To(Equal(new(s.Key)))
			Expect(t.MultiplexValues).To(Equal([]int32{0}))
			Expect(s.Multiplexor).To(BeNil())
		},
	)

	DescribeTable("Should skip the message of independent signals, whatever its ID",
		func(ctx SpecContext, id string) {
			data := []byte(dbcHeader + "BO_ " + id + ` VECTOR__INDEPENDENT_SIG_MSG: 0 X
 SG_ A : 0|8@1+ (1,0) [0|0] "" X

BO_ 256 M: 8 Ecu
 SG_ B : 0|8@1+ (1,0) [0|0] "" Ecu
`)
			messages := MustSucceed(importFile(ctx, icd.FormatDBC, data))
			Expect(messages).To(HaveLen(1))
			Expect(messages[0].Name).To(Equal("M"))
		},
		Entry("with the extended flag", "3221225472"),
		Entry("without the extended flag", "1073741824"),
	)

	DescribeTable(
		"Should reject an invalid file",
		func(ctx SpecContext, body, message string) {
			err := MatchError(SatisfyAll(
				HavePrefix("data: "+message),
				HaveSuffix(": validation error"),
			))
			Expect(
				importFile(ctx, icd.FormatDBC, []byte(dbcHeader+body)),
			).Error().
				To(err)
		},
		Entry("a syntax error", `BO_ 256 Engine 8 Ecu
`, "line 9, column 16: expected token"),
		Entry("a syntax error after an SG_MUL_VAL_ line", `BO_ 256 Engine: 8 Ecu
 SG_ S M : 0|8@1+ (1,0) [0|255] "" Ecu
 SG_ A m0 : 8|8@1+ (1,0) [0|255] "" Ecu
SG_MUL_VAL_ 256 A S 0-0;
BO_ x
`, "line 13, column 5"),
		Entry("a standard ID past 11 bits", `BO_ 2048 Engine: 8 Ecu
`, "line 9: standard ID 0x800 has more than 11 bits in message Engine"),
		Entry("an extended ID past 29 bits", `BO_ 4294967295 Engine: 8 Ecu
`, "line 9: extended ID 0x7FFFFFFF has more than 29 bits in message Engine"),
		Entry("a little-endian signal past the message", `BO_ 256 Engine: 2 Ecu
 SG_ A : 8|9@1+ (1,0) [0|0] "" Ecu
`, "line 10: signal A extends past the 2-byte message Engine"),
		Entry("a big-endian signal past the message", `BO_ 256 Engine: 2 Ecu
 SG_ A : 7|17@0+ (1,0) [0|0] "" Ecu
`, "line 10: signal A extends past the 2-byte message Engine"),
		Entry("a multiplexed signal past the message", `BO_ 256 Engine: 1 Ecu
 SG_ S M : 0|4@1+ (1,0) [0|0] "" Ecu
 SG_ A m1 : 4|8@1+ (1,0) [0|0] "" Ecu
`, "line 11: signal A extends past the 1-byte message Engine"),
		Entry("a message with two signals of the same name", `BO_ 256 Engine: 8 Ecu
 SG_ A : 0|8@1+ (1,0) [0|255] "" Ecu
 SG_ A : 8|8@1+ (1,0) [0|255] "" Ecu
`, "line 11: message Engine has more than one signal named A"),
		Entry("a malformed SG_MUL_VAL_ line", `SG_MUL_VAL_ 256 A;
`, "line 9, column 18: expected a name"),
		Entry("an inverted multiplexor range", `SG_MUL_VAL_ 256 A S 3-1;
`, `line 9: invalid multiplexor range "3-1"`),
		Entry("a multiplexor range past int32", `SG_MUL_VAL_ 256 A S 0-4294967296;
`, `line 9: multiplexor range "0-4294967296" of signal A exceeds the int32 range`),
		Entry("multiplexor ranges that are too large", `SG_MUL_VAL_ 256 A S 0-70000;
`, "line 9: the multiplexor ranges of the file hold more than 65536 values"),
		Entry(
			"multiplexor ranges across lines that are too large",
			`SG_MUL_VAL_ 256 A S 0-40000;
SG_MUL_VAL_ 256 B S 0-40000;
`,
			"line 10: the multiplexor ranges of the file hold more than 65536 values",
		),
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
	)
})
