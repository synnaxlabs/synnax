// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imports_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
)

var _ = Describe("Manager", func() {
	var mgr *imports.Manager

	BeforeEach(func() {
		mgr = imports.NewManager()
	})

	Describe("HasImports", func() {
		It("should return false when empty", func() {
			Expect(mgr.HasImports()).To(BeFalse())
		})

		It("should return true after adding an external import", func() {
			mgr.AddExternal("fmt")
			Expect(mgr.HasImports()).To(BeTrue())
		})

		It("should return true after adding an internal import", func() {
			mgr.AddInternal("user", "core/pkg/user")
			Expect(mgr.HasImports()).To(BeTrue())
		})
	})

	Describe("ExternalImports", func() {
		It("should return sorted external imports", func() {
			mgr.AddExternal("fmt")
			mgr.AddExternal("context")
			mgr.AddExternal("strings")
			Expect(mgr.ExternalImports()).To(Equal([]string{
				"context", "fmt", "strings",
			}))
		})

		It("should return empty slice when no external imports exist", func() {
			Expect(mgr.ExternalImports()).To(BeEmpty())
		})

		It("should deduplicate external imports", func() {
			mgr.AddExternal("fmt")
			mgr.AddExternal("fmt")
			Expect(mgr.ExternalImports()).To(HaveLen(1))
		})
	})

	Describe("InternalImports", func() {
		It("should return sorted internal imports", func() {
			mgr.AddInternal("user", "core/pkg/user")
			mgr.AddInternal("channel", "core/pkg/channel")
			result := mgr.InternalImports()
			Expect(result).To(HaveLen(2))
			Expect(result[0].Path).To(Equal("core/pkg/channel"))
			Expect(result[1].Path).To(Equal("core/pkg/user"))
		})

		It("should exclude imports already in the external list", func() {
			mgr.AddExternal("core/pkg/user")
			mgr.AddInternal("user", "core/pkg/user")
			Expect(mgr.InternalImports()).To(BeEmpty())
		})

		It("should preserve aliases", func() {
			mgr.AddInternal("pkgchannel", "core/pkg/channel")
			result := mgr.InternalImports()
			Expect(result).To(HaveLen(1))
			Expect(result[0].Alias).To(Equal("pkgchannel"))
		})
	})

	Describe("AddImport", func() {
		It("should route external category to external imports", func() {
			mgr.AddImport("external", "fmt", "")
			Expect(mgr.ExternalImports()).To(ContainElement("fmt"))
		})

		It("should route imports without alias to external imports", func() {
			mgr.AddImport("internal", "fmt", "")
			Expect(mgr.ExternalImports()).To(ContainElement("fmt"))
		})

		It("should route aliased internal imports to internal imports", func() {
			mgr.AddImport("internal", "core/pkg/user", "user")
			result := mgr.InternalImports()
			Expect(result).To(HaveLen(1))
			Expect(result[0].Alias).To(Equal("user"))
		})
	})

	Describe("NeedsAlias", func() {
		It("should return true when alias differs from base path", func() {
			data := imports.InternalImportData{
				Path:  "core/pkg/channel",
				Alias: "pkgchannel",
			}
			Expect(data.NeedsAlias()).To(BeTrue())
		})

		It("should return false when alias matches base path", func() {
			data := imports.InternalImportData{
				Path:  "core/pkg/channel",
				Alias: "channel",
			}
			Expect(data.NeedsAlias()).To(BeFalse())
		})

		It("should return false when alias is empty", func() {
			data := imports.InternalImportData{
				Path:  "core/pkg/channel",
				Alias: "",
			}
			Expect(data.NeedsAlias()).To(BeFalse())
		})
	})
})
