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
			Expect(fail(1, "sync", "-v")).To(ContainSubstring("buf generate"))

			schema := string(MustSucceed(os.ReadFile(
				filepath.Join(repoDir, "schemas", "synnax", "user.oracle"),
			)))
			Expect(schema).To(ContainSubstring("@go output \"x/go/user\""))
			Expect(filepath.Join(repoDir, "x", "go", "user", "types.gen.go")).
				To(BeAnExistingFile())
		})
	})

	Describe("with generated files on disk", func() {
		var generated string

		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/user.oracle": "@go output \"core/pkg/user\"\n\n" +
					"User struct {\n    key uuid\n    name string\n}\n",
			})
			generated = filepath.Join(repoDir, "core", "pkg", "user", "types.gen.go")
		})

		// The mini repo has no buf configuration, so every sync stops at buf generate
		// after it writes the generated files.
		sync := func() {
			GinkgoHelper()
			Expect(fail(1, "sync")).To(ContainSubstring("buf generate"))
		}

		It("Should restore a generated file that was edited by hand", func() {
			sync()
			want := MustSucceed(os.ReadFile(generated))
			Expect(os.WriteFile(generated, []byte("edited"), 0o644)).To(Succeed())
			sync()
			Expect(os.ReadFile(generated)).To(Equal(want))
		})

		DescribeTable(
			"Should not rewrite a generated file that is already current",
			func(cached bool) {
				sync()
				if !cached {
					Expect(
						os.Remove(filepath.Join(repoDir, ".oracle", "sync-cache.json")),
					).To(Succeed())
				}
				past := time.Now().Add(-time.Hour).Truncate(time.Second)
				Expect(os.Chtimes(generated, past, past)).To(Succeed())
				sync()
				Expect(
					MustSucceed(os.Stat(generated)).ModTime(),
				).To(BeTemporally("==", past))
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
			Expect(fail(1, "sync")).To(printOnce("generation failed"))
		})
	})

	Describe("with no schemas", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{})
		})

		It("errors when no schema files exist", func() {
			Expect(fail(1, "sync")).To(printOnce("no schema files found"))
		})
	})
})

var _ = Describe("sync with a new TypeScript output", func() {
	var (
		repoDir string
		cleanup func()
	)

	BeforeEach(func() {
		repoDir, cleanup = setupMiniRepo(map[string]string{
			"synnax/user.oracle": "@ts output \"client/ts/src/user\"\n\n" +
				"User struct {\n    key uuid\n    name string\n}\n",
		})
		// ESLint and Prettier run through pnpm, so a failing pnpm stops the sync while
		// it formats.
		bin := GinkgoT().TempDir()
		Expect(os.WriteFile(
			filepath.Join(bin, "pnpm"),
			[]byte("#!/bin/sh\nexit 1\n"),
			0o755,
		)).To(Succeed())
		GinkgoT().Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	})

	AfterEach(func() { cleanup() })

	It("Should put a new file on disk before formatting it", func() {
		fail(1, "sync")
		Expect(filepath.Join(repoDir, "client", "ts", "src", "user", "types.gen.ts")).
			To(BeAnExistingFile())
	})
})
