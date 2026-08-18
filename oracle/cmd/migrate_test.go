// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

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
			repoDir, cleanup = setupMiniRepo("0.53.4", map[string]string{
				"synnax/versions/channel/v0.oracle": chainV0,
			})
		})

		It("scaffolds the next version file for a resource name", func() {
			cmd := NewRootCmd()
			MustSucceed(executeCommand(cmd, "migrate", "channel"))
			next := filepath.Join(
				repoDir, "schemas", "synnax", "versions", "channel", "v1.oracle",
			)
			content := string(MustSucceed(os.ReadFile(next)))
			Expect(content).NotTo(BeEmpty())
		})

		It("errors when no resources are named", func() {
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "migrate")).Error().
				To(MatchError(ContainSubstring("name the resources to bump")))
		})

		It("errors for an unknown resource", func() {
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "migrate", "ghost")).Error().
				To(MatchError(ContainSubstring(`no version chain matches "ghost"`)))
		})

		It("refuses to overwrite an existing target file", func() {
			// Naming the resource twice makes the second scaffold hit the file the
			// first one just wrote.
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "migrate", "channel", "channel")).Error().
				To(MatchError(ContainSubstring("already exists")))
		})
	})

	Describe("with chains in multiple domains", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo("0.53.4", map[string]string{
				"synnax/versions/channel/v0.oracle": chainV0,
				"x/versions/channel/v0.oracle":      chainV0,
			})
		})

		It("rejects an ambiguous resource name", func() {
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "migrate", "channel")).Error().
				To(MatchError(ContainSubstring(`"channel" is ambiguous`)))
		})

		It("accepts a live path to disambiguate", func() {
			cmd := NewRootCmd()
			MustSucceed(executeCommand(cmd, "migrate", "schemas/x/channel"))
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
			repoDir, cleanup = setupMiniRepo("0.53.4", map[string]string{
				"user.oracle": "User struct {\n    name string\n}\n",
			})
		})

		It("errors when no chains exist", func() {
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "migrate", "channel")).Error().
				To(MatchError(ContainSubstring("no version chains exist")))
		})
	})
})
