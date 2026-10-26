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
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	. "github.com/synnaxlabs/x/testutil"
)

// diskFormatter reads the file from disk, as ESLint's TypeScript project service does.
type diskFormatter struct{}

func (diskFormatter) Format(
	_ context.Context,
	content []byte,
	path string,
) ([]byte, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return append(content, "// formatted\n"...), nil
}

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
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "sync", "-v")).Error().
				To(MatchError(ContainSubstring("buf generate")))

			schema := string(MustSucceed(os.ReadFile(
				filepath.Join(repoDir, "schemas", "synnax", "user.oracle"),
			)))
			Expect(schema).To(ContainSubstring("@go output \"x/go/user\""))
			Expect(filepath.Join(repoDir, "x", "go", "user", "types.gen.go")).
				To(BeAnExistingFile())
		})
	})

	Describe("with an invalid schema", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"bad.oracle": "Thing struct { name string }\n" +
					"Thing struct { other string }\n",
			})
		})

		It("prints diagnostics and fails generation", func() {
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "sync")).Error().
				To(MatchError(ContainSubstring("generation failed")))
		})
	})

	Describe("with no schemas", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{})
		})

		It("errors when no schema files exist", func() {
			cmd := NewRootCmd()
			Expect(executeCommand(cmd, "sync")).Error().
				To(MatchError(ContainSubstring("no schema files found")))
		})
	})
})

var _ = Describe("sync outputs", func() {
	It("Should put a new file on disk before formatting it", func(ctx SpecContext) {
		root := GinkgoT().TempDir()
		formatters := format.NewRegistry()
		formatters.Register(".ts", diskFormatter{})
		result := &pipeline.Result{Outputs: map[string][]plugin.File{
			"ts/types": {{Path: "src/thing/types.gen.ts", Content: []byte("x\n")}},
		}}
		r := MustSucceed(syncOutputs(
			ctx, result, root, formatters, format.LoadCache(root), 1,
		))
		Expect(r.Written).To(Equal([]string{"src/thing/types.gen.ts"}))
		Expect(os.ReadFile(filepath.Join(root, "src", "thing", "types.gen.ts"))).
			To(Equal([]byte("x\n// formatted\n")))
	})
})
