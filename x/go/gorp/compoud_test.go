// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("CompoundRetrieve", func() {
	Describe("Next", func() {
		It("Should return the next RetrieveP Clause", func() {
			c := &gorp.CompoundRetrieve[int32, entry]{}
			r := c.Next()
			Expect(r).To(Equal(c.Clauses[0]))
		})
	})
	Describe("Current", func() {
		It("Should return the current RetrieveP Clause", func() {
			c := &gorp.CompoundRetrieve[int32, entry]{}
			nr := c.Next()
			r := c.Current()
			Expect(r).To(Equal(nr))
		})
	})

})
