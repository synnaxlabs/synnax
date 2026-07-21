// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/v6"
)

var _ = Describe("LinePlot", func() {
	Describe("GorpKey", func() {
		It("Should return the linePlot's key", func() {
			k := uuid.New()
			Expect(v6.LinePlot{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v6.LinePlot{}.SetOptions()).To(BeNil())
		})
	})
})
