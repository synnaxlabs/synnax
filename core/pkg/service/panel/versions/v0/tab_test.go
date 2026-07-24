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
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
)

var _ = Describe("Tab", func() {
	Describe("Key", func() {
		It("Should return the key of a resource-backed tab", func() {
			k := uuid.New()
			variant := v0.TabResource{TabBase: v0.TabBase{Key: k}}
			Expect(v0.Tab{Variant: variant}.Key()).To(Equal(k))
		})

		It("Should return the key of a view-backed tab", func() {
			k := uuid.New()
			variant := v0.TabView{TabBase: v0.TabBase{Key: k}}
			Expect(v0.Tab{Variant: variant}.Key()).To(Equal(k))
		})

		It("Should return uuid.Nil for a tab with no variant", func() {
			Expect(v0.Tab{}.Key()).To(Equal(uuid.Nil))
		})
	})
})
