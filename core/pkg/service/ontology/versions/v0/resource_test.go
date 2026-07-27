// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
)

var _ = Describe("Resource", func() {
	Describe("GorpKey", func() {
		It("Should return the resource ID formatted as type:key", func() {
			r := v0.Resource{ID: v0.ID{Type: "channel", Key: "qux"}}
			Expect(r.GorpKey()).To(Equal("channel:qux"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Resource{}.SetOptions()).To(BeNil())
		})
	})
})
