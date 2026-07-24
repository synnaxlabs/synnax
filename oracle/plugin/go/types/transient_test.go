// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transient types", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		goPlugin = types.New(types.DefaultOptions())
	})

	It("Should generate unversioned typedefs at the package root", func(ctx SpecContext) {
		source := `
			@go output "out"
			Key uint32 {
				@ts type string
			}
			LocalKey uint20 {
			}
			Entry struct {
				@go version 0
				key   string   {@key}
				local LocalKey
				@go marshal
			}
			APIEntry struct {
				@go output "api/out"
				key Key
				local LocalKey
			}
		`
		req := MustGenerateRequest(ctx, source, "test", loader)
		resp := MustSucceed(goPlugin.Generate(req))
		var root string
		count := 0
		for _, f := range resp.Files {
			if f.Path == "out/types.gen.go" {
				root = string(f.Content)
				count++
			}
		}
		Expect(count).To(Equal(1))
		Expect(root).To(ContainSubstring("type Key uint32"))
		Expect(root).To(ContainSubstring("type LocalKey types.Uint20"))
		Expect(root).To(ContainSubstring("type Entry = versions.Entry"))
		Expect(strings.Count(root, "package out")).To(Equal(1))
		Expect(strings.Count(root, "import")).To(Equal(1))
	})
})
