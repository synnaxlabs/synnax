// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OrphanGate", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "orphan"))
		DeferCleanup(func() { Expect(os.RemoveAll(repoRoot)).To(Succeed()) })
	})

	writeGen := func(rel, content string) {
		path := filepath.Join(repoRoot, rel)
		Expect(os.MkdirAll(filepath.Dir(path), 0755)).To(Succeed())
		Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	}

	resultWith := func(produced []string) *pipeline.Result {
		files := make([]plugin.File, len(produced))
		for i, p := range produced {
			files[i] = plugin.File{Path: p, Content: []byte("x")}
		}
		return &pipeline.Result{Outputs: map[string][]plugin.File{"stub": files}}
	}

	It("flags a sibling file in a managed directory that no schema produces", func(ctx SpecContext) {
		writeGen("out/types.gen.go", "x")
		writeGen("out/dead.gen.go", "x")
		gate := check.NewOrphanGate(nil)
		report := gate.Run(ctx, resultWith([]string{"out/types.gen.go"}),
			check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings).To(HaveLen(1))
		Expect(report.Findings[0].Path).To(Equal("out/dead.gen.go"))
	})

	It("does not descend into subdirectories of managed dirs", func(ctx SpecContext) {
		writeGen("out/types.gen.go", "x")
		writeGen("out/migrations/v0/types.gen.go", "x")
		gate := check.NewOrphanGate(nil)
		report := gate.Run(ctx, resultWith([]string{"out/types.gen.go"}),
			check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("respects ignore patterns", func(ctx SpecContext) {
		writeGen("out/types.gen.go", "x")
		writeGen("out/migrate_auto.gen.go", "x")
		gate := check.NewOrphanGate(nil).WithIgnores("migrate_auto.gen.*")
		report := gate.Run(ctx, resultWith([]string{"out/types.gen.go"}),
			check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusPass))
	})

	It("uses the cache as a secondary signal", func(ctx SpecContext) {
		// File is on disk, in cache, but not in current produced set.
		writeGen("out/types.gen.go", "x")
		writeGen("out/legacy.gen.go", "x")
		cache := format.LoadCache(repoRoot)
		cache.PutRaw("out/legacy.gen.go", "abc")
		gate := check.NewOrphanGate(cache)
		report := gate.Run(ctx, resultWith([]string{"out/types.gen.go"}),
			check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusFail))
		paths := []string{}
		for _, f := range report.Findings {
			paths = append(paths, f.Path)
		}
		Expect(paths).To(ContainElement("out/legacy.gen.go"))
	})

	It("does not flag cached entries whose file no longer exists", func(ctx SpecContext) {
		// Manual deletion of an output file leaves the cache entry behind.
		// That is for the cache gate to surface, not the orphan gate.
		writeGen("out/types.gen.go", "x")
		cache := format.LoadCache(repoRoot)
		cache.PutRaw("out/legacy.gen.go", "abc")
		gate := check.NewOrphanGate(cache)
		report := gate.Run(ctx, resultWith([]string{"out/types.gen.go"}),
			check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusPass))
	})
})
