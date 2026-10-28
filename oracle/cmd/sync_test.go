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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/cmd"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("sync command", func() {
	var (
		repoDir string
		cleanup func()
	)

	AfterEach(func() { cleanup() })

	Describe("with a valid schema", func() {
		BeforeEach(func() {
			// Intentionally unformatted so sync rewrites the schema source.
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/user.oracle": "@go output  \"x/go/user\"\n" +
					"User struct {key uuid\nname   string}",
			})
		})

		It("writes schema sources and generated outputs before buf fails", func() {
			// The mini repo has no buf configuration, so the run stops at the
			// buf-generate step; everything before it must have completed.
			root := cmd.New()
			Expect(executeCommand(root, "sync", "-v")).Error().
				To(MatchError(ContainSubstring("buf generate")))

			schema := string(MustSucceed(os.ReadFile(
				filepath.Join(repoDir, "schemas", "synnax", "user.oracle"),
			)))
			Expect(schema).To(ContainSubstring("@go output \"x/go/user\""))
			Expect(filepath.Join(repoDir, "x", "go", "user", "types.gen.go")).
				To(BeAnExistingFile())
		})
	})

	Describe("with Go and protobuf outputs", func() {
		var generated []string

		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/user.oracle": "@go output \"core/pkg/user\"\n@pb\n\n" +
					"User struct {\n    key uuid\n    name string\n}\n",
			})
			dir := filepath.Join(repoDir, "core", "pkg", "user")
			generated = []string{
				filepath.Join(dir, "types.gen.go"),
				filepath.Join(dir, "pb", "user.proto"),
				filepath.Join(dir, "pb", "translator.gen.go"),
			}
		})

		// The mini repo has no buf configuration, so every sync stops at buf generate
		// after it writes the generated files.
		sync := func() {
			GinkgoHelper()
			Expect(executeCommand(cmd.New(), "sync")).Error().
				To(MatchError(ContainSubstring("buf generate")))
		}

		It("Should generate the Go types, the protobuf definition, and its translator",
			func() {
				sync()
				for _, f := range generated {
					Expect(f).To(BeAnExistingFile())
				}
			},
		)

		It("Should restore a generated file that was edited by hand", func() {
			sync()
			want := MustSucceed(os.ReadFile(generated[0]))
			Expect(os.WriteFile(generated[0], []byte("edited"), 0o644)).To(Succeed())
			sync()
			Expect(os.ReadFile(generated[0])).To(Equal(want))
		})

		DescribeTable(
			"Should not rewrite generated files that are already current",
			func(cached bool) {
				sync()
				if !cached {
					Expect(
						os.Remove(filepath.Join(repoDir, ".oracle", "sync-cache.json")),
					).To(Succeed())
				}
				past := time.Now().Add(-time.Hour).Truncate(time.Second)
				for _, f := range generated {
					Expect(os.Chtimes(f, past, past)).To(Succeed())
				}
				sync()
				for _, f := range generated {
					Expect(
						MustSucceed(os.Stat(f)).ModTime(),
					).To(BeTemporally("==", past))
				}
			},
			Entry("with the sync cache", true),
			Entry("without the sync cache", false),
		)
	})

	Describe("with an invalid schema", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"bad.oracle": "Thing struct { name string }\n" +
					"Thing struct { other string }\n",
			})
		})

		It("prints diagnostics and fails generation", func() {
			root := cmd.New()
			Expect(executeCommand(root, "sync")).Error().
				To(MatchError(ContainSubstring("generation failed")))
		})
	})

	Describe("with no schemas", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{})
		})

		It("errors when no schema files exist", func() {
			root := cmd.New()
			Expect(executeCommand(root, "sync")).Error().
				To(MatchError(ContainSubstring("no schema files found")))
		})
	})
})
