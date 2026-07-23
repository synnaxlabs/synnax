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
		paths := make([]string, 0, len(resp.Files))
		for _, f := range resp.Files {
			paths = append(paths, f.Path)
		}
		Expect(paths).To(ContainElement("out/transient.gen.go"))
	})
})
