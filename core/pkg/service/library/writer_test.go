// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
)

func enumEntry(name string, values ...library.EnumValue) library.Entry {
	return library.Entry{Variant: library.EnumEntry{
		BaseEntry: library.BaseEntry{Name: name},
		Values:    values,
	}}
}

func binaryField(name string, startBit uint16, bitLength uint8) library.BinaryField {
	return library.BinaryField{
		Name:      name,
		StartBit:  startBit,
		BitLength: bitLength,
		ByteOrder: library.ByteOrderLittleEndian,
	}
}

func messageEntry(m library.MessageEntry) library.Entry {
	return library.Entry{Variant: m}
}

func binaryMessage(name string, fields ...library.BinaryField) library.MessageEntry {
	m := library.MessageEntry{
		Name:   name,
		Format: library.FormatBinary,
	}
	for _, f := range fields {
		m.Fields = append(m.Fields, library.Field{Variant: f})
	}
	return m
}

func canMessage(
	name string,
	id uint32,
	fields ...library.BinaryField,
) library.MessageEntry {
	m := binaryMessage(name, fields...)
	m.Identifier = &library.Identifier{Variant: library.CanIdentifier{ID: id}}
	m.Length = new(uint16(8))
	return m
}

func messageOf(l library.Library, i int) library.MessageEntry {
	return l.Entries[i].Variant.(library.MessageEntry)
}

func binaryFieldOf(m library.MessageEntry, i int) library.BinaryField {
	return m.Fields[i].Variant.(library.BinaryField)
}

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It(
			"Should create a library with keys for it, its entries, and fields",
			func(ctx SpecContext) {
				l := library.Library{
					Name: "Engine",
					Entries: []library.Entry{
						enumEntry("State", library.EnumValue{Value: 0, Name: "Off"}),
						messageEntry(
							canMessage("Status", 0x100, binaryField("rpm", 0, 16)),
						),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
				Expect(l.Key).ToNot(Equal(uuid.Nil()))
				Expect(
					l.Entries[0].Variant.(library.EnumEntry).Key,
				).ToNot(Equal(uuid.Nil()))
				m := messageOf(l, 1)
				Expect(m.Key).ToNot(Equal(uuid.Nil()))
				Expect(binaryFieldOf(m, 0).Key).ToNot(Equal(uuid.Nil()))
				var res library.Library
				Expect(svc.NewRetrieve().
					Where(library.MatchKeys(l.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal(l))
			},
		)

		It("Should apply schema defaults to entries and fields", func(ctx SpecContext) {
			m := library.MessageEntry{
				Name:   "Line",
				Format: library.FormatText,
				Fields: []library.Field{{Variant: library.DelimitedField{
					BaseField: library.BaseField{Name: "temperature"},
					Position:  1,
				}}},
			}
			l := library.Library{
				Name:    "Sensor",
				Entries: []library.Entry{messageEntry(m)},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			res := messageOf(l, 0)
			Expect(res.Delimiter).To(Equal(","))
			Expect(res.Fields[0].Variant.(library.DelimitedField).Scale).To(Equal(1.0))
		})

		It("Should define the library in the ontology", func(ctx SpecContext) {
			l := library.Library{Name: "Ontology"}
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(l.OntologyID()).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("Ontology"))
		})

		It(
			"Should keep entry and field keys when replacing a library",
			func(ctx SpecContext) {
				l := library.Library{
					Name: "Replace",
					Entries: []library.Entry{
						messageEntry(
							canMessage("Status", 0x100, binaryField("rpm", 0, 16)),
						),
					},
				}
				w := svc.NewWriter(tx)
				Expect(w.Create(ctx, &l)).To(Succeed())
				entryKey := messageOf(l, 0).Key
				fieldKey := binaryFieldOf(messageOf(l, 0), 0).Key
				m := messageOf(l, 0)
				f := binaryFieldOf(m, 0)
				f.Scale = 0.25
				m.Fields[0].Variant = f
				l.Entries[0].Variant = m
				Expect(w.Create(ctx, &l)).To(Succeed())
				Expect(messageOf(l, 0).Key).To(Equal(entryKey))
				Expect(binaryFieldOf(messageOf(l, 0), 0).Key).To(Equal(fieldKey))
				Expect(binaryFieldOf(messageOf(l, 0), 0).Scale).To(Equal(0.25))
			},
		)

		DescribeTable("Should reject an invalid library",
			func(ctx SpecContext, l library.Library, msg string) {
				Expect(svc.NewWriter(tx).Create(ctx, &l)).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("missing name", library.Library{}, "name: required"),
			Entry("entry without a kind",
				library.Library{Name: "L", Entries: []library.Entry{{}}},
				"kind is required",
			),
			Entry("duplicate entry names",
				library.Library{Name: "L", Entries: []library.Entry{
					enumEntry("State"),
					enumEntry("State"),
				}},
				`duplicate entry name "State"`,
			),
			Entry("duplicate enum values",
				library.Library{Name: "L", Entries: []library.Entry{enumEntry(
					"State",
					library.EnumValue{Value: 1, Name: "On"},
					library.EnumValue{Value: 1, Name: "Running"},
				)}},
				"duplicate value 1",
			),
			Entry("duplicate enum names",
				library.Library{Name: "L", Entries: []library.Entry{enumEntry(
					"State",
					library.EnumValue{Value: 1, Name: "On"},
					library.EnumValue{Value: 2, Name: "On"},
				)}},
				`duplicate name "On"`,
			),
			Entry("duplicate field names",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					canMessage("M", 1, binaryField("a", 0, 8), binaryField("a", 8, 8)),
				)}},
				`duplicate field name "a"`,
			),
			Entry("bit length over 64",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					binaryMessage("M", binaryField("a", 0, 65)),
				)}},
				"bit_length must be between 1 and 64",
			),
			Entry("float field of 24 bits",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					binaryMessage("M", func() library.BinaryField {
						f := binaryField("a", 0, 24)
						f.Float = true
						return f
					}()),
				)}},
				"a float field must be 32 or 64 bits",
			),
			Entry("little-endian field past the payload",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					canMessage("M", 1, binaryField("a", 60, 8)),
				)}},
				"field extends past the 8-byte payload",
			),
			Entry("big-endian field past the payload",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					canMessage("M", 1, func() library.BinaryField {
						f := binaryField("a", 56, 16)
						f.ByteOrder = library.ByteOrderBigEndian
						return f
					}()),
				)}},
				"field extends past the 8-byte payload",
			),
			Entry("standard CAN identifier over 11 bits",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					canMessage("M", 0x800),
				)}},
				"exceeds the maximum 0x7FF",
			),
			Entry("classic CAN frame over 8 bytes",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					func() library.MessageEntry {
						m := canMessage("M", 1)
						m.Length = new(uint16(12))
						return m
					}(),
				)}},
				"exceeds the 8 bytes a CAN frame carries",
			),
			Entry("1553 word count over 32",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					func() library.MessageEntry {
						m := binaryMessage("M")
						m.Identifier = &library.Identifier{
							Variant: library.Mil1553Identifier{
								Rt:        1,
								Direction: library.DirectionReceive,
								WordCount: 33,
							},
						}
						return m
					}(),
				)}},
				"word_count must be between 1 and 32",
			),
			Entry("ARINC 429 SDI over 3",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					func() library.MessageEntry {
						m := binaryMessage("M")
						m.Identifier = &library.Identifier{
							Variant: library.Arinc429Identifier{
								Label: 0o203,
								Sdi:   4,
							},
						}
						return m
					}(),
				)}},
				"sdi must be between 0 and 3",
			),
			Entry("token identifier on a binary message",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					func() library.MessageEntry {
						m := binaryMessage("M")
						m.Identifier = &library.Identifier{
							Variant: library.TokenIdentifier{Prefix: "$"},
						}
						return m
					}(),
				)}},
				"identifier does not apply to a binary message",
			),
			Entry("field identifier naming a missing field",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					func() library.MessageEntry {
						m := binaryMessage("M", binaryField("a", 0, 8))
						m.Identifier = &library.Identifier{
							Variant: library.FieldIdentifier{
								Field: uuid.New(),
								Value: 1,
							},
						}
						return m
					}(),
				)}},
				"no field with key",
			),
			Entry("text field in a binary message",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					library.MessageEntry{
						Name:   "M",
						Format: library.FormatBinary,
						Fields: []library.Field{{Variant: library.DelimitedField{
							BaseField: library.BaseField{Name: "a"},
						}}},
					},
				)}},
				"a text field requires a text message",
			),
			Entry("tagged field without a tag",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					library.MessageEntry{
						Name:   "M",
						Format: library.FormatText,
						Fields: []library.Field{{Variant: library.TaggedField{
							BaseField: library.BaseField{Name: "a"},
						}}},
					},
				)}},
				"tag is required",
			),
			Entry("enumeration naming a missing enum",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					binaryMessage("M", func() library.BinaryField {
						f := binaryField("a", 0, 8)
						f.Enumeration = new(uuid.New())
						return f
					}()),
				)}},
				"no enum entry with key",
			),
			Entry("field multiplexing itself",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					binaryMessage("M", func() library.BinaryField {
						f := binaryField("a", 0, 8)
						f.Key = uuid.New()
						f.Multiplexor = new(f.Key)
						f.MultiplexValues = []int32{1}
						return f
					}()),
				)}},
				"a field cannot multiplex itself",
			),
			Entry("multiplexed field without values",
				library.Library{Name: "L", Entries: []library.Entry{messageEntry(
					func() library.MessageEntry {
						mux := binaryField("mux", 0, 8)
						mux.Key = uuid.New()
						sig := binaryField("sig", 8, 8)
						sig.Multiplexor = new(mux.Key)
						return binaryMessage("M", mux, sig)
					}(),
				)}},
				"at least one value is required",
			),
		)

		It(
			"Should accept a multiplexed message referencing an enum",
			func(ctx SpecContext) {
				state := enumEntry("State", library.EnumValue{Value: 1, Name: "On"})
				stateKey := uuid.New()
				en := state.Variant.(library.EnumEntry)
				en.Key = stateKey
				state.Variant = en
				mux := binaryField("mux", 0, 8)
				mux.Key = uuid.New()
				sig := binaryField("sig", 8, 8)
				sig.Multiplexor = new(mux.Key)
				sig.MultiplexValues = []int32{0, 2}
				sig.Enumeration = new(stateKey)
				l := library.Library{
					Name: "Mux",
					Entries: []library.Entry{
						state,
						messageEntry(canMessage("M", 1, mux, sig)),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			},
		)
	})

	Describe("Rename", func() {
		It("Should rename a library", func(ctx SpecContext) {
			l := library.Library{Name: "Before"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &l)).To(Succeed())
			Expect(w.Rename(ctx, l.Key, "After")).To(Succeed())
			var res library.Library
			Expect(svc.NewRetrieve().
				Where(library.MatchKeys(l.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("After"))
		})

		It("Should reject an empty name", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Rename(ctx, uuid.New(), "")).
				To(MatchError(ContainSubstring("name: name is required: validation error")))
		})
	})

	Describe("Delete", func() {
		It("Should delete a library and its ontology resource", func(ctx SpecContext) {
			l := library.Library{Name: "Doomed"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &l)).To(Succeed())
			Expect(w.Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(library.MatchKeys(l.Key)).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			Expect(otg.NewRetrieve().
				WhereIDs(l.OntologyID()).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It(
			"Should find a library by a fuzzy search on its name",
			func(ctx SpecContext) {
				l := library.Library{Name: "Flight Controls ICD"}
				Expect(svc.NewWriter(nil).Create(ctx, &l)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					Expect(svc.NewWriter(nil).Delete(ctx, l.Key)).To(Succeed())
				})
				Eventually(func(g Gomega) {
					var res []library.Library
					g.Expect(svc.NewRetrieve().
						Search("flight controls").
						Entries(&res).
						Exec(ctx, nil)).To(Succeed())
					g.Expect(res).To(ContainElement(HaveField("Key", l.Key)))
				}).Should(Succeed())
			},
		)
	})
})

var _ = Describe("Payload bounds", func() {
	DescribeTable(
		"Should require the payload to reach the last byte a binary field touches",
		func(
			ctx SpecContext,
			startBit uint16,
			bitLength uint8,
			order library.ByteOrder,
			lastByte uint16,
		) {
			create := func(length uint16) error {
				GinkgoHelper()
				f := binaryField("a", startBit, bitLength)
				f.ByteOrder = order
				m := binaryMessage("M", f)
				m.Length = new(length)
				l := library.Library{
					Name:    "L",
					Entries: []library.Entry{messageEntry(m)},
				}
				return svc.NewWriter(tx).Create(ctx, &l)
			}
			Expect(create(lastByte + 1)).To(Succeed())
			Expect(create(lastByte)).To(MatchError(ContainSubstring(fmt.Sprintf(
				"entries.0.fields.0.start_bit: field extends past the %d-byte payload",
				lastByte,
			))))
		},
		Entry(
			"little-endian byte",
			uint16(0),
			uint8(8),
			library.ByteOrderLittleEndian,
			uint16(0),
		),
		Entry(
			"little-endian across bytes",
			uint16(4),
			uint8(8),
			library.ByteOrderLittleEndian,
			uint16(1),
		),
		Entry(
			"little-endian last bit",
			uint16(63),
			uint8(1),
			library.ByteOrderLittleEndian,
			uint16(7),
		),
		Entry(
			"big-endian byte",
			uint16(7),
			uint8(8),
			library.ByteOrderBigEndian,
			uint16(0),
		),
		Entry(
			"big-endian word",
			uint16(7),
			uint8(16),
			library.ByteOrderBigEndian,
			uint16(1),
		),
		Entry(
			"big-endian from a mid-byte MSB",
			uint16(3),
			uint8(8),
			library.ByteOrderBigEndian,
			uint16(1),
		),
		Entry(
			"big-endian last byte",
			uint16(63),
			uint8(8),
			library.ByteOrderBigEndian,
			uint16(7),
		),
	)
})
