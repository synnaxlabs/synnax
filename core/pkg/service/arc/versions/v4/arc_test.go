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
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc", func() {
	base := v4.Base{Key: uuid.New(), Name: "m", Mode: v4.ModeText}
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

	Describe("Type", func() {
		It("Should return the type of each variant", func() {
			Expect(automation.Type()).To(Equal(v4.TypeAutomation))
			Expect(module.Type()).To(Equal(v4.TypeModule))
		})

		It("Should return no type for an Arc without a variant", func() {
			Expect(v4.Arc{}.Type()).To(BeEmpty())
		})
	})

	Describe("WithBase", func() {
		renamed := base
		renamed.Name = "renamed"

		It("Should replace the base and keep an automation's program", func() {
			got := automation.WithBase(renamed)
			a, ok := got.Variant.(v4.AutomationArc)
			Expect(ok).To(BeTrue())
			Expect(a.Base).To(Equal(renamed))
			Expect(a.Program).ToNot(BeNil())
		})

		It("Should replace the base of a module", func() {
			got := module.WithBase(renamed)
			Expect(got.Variant).To(Equal(v4.ModuleArc{Base: renamed}))
		})

		It("Should leave an Arc without a variant unchanged", func() {
			Expect(v4.Arc{}.WithBase(renamed)).To(Equal(v4.Arc{}))
		})
	})

	Describe("ExportBody", func() {
		It("Should carry the Arc's type as the file type header", func() {
			var body map[string]any
			raw := MustSucceed(json.Marshal(module.ExportBody()))
			Expect(json.Unmarshal(raw, &body)).To(Succeed())
			Expect(body).To(HaveKeyWithValue("type", "arc-module"))
			Expect(body).To(HaveKeyWithValue("name", "m"))
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
