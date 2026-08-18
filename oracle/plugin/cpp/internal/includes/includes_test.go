// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package includes_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
)

var _ = Describe("Manager", func() {
	var mgr *includes.Manager

	BeforeEach(func() { mgr = includes.NewManager() })

	Describe("AddSystem", func() {
		It("should record system includes in first-added order", func() {
			mgr.AddSystem("vector")
			mgr.AddSystem("string")
			mgr.AddSystem("memory")
			Expect(mgr.SystemIncludes()).To(Equal([]string{
				"vector", "string", "memory",
			}))
		})

		It("should deduplicate repeated system includes", func() {
			mgr.AddSystem("vector")
			mgr.AddSystem("string")
			mgr.AddSystem("vector")
			Expect(mgr.SystemIncludes()).To(Equal([]string{"vector", "string"}))
		})
	})

	Describe("AddInternal", func() {
		It("should record internal includes in first-added order", func() {
			mgr.AddInternal("x/cpp/telem/telem.h")
			mgr.AddInternal("client/cpp/task/task.h")
			Expect(mgr.InternalIncludes()).To(Equal([]string{
				"x/cpp/telem/telem.h", "client/cpp/task/task.h",
			}))
		})

		It("should deduplicate repeated internal includes", func() {
			mgr.AddInternal("x/cpp/telem/telem.h")
			mgr.AddInternal("x/cpp/telem/telem.h")
			Expect(mgr.InternalIncludes()).To(Equal([]string{"x/cpp/telem/telem.h"}))
		})

		It("should keep system and internal includes separate", func() {
			mgr.AddSystem("vector")
			mgr.AddInternal("x/cpp/telem/telem.h")
			Expect(mgr.SystemIncludes()).To(Equal([]string{"vector"}))
			Expect(mgr.InternalIncludes()).To(Equal([]string{"x/cpp/telem/telem.h"}))
		})
	})

	Describe("HasIncludes", func() {
		It("should return false when nothing was recorded", func() {
			Expect(mgr.HasIncludes()).To(BeFalse())
		})

		It("should return true after a system include", func() {
			mgr.AddSystem("vector")
			Expect(mgr.HasIncludes()).To(BeTrue())
		})

		It("should return true after an internal include", func() {
			mgr.AddInternal("x/cpp/telem/telem.h")
			Expect(mgr.HasIncludes()).To(BeTrue())
		})
	})
})
