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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

func binaryField(name string, startBit uint16, bitLength uint8) icd.BinaryField {
	return icd.BinaryField{
		Name: name, Scale: 1,
		StartBit:  startBit,
		BitLength: bitLength,
		ByteOrder: icd.ByteOrderLittleEndian,
	}
}

func binaryMessage(name string, fields ...icd.BinaryField) icd.Message {
	return icd.Message{
		Name:    name,
		Payload: icd.Payload{Variant: icd.BinaryPayload{Fields: fields}},
	}
}

func textMessage(name string, fields ...icd.TextField) icd.Message {
	return icd.Message{
		Name: name,
		Payload: icd.Payload{Variant: icd.TextPayload{
			Delimiter: ",",
			Fields:    fields,
		}},
	}
}

// updateBinary returns m with its binary payload changed by update.
func updateBinary(
	m icd.Message,
	update func(*icd.BinaryPayload),
) icd.Message {
	p := binaryPayloadOf(m)
	update(&p)
	m.Payload.Variant = p
	return m
}

func canMessage(name string, id uint32, fields ...icd.BinaryField) icd.Message {
	return updateBinary(binaryMessage(name, fields...), func(p *icd.BinaryPayload) {
		p.Identifier = &icd.Identifier{Variant: icd.CanIdentifier{ID: id}}
		p.Length = new(uint16(8))
	})
}

func binaryPayloadOf(m icd.Message) icd.BinaryPayload {
	return m.Payload.Variant.(icd.BinaryPayload)
}

func binaryFieldOf(m icd.Message, i int) icd.BinaryField {
	return binaryPayloadOf(m).Fields[i]
}

// setScale returns a copy of ic with the scale of the first field of message i set.
func setScale(ic icd.ICD, i int, scale float64) icd.ICD {
	ic.Messages = append([]icd.Message(nil), ic.Messages...)
	ic.Messages[i] = updateBinary(
		ic.Messages[i],
		func(p *icd.BinaryPayload) {
			p.Fields = append([]icd.BinaryField(nil), p.Fields...)
			p.Fields[0].Scale = scale
		},
	)
	return ic
}

func retrieveICD(ctx SpecContext, key icd.Key) icd.ICD {
	GinkgoHelper()
	var res icd.ICD
	Expect(svc.NewRetrieve().
		Where(icd.MatchKeys(key)).
		Entry(&res).
		Exec(ctx, tx)).To(Succeed())
	return res
}

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It(
			"Should create an ICD with keys for it, its messages, and fields",
			func(ctx SpecContext) {
				ic := icd.ICD{
					Name: "Engine",
					Messages: []icd.Message{
						canMessage("Status", 0x100, binaryField("rpm", 0, 16)),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
				Expect(ic.Key).ToNot(Equal(uuid.Nil()))
				m := ic.Messages[0]
				Expect(m.Key).ToNot(Equal(uuid.Nil()))
				Expect(binaryFieldOf(m, 0).Key).ToNot(Equal(uuid.Nil()))
				Expect(retrieveICD(ctx, ic.Key)).To(Equal(ic))
			},
		)

		It("Should create an ICD under the key a client chose", func(ctx SpecContext) {
			key := uuid.New()
			ic := icd.ICD{Key: key, Name: "Chosen"}
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			Expect(retrieveICD(ctx, key).Name).To(Equal("Chosen"))
		})

		It(
			"Should store fields as sent, without filling defaults",
			func(ctx SpecContext) {
				m := textMessage("Line", icd.TextField{Variant: icd.DelimitedTextField{
					Name: "temperature", Scale: 2,
					Position: 1,
				}})
				ic := icd.ICD{Name: "Sensor", Messages: []icd.Message{m}}
				Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
				res := retrieveICD(ctx, ic.Key).Messages[0].Payload.Variant.(icd.TextPayload)
				Expect(res.Delimiter).To(Equal(","))
				Expect(
					res.Fields[0].Variant.(icd.DelimitedTextField).Scale,
				).To(Equal(2.0))
			},
		)

		It("Should refuse an ICD whose key exists", func(ctx SpecContext) {
			ic := icd.ICD{Name: "Stored"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &ic)).To(Succeed())
			again := icd.ICD{Key: ic.Key, Name: "Replaced"}
			Expect(w.Create(ctx, &again)).To(
				MatchError(ContainSubstring("icd %s already exists", ic.Key)),
			)
			Expect(retrieveICD(ctx, ic.Key).Name).To(Equal("Stored"))
		})

		It("Should store an ICD with problems that Check finds", func(ctx SpecContext) {
			ic := icd.ICD{Name: "Duplicates", Messages: []icd.Message{
				binaryMessage("State"),
				binaryMessage("State"),
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			Expect(retrieveICD(ctx, ic.Key)).To(Equal(ic))
			Expect(icd.Check(ic)).To(ConsistOf(validate.Problem{
				Path:    "messages.1.name",
				Message: `duplicate message name "State"`,
			}))
		})

		It("Should reject an ICD that breaks a rule of its schema", func(
			ctx SpecContext,
		) {
			ic := icd.ICD{Messages: []icd.Message{binaryMessage("")}}
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).
				To(MatchError("name: required\nmessages.0.name: required"))
			Expect(svc.NewRetrieve().
				Where(icd.MatchKeys(ic.Key)).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It("Should delete an ICD", func(ctx SpecContext) {
			ic := icd.ICD{Name: "Doomed"}
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &ic)).To(Succeed())
			Expect(w.Delete(ctx, ic.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(icd.MatchKeys(ic.Key)).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should be idempotent", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx, uuid.New())).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should find an ICD by a fuzzy search on its name", func(ctx SpecContext) {
			ic := icd.ICD{Name: "Flight Controls ICD"}
			Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
			var res []icd.ICD
			Expect(svc.NewRetrieve().
				Search("flight controls").
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveExactElements(HaveField("Key", ic.Key)))
		})
	})
})
