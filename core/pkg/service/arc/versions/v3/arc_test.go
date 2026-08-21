// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/program"
	text "github.com/synnaxlabs/arc/text/versions/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Arc", func() {
	Describe("GorpKey", func() {
		It("Should return the Arc's key", func() {
			k := uuid.New()
			Expect(v3.Arc{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v3.Arc{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the Arc ontology identifier", func() {
			k := uuid.New()
			Expect(v3.Arc{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeArc, Key: k.String(),
			}))
		})
	})
})

var _ = Describe("ExportBody", func() {
	body := func(a v3.Arc) map[string]any {
		GinkgoHelper()
		data := MustSucceed(json.Marshal(a.ExportBody()))
		var m map[string]any
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		return m
	}

	It("Should carry the text as materialized source", func() {
		a := v3.Arc{
			Name: "startup",
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create("x := 1")},
		}
		Expect(body(a)["text"]).To(Equal(map[string]any{"raw": "x := 1"}))
	})

	It("Should carry the name, mode, and graph", func() {
		a := v3.Arc{Name: "abort", Mode: v3.ModeGraph}
		m := body(a)
		Expect(m["name"]).To(Equal("abort"))
		Expect(m["mode"]).To(BeEquivalentTo(v3.ModeGraph))
		Expect(m).To(HaveKey("graph"))
	})

	It("Should leave off the key and the derived program and status", func() {
		a := v3.Arc{
			Key:     uuid.New(),
			Name:    "derived",
			Mode:    v3.ModeText,
			Program: &program.Program{},
			Status:  &v3.Status{},
		}
		Expect(body(a)).To(SatisfyAll(
			Not(HaveKey("key")),
			Not(HaveKey("program")),
			Not(HaveKey("status")),
		))
	})
})

var _ = Describe("StatusDetails", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := v3.StatusDetails{Running: true}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v3.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should decode legacy uppercase Go field name", func() {
			legacy := struct{ Running bool }{Running: true}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded v3.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeTrue())
		})
		It("Should handle false value correctly for both formats", func() {
			original := v3.StatusDetails{Running: false}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v3.StatusDetails
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Running).To(BeFalse())
		})
	})
})
