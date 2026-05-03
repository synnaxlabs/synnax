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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
)

func TestImports(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TS Imports Suite")
}

var _ = Describe("Manager", func() {
	var m *imports.Manager

	BeforeEach(func() { m = imports.NewManager() })

	It("Should deduplicate repeated names from the same path", func() {
		m.AddImport("@synnaxlabs/x", "color")
		m.AddImport("@synnaxlabs/x", "color")
		m.AddImport("@synnaxlabs/x", "xy")
		s := m.SynnaxImports()
		Expect(s).To(HaveLen(1))
		Expect(s[0].Names).To(Equal([]string{"color", "xy"}))
	})

	It("Should sort names within an import alphabetically", func() {
		m.AddImport("@synnaxlabs/x", "z")
		m.AddImport("@synnaxlabs/x", "a")
		m.AddImport("@synnaxlabs/x", "m")
		s := m.SynnaxImports()
		Expect(s[0].Names).To(Equal([]string{"a", "m", "z"}))
	})

	It("Should sort imports by path", func() {
		m.AddImport("@synnaxlabs/x", "x")
		m.AddImport("@synnaxlabs/pluto", "p")
		m.AddImport("@synnaxlabs/client", "c")
		paths := []string{}
		for _, n := range m.SynnaxImports() {
			paths = append(paths, n.Path)
		}
		Expect(paths).To(Equal([]string{
			"@synnaxlabs/client",
			"@synnaxlabs/pluto",
			"@synnaxlabs/x",
		}))
	})

	It("Should partition Synnax, external, and internal imports by prefix", func() {
		m.AddImport("@synnaxlabs/x", "x")
		m.AddImport("zod", "z")
		m.AddImport("immer", "produce")
		m.AddImport("@/schematic/types.gen", "Schematic")
		m.AddImport("@/util/keys", "keyZ")

		Expect(m.SynnaxImports()).To(HaveLen(1))
		Expect(m.SynnaxImports()[0].Path).To(Equal("@synnaxlabs/x"))

		ext := m.ExternalNamedImports()
		Expect(ext).To(HaveLen(2))
		Expect(ext[0].Path).To(Equal("immer"))
		Expect(ext[1].Path).To(Equal("zod"))

		internal := m.InternalNamedImports()
		Expect(internal).To(HaveLen(2))
		Expect(internal[0].Path).To(Equal("@/schematic/types.gen"))
		Expect(internal[1].Path).To(Equal("@/util/keys"))
	})

	It("Should return nil when no imports were added", func() {
		Expect(m.SynnaxImports()).To(BeNil())
		Expect(m.ExternalNamedImports()).To(BeNil())
		Expect(m.InternalNamedImports()).To(BeNil())
	})
})
