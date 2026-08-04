// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versioning_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func analyze(
	ctx context.Context,
	source string,
	loader *testutil.MockFileLoader,
) (*resolution.Table, error) {
	table, diag := analyzer.AnalyzeSource(ctx, source, "test", loader)
	if diag != nil && !diag.Ok() {
		return nil, diag
	}
	return table, nil
}

var _ = Describe("Versioning", func() {
	var loader *testutil.MockFileLoader

	BeforeEach(func() { loader = testutil.NewMockFileLoader() })

	Describe("Version", func() {
		It("Should return the declared @go version", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
				    @go version 3
					key uuid @key
					name string
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			v := MustBeOk(versioning.Version(table.MustGet("test.Entry")))
			Expect(v).To(Equal(3))
		})

		It(
			"Should return false when the file declares no version",
			func(ctx SpecContext) {
				source := `
				@go output "out"
				Entry struct {
					key uuid @key
				}
			`
				table := MustSucceed(analyze(ctx, source, loader))
				_, ok := versioning.Version(table.MustGet("test.Entry"))
				Expect(ok).To(BeFalse())
			},
		)

		It("Should return false when the type has no go domain", func(ctx SpecContext) {
			source := `
				Entry struct {
					key uuid @key
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			_, ok := versioning.Version(table.MustGet("test.Entry"))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Pinned", func() {
		It(
			"Should report the pinned marker on a version declaration",
			func(ctx SpecContext) {
				source := `
				@go output "out"
				Entry struct {
				    @go version 2 pinned
					key uuid @key
				}
			`
				table := MustSucceed(analyze(ctx, source, loader))
				Expect(versioning.Pinned(table.MustGet("test.Entry"))).To(BeTrue())
				v := MustBeOk(versioning.Version(table.MustGet("test.Entry")))
				Expect(v).To(Equal(2))
			},
		)

		It("Should report false without the marker", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
				    @go version 2
					key uuid @key
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			Expect(versioning.Pinned(table.MustGet("test.Entry"))).To(BeFalse())
		})
	})

	Describe("Dir", func() {
		It("Should format the version sub-directory name", func() {
			Expect(versioning.Dir(0)).To(Equal("v0"))
			Expect(versioning.Dir(12)).To(Equal("v12"))
		})
	})

	Describe("VersionedPath", func() {
		It("Should append the types/vN sub-path", func() {
			Expect(versioning.VersionedPath("core/out", 3)).To(
				Equal("core/out/versions/v3"))
		})
	})

	Describe("PathVersions", func() {
		It(
			"Should map each versioned output path to its version",
			func(ctx SpecContext) {
				loader.Add("schemas/dep.oracle", `
				@go output "dep"
				Item struct {
				    @go version 5
					key uuid @key
				}
			`)
				source := `
				import "schemas/dep"
				@go output "out"
				Entry struct {
				    @go version 3
					key uuid @key
					item dep.Item
				}
			`
				table := MustSucceed(analyze(ctx, source, loader))
				Expect(versioning.PathVersions(table)).To(Equal(
					map[string]int{"out": 3, "dep": 5}))
			},
		)

		It(
			"Should return an empty map when no versions are declared",
			func(ctx SpecContext) {
				source := `
				@go output "out"
				Entry struct {
					key uuid @key
				}
			`
				table := MustSucceed(analyze(ctx, source, loader))
				Expect(versioning.PathVersions(table)).To(BeEmpty())
			},
		)

		It("Should error on a negative version", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					@go version -1
					key uuid @key
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			Expect(versioning.PathVersions(table)).Error().To(MatchError(
				ContainSubstring("must be a non-negative integer")))
		})

		It("Should error on conflicting versions at one path", func(ctx SpecContext) {
			loader.Add("schemas/dep.oracle", `
				@go output "out"
				Item struct {
				    @go version 2
					key uuid @key
				}
			`)
			source := `
				import "schemas/dep"
				@go output "out"
				Entry struct {
				    @go version 1
					key uuid @key
					item dep.Item
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			Expect(versioning.PathVersions(table)).Error().To(MatchError(
				ContainSubstring("conflicting @go version declarations for out")))
		})

		It("Should error on @go migrate without a version", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					key uuid @key
					@go migrate
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			Expect(versioning.PathVersions(table)).Error().To(MatchError(
				ContainSubstring("@go migrate requires a @go version declaration")))
		})
	})

	Describe("EntryPaths", func() {
		It(
			"Should include versioned paths containing a keyed struct",
			func(ctx SpecContext) {
				source := `
				@go output "out"
				Entry struct {
				    @go version 3
					key uuid @key
					name string
					@go marshal
				}
			`
				table := MustSucceed(analyze(ctx, source, loader))
				Expect(versioning.EntryPaths(table)).To(Equal(map[string]int{"out": 3}))
			},
		)

		It(
			"Should include versioned paths with no keyed struct",
			func(ctx SpecContext) {
				source := `
				@go output "out"
				Value struct {
				    @go version 3
					name string
				}
			`
				table := MustSucceed(analyze(ctx, source, loader))
				Expect(versioning.EntryPaths(table)).To(Equal(map[string]int{"out": 3}))
			},
		)

		It("Should exclude keyed structs with no version", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					key uuid @key
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			Expect(versioning.EntryPaths(table)).To(BeEmpty())
		})
	})

	Describe("RewriteCurrent", func() {
		It("Should rewrite only version-laid-out paths", func(ctx SpecContext) {
			loader.Add("schemas/dep.oracle", `
				@go output "dep"
				Item struct {
					name string
				}
			`)
			source := `
				import "schemas/dep"
				@go output "out"
				Entry struct {
				    @go version 3
					key uuid @key
					item dep.Item
					@go marshal
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			rewritten, pathMap := MustSucceed2(versioning.RewriteCurrent(table))
			Expect(pathMap).To(Equal(map[string]string{"out": "out/versions/v3"}))
			entry := rewritten.MustGet("test.Entry")
			Expect(output.GetPath(entry, "go")).To(Equal("out/versions/v3"))
			item := rewritten.MustGet("dep.Item")
			Expect(output.GetPath(item, "go")).To(Equal("dep"))
			Expect(output.GetPath(table.MustGet("test.Entry"), "go")).To(Equal("out"))
		})

		It("Should return the table unchanged with no versions", func(ctx SpecContext) {
			source := `
				@go output "out"
				Entry struct {
					key uuid @key
				}
			`
			table := MustSucceed(analyze(ctx, source, loader))
			rewritten, pathMap := MustSucceed2(versioning.RewriteCurrent(table))
			Expect(rewritten).To(BeIdenticalTo(table))
			Expect(pathMap).To(BeEmpty())
		})
	})
})

var _ = Describe("AliasSplit", func() {
	var loader *testutil.MockFileLoader

	BeforeEach(func() { loader = testutil.NewMockFileLoader() })

	analyzeTable := func(ctx context.Context, source string) *resolution.Table {
		GinkgoHelper()
		table := MustSucceed(analyze(ctx, source, testutil.NewMockFileLoader()))
		return table
	}

	It(
		"Should use the latest snapshot declaring the predecessor version",
		func(ctx SpecContext) {
			liveTable := MustSucceed(analyze(ctx, `
			@go output "out"
			Stable struct {
			    @go version 2
			    name string
			}
			Grown  struct {
			    @go version 2
			    value int32  extra string
			}
		`, loader))
			snapshots := map[int]string{
				// v56 already declares the current version and cannot anchor.
				56: `
				@go output "out"
				Stable struct {
				    @go version 2
				    name string
				}
				Grown  struct {
				    @go version 2
				    value int32  extra string
				}
			`,
				55: `
				@go output "out"
				Stable struct {
				    @go version 1
				    name string
				}
				Grown  struct {
				    @go version 1
				    value int32
				}
			`,
			}
			split := MustSucceed(versioning.AliasSplit(
				liveTable, 56,
				func(version int) (*resolution.Table, error) {
					src, ok := snapshots[version]
					if !ok {
						return nil, nil
					}
					return analyzeTable(ctx, src), nil
				},
			))
			Expect(split).To(HaveKey("out"))
			Expect(split["out"].PredecessorVersion).To(Equal(1))
			Expect(split["out"].Aliased).To(HaveLen(1))
			for qn := range split["out"].Aliased {
				Expect(qn).To(HaveSuffix("Stable"))
			}
		},
	)

	It(
		"Should return nothing when no snapshot declares the predecessor",
		func(ctx SpecContext) {
			liveTable := MustSucceed(analyze(ctx, `
			@go output "out"
			Stable struct {
			    @go version 2
			    name string
			}
		`, loader))
			split := MustSucceed(versioning.AliasSplit(
				liveTable, 56,
				func(int) (*resolution.Table, error) { return nil, nil },
			))
			Expect(split).To(BeEmpty())
		},
	)

	It("Should return nothing for paths at version zero", func(ctx SpecContext) {
		liveTable := MustSucceed(analyze(ctx, `
			@go output "out"
			Stable struct {
			    @go version 0
			    name string
			}
		`, loader))
		split := MustSucceed(versioning.AliasSplit(
			liveTable, 56,
			func(int) (*resolution.Table, error) {
				return analyzeTable(ctx, `
					@go output "out"
					Stable struct {
					    @go version 0
					    name string
					}
				`), nil
			},
		))
		Expect(split).To(BeEmpty())
	})
})
