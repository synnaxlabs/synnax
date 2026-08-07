// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
)

var _ = Describe("Transition", func() {
	Describe("String", func() {
		It("Should render a transition targeting a sibling step", func() {
			target := "next"
			t := v1.Transition{
				On:        v1.Handle{Node: "n", Param: "done"},
				TargetKey: &target,
			}
			Expect(t.String()).To(Equal("on n/done => next"))
		})

		It("Should render an exiting transition when TargetKey is nil", func() {
			t := v1.Transition{On: v1.Handle{Node: "n", Param: "done"}}
			Expect(t.String()).To(Equal("on n/done => exit"))
		})
	})
})
