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
	"context"
	"io"
	"math"
	"slices"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

const importDBC = `VERSION ""

NS_ :

BS_:

BU_: Ecu

BO_ 256 Engine: 8 Ecu
 SG_ Rpm : 0|16@1+ (0.25,0) [0|16383.75] "rpm" Ecu
 SG_ Gear : 16|8@1+ (1,0) [0|255] "" Ecu

VAL_ 256 Gear 0 "Park" 1 "Drive" ;
`

// importFile creates an ICD, imports data in the given format into it, and returns the
// messages of the result.
func importFile(
	ctx context.Context,
	format icd.Format,
	data []byte,
) ([]icd.Message, error) {
	GinkgoHelper()
	ic := icd.ICD{Name: "Imported"}
	Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
	res, err := svc.NewWriter(tx).Import(ctx, ic.Key, format, data)
	return res.Messages, err
}

// checkImport fails t unless err is a validation error at the data path, or the
// messages have unique keys, references that resolve, and finite numbers.
func checkImport(t *testing.T, messages []icd.Message, err error) {
	t.Helper()
	if err != nil {
		var pathErr validate.PathError
		if !errors.As(err, &pathErr) || !slices.Equal(pathErr.Path, []string{"data"}) ||
			!errors.Is(pathErr.Err, validate.ErrValidation) {
			t.Fatalf("not a validation error at the data path: %v", err)
		}
		return
	}
	keys := make(set.Set[icd.MessageKey], len(messages))
	for _, m := range messages {
		if keys.Contains(m.Key) {
			t.Fatalf("duplicate message key %s", m.Key)
		}
		keys.Add(m.Key)
		p, ok := m.Payload.Variant.(icd.BinaryPayload)
		if !ok {
			t.Fatalf("message %s has payload %T", m.Name, m.Payload.Variant)
		}
		fields := make(set.Set[icd.FieldKey], len(p.Fields))
		for _, f := range p.Fields {
			if fields.Contains(f.Key) {
				t.Fatalf("message %s has duplicate field key %s", m.Name, f.Key)
			}
			fields.Add(f.Key)
		}
		for _, f := range p.Fields {
			for _, v := range []*float64{&f.Scale, &f.Offset, f.Min, f.Max} {
				if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
					t.Fatalf(
						"field %s.%s has a number that is not finite",
						m.Name,
						f.Name,
					)
				}
			}
			if f.Multiplexor != nil && !fields.Contains(*f.Multiplexor) {
				t.Fatalf("field %s.%s has no multiplexor field", m.Name, f.Name)
			}
		}
	}
}

// fuzzImport opens a Service for f and returns a fuzz function that imports its input
// in the given format into a new ICD and checks the result with checkImport. Nothing it
// imports persists.
func fuzzImport(f *testing.F, format icd.Format) func(*testing.T, []byte) {
	closeOnCleanup := func(c io.Closer) {
		f.Cleanup(func() {
			if err := c.Close(); err != nil {
				f.Error(err)
			}
		})
	}
	db := gorp.Wrap(memkv.New())
	closeOnCleanup(db)
	s, err := icd.OpenService(f.Context(), icd.ServiceConfig{
		DB:   db,
		ImEx: imex.NewService(),
	})
	if err != nil {
		f.Fatal(err)
	}
	closeOnCleanup(s)
	return func(t *testing.T, data []byte) {
		tx := db.OpenTx()
		defer func() {
			if err := tx.Close(); err != nil {
				t.Error(err)
			}
		}()
		w := s.NewWriter(tx)
		ic := icd.ICD{Name: "Fuzz"}
		if err := w.Create(t.Context(), &ic); err != nil {
			t.Fatal(err)
		}
		res, err := w.Import(t.Context(), ic.Key, format, data)
		checkImport(t, res.Messages, err)
	}
}

func messageKeys(messages []icd.Message) set.Set[uuid.UUID] {
	keys := make(set.Set[uuid.UUID])
	for _, m := range messages {
		keys.Add(m.Key)
		for _, b := range m.Payload.FieldBases() {
			keys.Add(b.Key)
		}
	}
	return keys
}

var _ = Describe("Import", func() {
	var ic icd.ICD
	BeforeEach(func(ctx SpecContext) {
		ic = icd.ICD{
			Name: "Bus",
			Messages: []icd.Message{
				canMessage("Engine", 0x100, binaryField("Rpm", 0, 16)),
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
	})

	importICD := func(
		ctx SpecContext,
		key icd.Key,
		format icd.Format,
		data string,
	) (icd.ICD, error) {
		return svc.NewWriter(tx).Import(ctx, key, format, []byte(data))
	}

	DescribeTable("Should import each format",
		func(ctx SpecContext, format icd.Format, data func() []byte, count int) {
			Expect(MustSucceed(importFile(ctx, format, data()))).To(HaveLen(count))
		},
		Entry("DBC", icd.FormatDBC, func() []byte {
			data, _ := readFixture("basic")
			return data
		}, 5),
		Entry("CSV", icd.FormatCSV, func() []byte { return toCSV(tableRows) }, 3),
		Entry("XLSX", icd.FormatXLSX, func() []byte { return toXLSX(tableRows) }, 3),
	)

	It("Should reject an unknown format", func(ctx SpecContext) {
		Expect(importICD(ctx, ic.Key, "arxml", "")).Error().To(MatchError(
			`format: unsupported format "arxml": validation error`,
		))
	})

	It("Should replace the messages of the ICD", func(ctx SpecContext) {
		res := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, importDBC))
		Expect(res.Key).To(Equal(ic.Key))
		Expect(res.Name).To(Equal("Bus"))
		Expect(res.Messages).To(HaveLen(1))
		m := res.Messages[0]
		Expect(m.Name).To(Equal("Engine"))
		Expect(binaryFieldOf(m, 1).Name).To(Equal("Gear"))
		Expect(retrieveICD(ctx, ic.Key)).To(Equal(res))
	})

	DescribeTable("Should keep every key when a file is imported again",
		func(ctx SpecContext, name string) {
			data, _ := readFixture(name)
			first := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, string(data)))
			second := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, string(data)))
			Expect(second).To(Equal(first))
		},
		Entry("value tables", "basic"),
		Entry("multiplexor references", "extended_multiplexed"),
	)

	It("Should keep a key for each field that repeats a name", func(
		ctx SpecContext,
	) {
		csv := "message,field,start_bit,bit_length\nm,a,0,8\nm,a,8,8\nm,b,16,8\n" +
			"n,c,0,8\nn,c,8,8\n"
		first := MustSucceed(importICD(ctx, ic.Key, icd.FormatCSV, csv))
		second := MustSucceed(importICD(ctx, ic.Key, icd.FormatCSV, csv))
		Expect(second).To(Equal(first))
		Expect(messageKeys(second.Messages)).To(HaveLen(7))
	})

	It("Should drop missing messages and fields and add new ones", func(
		ctx SpecContext,
	) {
		first := MustSucceed(importICD(ctx, ic.Key, icd.FormatCSV,
			"message,field,start_bit,bit_length\n"+
				"Engine,rpm,0,8\nEngine,temp,8,8\nBrakes,pressure,0,8\n",
		))
		second := MustSucceed(importICD(ctx, ic.Key, icd.FormatCSV,
			"message,field,start_bit,bit_length\n"+
				"Engine,rpm,16,8\nEngine,load,24,8\nDoors,open,0,1\n",
		))
		oldKeys := messageKeys(first.Messages)
		Expect(second.Messages).To(HaveLen(2))
		engine := messageNamed(second.Messages, "Engine")
		Expect(engine.Key).To(Equal(messageNamed(first.Messages, "Engine").Key))
		Expect(fieldNamed(engine, "rpm").Key).
			To(Equal(fieldNamed(messageNamed(first.Messages, "Engine"), "rpm").Key))
		Expect(fieldNamed(engine, "rpm").StartBit).To(Equal(uint16(16)))
		Expect(oldKeys.Contains(fieldNamed(engine, "load").Key)).To(BeFalse())
		Expect(oldKeys.Contains(messageNamed(second.Messages, "Doors").Key)).
			To(BeFalse())
	})

	It("Should keep the keys of the fields of a text message", func(ctx SpecContext) {
		line := textMessage("Line", icd.TextField{Variant: icd.TaggedTextField{
			BaseField: icd.BaseField{Name: "temp", Scale: 1},
			Tag:       "T=",
		}})
		text := icd.ICD{Name: "Text", Messages: []icd.Message{line}}
		Expect(svc.NewWriter(tx).Create(ctx, &text)).To(Succeed())
		res := MustSucceed(importICD(ctx, text.Key, icd.FormatCSV,
			"message,field,start_bit,bit_length\nLine,temp,0,8\nLine,rh,8,8\n",
		))
		m := messageNamed(res.Messages, "Line")
		Expect(m.Key).To(Equal(text.Messages[0].Key))
		oldTemp := text.Messages[0].Payload.FieldBases()[0].Key
		Expect(fieldNamed(m, "temp").Key).To(Equal(oldTemp))
		Expect(fieldNamed(m, "rh").Key).ToNot(Equal(oldTemp))
	})

	It("Should reject data that does not parse", func(ctx SpecContext) {
		Expect(importICD(ctx, ic.Key, icd.FormatCSV, "color\n")).Error().To(
			MatchError(`data: row 1: unknown column "color": validation error`),
		)
	})

	It("Should store an import that Check finds problems in", func(ctx SpecContext) {
		csv := "message,field,start_bit,bit_length\nm,a,0,8\nm,a,8,8\n"
		res := MustSucceed(importICD(ctx, ic.Key, icd.FormatCSV, csv))
		Expect(retrieveICD(ctx, ic.Key)).To(Equal(res))
		Expect(icd.Check(res)).To(ConsistOf(validate.Problem{
			Path:    "messages.0.payload.fields.1.name",
			Message: `duplicate field name "a"`,
		}))
	})

	It("Should return not found for a missing ICD", func(ctx SpecContext) {
		Expect(importICD(ctx, uuid.New(), icd.FormatDBC, importDBC)).Error().
			To(MatchError(query.ErrNotFound))
	})
})
