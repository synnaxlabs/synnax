// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/text"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc", func() {
	base := v4.Base{
		Key:  uuid.New(),
		Name: "m",
		Source: v4.Source{
			Variant: v4.TextSource{Text: text.Text{Doc: text.Create("x := 1")}},
		},
	}
	module := v4.Arc{Variant: v4.ModuleArc{Base: base}}
	automation := v4.Arc{Variant: v4.AutomationArc{
		Base:    base,
		Program: &program.Program{},
	}}

	Describe("OntologyID", func() {
		It("Should return the Arc ontology identifier", func() {
			Expect(module.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeArc,
				Key:  base.Key.String(),
			}))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the key of the Arc's base", func() {
			Expect(automation.GorpKey()).To(Equal(base.Key))
		})
	})

	Describe("ExportBody", func() {
		It("Should carry the Arc's type as the file type header", func() {
			var body map[string]any
			raw := MustSucceed(json.Marshal(module.ExportBody()))
			Expect(json.Unmarshal(raw, &body)).To(Succeed())
			Expect(body).To(HaveKeyWithValue("type", "arc-module"))
			Expect(body).To(HaveKeyWithValue("name", "m"))
			Expect(body).To(HaveKeyWithValue("source", map[string]any{
				"mode": "text",
				"text": map[string]any{"raw": "x := 1"},
			}))
		})
	})

	DescribeTable("TypeFromFileType",
		func(header string, want v4.Type, wantOK bool) {
			t, ok := v4.TypeFromFileType(header)
			Expect(ok).To(Equal(wantOK))
			Expect(t).To(Equal(want))
		},
		Entry("automation", "arc-automation", v4.TypeAutomation, true),
		Entry("module", "arc-module", v4.TypeModule, true),
		Entry("header before v4", "arc", v4.Type(""), false),
		Entry("unknown type", "arc-thing", v4.Type(""), false),
	)
})
