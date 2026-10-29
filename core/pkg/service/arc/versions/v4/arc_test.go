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
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc", func() {
	Describe("OntologyID", func() {
		It("Should return the Arc ontology identifier", func() {
			k := uuid.New()
			Expect(v4.Arc{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeArc,
				Key:  k.String(),
			}))
		})
	})

	Describe("ExportBody", func() {
		It("Should carry the Arc's type as the file type header", func() {
			var body map[string]any
			raw := MustSucceed(json.Marshal(
				v4.Arc{Name: "m", Type: v4.TypeModule}.ExportBody(),
			))
			Expect(json.Unmarshal(raw, &body)).To(Succeed())
			Expect(body).To(HaveKeyWithValue("type", "arc-module"))
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
