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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/kv/types/v0"
)

var _ = Describe("Pair", func() {
	Describe("GorpKey", func() {
		It("Should join the range key and pair key", func() {
			rng := uuid.New()
			Expect(v0.Pair{Range: rng, Key: "temperature"}.GorpKey()).
				To(Equal(rng.String() + "<--->" + "temperature"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Pair{}.SetOptions()).To(BeNil())
		})
	})
})
