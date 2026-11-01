// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

const chainV0 = `Channel struct {
    name string {
        @doc value "names the channel."
    }

    @go marshal
}
`

var _ = Describe("migrate command", func() {
	var (
		repoDir string
		cleanup func()
	)

	AfterEach(func() { cleanup() })

	Describe("with a single version chain", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/versions/channel/v0.oracle": chainV0,
			})
		})

		It("scaffolds the next version file for a resource name", func() {
			succeed("migrate", "channel")
			next := filepath.Join(
				repoDir, "schemas", "synnax", "versions", "channel", "v1.oracle",
			)
			content := string(MustSucceed(os.ReadFile(next)))
			Expect(content).NotTo(BeEmpty())
		})

		It("errors when no resources are named", func() {
			Expect(
				fail(1, "migrate"),
			).To(printOnce("name the resources to bump"))
		})

		It("errors for an unknown resource", func() {
			Expect(
				fail(1, "migrate", "ghost"),
			).To(printOnce(`no version chain matches "ghost"`))
		})

		It("refuses to overwrite an existing target file", func() {
			// Naming the resource twice makes the second scaffold hit the file the
			// first one just wrote.
			Expect(
				fail(1, "migrate", "channel", "channel"),
			).To(printOnce("already exists"))
		})
	})

	Describe("with chains in multiple domains", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/versions/channel/v0.oracle": chainV0,
				"x/versions/channel/v0.oracle":      chainV0,
			})
		})

		It("rejects an ambiguous resource name", func() {
			Expect(
				fail(1, "migrate", "channel"),
			).To(printOnce(`"channel" is ambiguous`))
		})

		It("accepts a live path to disambiguate", func() {
			succeed("migrate", "schemas/x/channel")
			next := filepath.Join(
				repoDir, "schemas", "x", "versions", "channel", "v1.oracle",
			)
			Expect(next).To(BeAnExistingFile())
			synnaxNext := filepath.Join(
				repoDir, "schemas", "synnax", "versions", "channel", "v1.oracle",
			)
			Expect(synnaxNext).NotTo(BeAnExistingFile())
		})
	})

	Describe("without version chains", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"user.oracle": "User struct {\n    name string\n}\n",
			})
		})

		It("errors when no chains exist", func() {
			Expect(
				fail(1, "migrate", "channel"),
			).To(printOnce("no version chains exist"))
		})
	})
})
