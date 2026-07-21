// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v2"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Arc", func() {
	Describe("GorpKey", func() {
		It("Should return the arc's key", func() {
			k := uuid.New()
			Expect(v2.Arc{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v2.Arc{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the arc ontology identifier", func() {
			k := uuid.New()
			Expect(v2.Arc{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeArc, Key: k.String(),
			}))
		})
	})
})

var _ = Describe("StatusDetails DecodeMsgpack", func() {
	It("Should decode new lowercase msgpack fields", func() {
		original := v2.StatusDetails{Running: true}
		data := MustSucceed(msgpack.Marshal(original))
		var decoded v2.StatusDetails
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Running).To(BeTrue())
	})
	It("Should decode legacy uppercase Go field name", func() {
		legacy := struct{ Running bool }{Running: true}
		data := MustSucceed(msgpack.Marshal(legacy))
		var decoded v2.StatusDetails
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Running).To(BeTrue())
	})
	It("Should handle false value correctly for both formats", func() {
		original := v2.StatusDetails{Running: false}
		data := MustSucceed(msgpack.Marshal(original))
		var decoded v2.StatusDetails
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Running).To(BeFalse())
	})
})
