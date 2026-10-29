// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exports_test

import (
	"go/token"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/exports"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type finding struct {
	File      string
	Kind      string
	Name      string
	TestsOnly bool
}

func summarize(findings []exports.Finding) []finding {
	out := make([]finding, len(findings))
	for i, f := range findings {
		out[i] = finding{
			File:      filepath.Base(f.Position.Filename),
			Kind:      f.Kind,
			Name:      f.Name,
			TestsOnly: f.TestsOnly,
		}
	}
	return out
}

var modules = []string{"testdata/a", "testdata/b"}

// setTempDir points os.TempDir at dir: TMPDIR on Unix, TMP and TEMP on Windows.
func setTempDir(dir string) {
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		GinkgoT().Setenv(key, dir)
	}
}

var _ = Describe("Check", func() {
	It(
		"Should report exported identifiers no other package uses",
		func(ctx SpecContext) {
			findings := MustSucceed(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux", "windows"},
			}))
			Expect(summarize(findings)).To(Equal([]finding{
				{File: "lib.go", Kind: "func", Name: "Internal"},
				{File: "lib.go", Kind: "func", Name: "Tested", TestsOnly: true},
				{File: "lib.go", Kind: "func", Name: "OtherTests", TestsOnly: true},
				{File: "lib.go", Kind: "var", Name: "Dead"},
				{File: "lib.go", Kind: "method", Name: "Side"},
				{File: "lib.go", Kind: "type", Name: "Handler"},
				{File: "lib.go", Kind: "type", Name: "Box"},
				{File: "lib.go", Kind: "method", Name: "Read"},
				{File: "windows.go", Kind: "func", Name: "WindowsOnly"},
				{File: "user.go", Kind: "func", Name: "Run"},
			}))
		},
	)

	It(
		"Should judge an identifier only on the GOOS values given",
		func(ctx SpecContext) {
			findings := summarize(MustSucceed(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux"},
			})))
			Expect(findings).To(ContainElement(
				finding{File: "shared.go", Kind: "func", Name: "UsedOnWindows"},
			))
			Expect(findings).ToNot(ContainElement(HaveField("Name", "WindowsOnly")))
		},
	)

	It(
		"Should return an error when it cannot create the workspace",
		func(ctx SpecContext) {
			notDir := filepath.Join(GinkgoT().TempDir(), "file")
			Expect(os.WriteFile(notDir, nil, 0o644)).To(Succeed())
			setTempDir(notDir)
			Expect(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux"},
			})).Error().To(MatchError(ContainSubstring("exports: create workspace directory")))
		},
	)

	It(
		"Should remove its workspace when a package fails to load",
		func(ctx SpecContext) {
			tmp := GinkgoT().TempDir()
			setTempDir(tmp)
			Expect(exports.Check(ctx, exports.Config{
				Modules: []string{"testdata/missing"},
				GOOS:    []string{"linux"},
			})).Error().To(MatchError(ContainSubstring("exports: load packages")))
			Expect(os.ReadDir(tmp)).To(BeEmpty())
		},
	)

	It("Should return an error when a package fails to load", func(ctx SpecContext) {
		Expect(exports.Check(ctx, exports.Config{
			Modules: []string{"testdata/missing"},
			GOOS:    []string{"linux"},
		})).Error().To(MatchError(ContainSubstring("exports: load packages")))
	})

	It("Should load modules under a path with a space", func(ctx SpecContext) {
		dir := filepath.Join(GinkgoT().TempDir(), "with space")
		for _, m := range modules {
			Expect(os.CopyFS(filepath.Join(dir, m), os.DirFS(m))).To(Succeed())
		}
		findings := MustSucceed(exports.Check(ctx, exports.Config{
			Modules: []string{
				filepath.Join(dir, "testdata/a"),
				filepath.Join(dir, "testdata/b"),
			},
			GOOS: []string{"linux"},
		}))
		Expect(summarize(findings)).To(ContainElement(
			finding{File: "lib.go", Kind: "func", Name: "Internal"},
		))
	})

	DescribeTable("Should return a validation error for an incomplete config",
		func(ctx SpecContext, cfg exports.Config, field string) {
			Expect(exports.Check(ctx, cfg)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
		},
		Entry("no modules", exports.Config{GOOS: []string{"linux"}}, "modules"),
		Entry("no GOOS", exports.Config{Modules: modules}, "goos"),
	)
})

var _ = Describe("Finding", func() {
	Describe("String", func() {
		It("Should say when only test code uses it", func() {
			f := exports.Finding{
				Position:  token.Position{Filename: "a.go", Line: 3},
				Kind:      "func",
				Name:      "Do",
				TestsOnly: true,
			}
			Expect(f.String()).To(Equal(
				"a.go:3: func Do is exported but only test code uses it",
			))
		})

		It("Should say when no other package uses it", func() {
			f := exports.Finding{
				Position: token.Position{Filename: "a.go", Line: 3},
				Kind:     "var",
				Name:     "X",
			}
			Expect(f.String()).To(Equal(
				"a.go:3: var X is exported but no other package uses it",
			))
		})
	})
})
