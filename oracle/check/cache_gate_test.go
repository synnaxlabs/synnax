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

var _ = Describe("CacheGate", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "cache"))
		DeferCleanup(func() { Expect(os.RemoveAll(repoRoot)).To(Succeed()) })
	})

	resultWith := func(content []byte) *pipeline.Result {
		return &pipeline.Result{
			Outputs: map[string][]plugin.File{
				"stub": {{Path: "out/x.gen.go", Content: content}},
			},
		}
	}

	It("skips when cache is nil", func(ctx SpecContext) {
		gate := check.NewCacheGate(nil)
		Expect(gate.Run(ctx, resultWith([]byte("hi")), check.Env{}).Status).
			To(Equal(check.StatusSkipped))
	})

	It("flags a cache hit where the on-disk file is missing", func(ctx SpecContext) {
		cache := format.LoadCache(repoRoot)
		content := []byte("hi")
		cache.Put("out/x.gen.go", format.Entry{
			Raw:       format.Hash(content),
			Canonical: format.Hash([]byte("hi (canonical)")),
		})
		gate := check.NewCacheGate(cache)
		report := gate.Run(ctx, resultWith(content), check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).To(ContainSubstring("does not exist"))
	})

	It("flags a cache hit whose canonical hash diverges from disk", func(ctx SpecContext) {
		// The user's reported failure mode: cache says "fresh" but the
		// on-disk bytes hash to something else (formatter version bump
		// or hand edit). Sync would skip; this gate must catch it.
		Expect(os.MkdirAll(filepath.Join(repoRoot, "out"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(repoRoot, "out", "x.gen.go"), []byte("stale"), 0644)).To(Succeed())
		cache := format.LoadCache(repoRoot)
		content := []byte("hi")
		cache.Put("out/x.gen.go", format.Entry{
			Raw:       format.Hash(content),
			Canonical: format.Hash([]byte("hi (canonical)")),
		})
		gate := check.NewCacheGate(cache)
		report := gate.Run(ctx, resultWith(content), check.Env{RepoRoot: repoRoot})
		Expect(report.Status).To(Equal(check.StatusFail))
		Expect(report.Findings[0].Message).To(ContainSubstring("cached canonical hash"))
	})

	It("passes when cache and disk are consistent", func(ctx SpecContext) {
		Expect(os.MkdirAll(filepath.Join(repoRoot, "out"), 0755)).To(Succeed())
		canonical := []byte("hi (canonical)")
		Expect(os.WriteFile(filepath.Join(repoRoot, "out", "x.gen.go"), canonical, 0644)).To(Succeed())
		cache := format.LoadCache(repoRoot)
		content := []byte("hi")
		cache.Put("out/x.gen.go", format.Entry{
			Raw:       format.Hash(content),
			Canonical: format.Hash(canonical),
		})
		gate := check.NewCacheGate(cache)
		Expect(gate.Run(ctx, resultWith(content), check.Env{RepoRoot: repoRoot}).Status).
			To(Equal(check.StatusPass))
	})

	It("ignores cache miss (different raw hash)", func(ctx SpecContext) {
		// On a raw mismatch, sync would reformat - this is just a cache
		// miss, not poisoning. The gate must not fail.
		cache := format.LoadCache(repoRoot)
		cache.Put("out/x.gen.go", format.Entry{Raw: "stale-hash", Canonical: "stale-canonical"})
		gate := check.NewCacheGate(cache)
		Expect(gate.Run(ctx, resultWith([]byte("hi")), check.Env{RepoRoot: repoRoot}).Status).
			To(Equal(check.StatusPass))
	})
})
