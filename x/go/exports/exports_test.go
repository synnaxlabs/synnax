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
	"strings"

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
				{File: "lib.go", Kind: "method", Name: "Handle"},
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
		"Should check only the modules that the workspace does not add",
		func(ctx SpecContext) {
			findings := MustSucceed(exports.Check(ctx, exports.Config{
				Modules:   []string{"testdata/b"},
				Workspace: []string{"testdata/a", "testdata/b"},
				GOOS:      []string{"linux"},
			}))
			Expect(summarize(findings)).To(Equal([]finding{
				{File: "user.go", Kind: "func", Name: "Run"},
			}))
		},
	)

	It("Should name the package and receiver of a method", func(ctx SpecContext) {
		findings := MustSucceed(exports.Check(ctx, exports.Config{
			Modules: modules,
			GOOS:    []string{"linux"},
		}))
		Expect(findings).To(ContainElement(SatisfyAll(
			HaveField("Package", "example.com/a/lib"),
			HaveField("Receiver", "Box"),
			HaveField("Name", "Read"),
		)))
	})

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

type edit struct{ file, old, new string }

// copyModules copies the test modules into a new directory, applies the edits, and
// returns the directory.
func copyModules(edits ...edit) string {
	GinkgoHelper()
	dir := GinkgoT().TempDir()
	for _, m := range modules {
		Expect(os.CopyFS(filepath.Join(dir, m), os.DirFS(m))).To(Succeed())
	}
	for _, e := range edits {
		path := filepath.Join(dir, e.file)
		src := string(MustSucceed(os.ReadFile(path)))
		Expect(src).To(ContainSubstring(e.old))
		src = strings.Replace(src, e.old, e.new, 1)
		Expect(os.WriteFile(path, []byte(src), 0o644)).To(Succeed())
	}
	return dir
}

const (
	libGo     = "testdata/a/lib/lib.go"
	libTestGo = "testdata/a/lib/lib_test.go"
	userGo    = "testdata/b/user/user.go"
)

var _ = Describe("Check with a base", func() {
	DescribeTable("Should return only the findings that the base lacks",
		func(ctx SpecContext, edits []edit, expected []finding) {
			findings := MustSucceed(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux"},
				Base:    copyModules(edits...),
			}))
			Expect(summarize(findings)).To(Equal(expected))
		},
		Entry(
			"an unchanged backlog at other lines",
			[]edit{{libGo, "package lib\n", "package lib\n\n\n"}},
			[]finding{},
		),
		Entry(
			"a new function that only test code uses",
			[]edit{
				{libGo, "func Tested() {}\n", ""},
				{libTestGo, "lib.Tested", "lib.Used"},
			},
			[]finding{{File: "lib.go", Kind: "func", Name: "Tested", TestsOnly: true}},
		),
		Entry(
			"a method of the same name on another type",
			[]edit{
				{libGo, "func (Square) Side()", "func (Op) Side()"},
				{libGo, "Square{}.Side()", "Op{}.Side()"},
			},
			[]finding{{File: "lib.go", Kind: "method", Name: "Side"}},
		),
		Entry(
			"an identifier of the same name and another kind",
			[]edit{{libGo, "var Dead = 1", "const Dead = 1"}},
			[]finding{{File: "lib.go", Kind: "var", Name: "Dead"}},
		),
		Entry(
			"an identifier of the same name in another package",
			[]edit{
				{libGo, "func Internal() {}\n", ""},
				{libGo, "{ Internal() }", "{}"},
				{userGo, "func Run()", "func Internal() {}\n\nfunc Run()"},
			},
			[]finding{{File: "lib.go", Kind: "func", Name: "Internal"}},
		),
		Entry(
			"a finding that only test code now uses",
			[]edit{{libTestGo, "lib.Tested", "lib.Used"}},
			[]finding{},
		),
	)

	It(
		"Should return every finding of a module that the base lacks",
		func(ctx SpecContext) {
			base := GinkgoT().TempDir()
			Expect(os.CopyFS(
				filepath.Join(base, "testdata/a"),
				os.DirFS("testdata/a"),
			)).To(Succeed())
			findings := MustSucceed(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux"},
				Base:    base,
			}))
			Expect(summarize(findings)).To(Equal([]finding{
				{File: "user.go", Kind: "func", Name: "Run"},
			}))
		},
	)

	It(
		"Should return every finding when the base holds no module",
		func(ctx SpecContext) {
			findings := MustSucceed(exports.Check(ctx, exports.Config{
				Modules: []string{"testdata/a"},
				GOOS:    []string{"linux"},
				Base:    GinkgoT().TempDir(),
			}))
			Expect(findings).To(ContainElement(HaveField("Name", "Internal")))
		},
	)

	It(
		"Should return every finding of a gated module and usage",
		func(ctx SpecContext) {
			findings := MustSucceed(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux"},
				Base:    copyModules(),
				Gated: []exports.Gate{
					{Module: "testdata/a", Usage: "tests-only"},
					{Module: "testdata/b", Usage: "unused"},
				},
			}))
			Expect(summarize(findings)).To(Equal([]finding{
				{File: "lib.go", Kind: "func", Name: "Tested", TestsOnly: true},
				{File: "lib.go", Kind: "func", Name: "OtherTests", TestsOnly: true},
				{File: "user.go", Kind: "func", Name: "Run"},
			}))
			Expect(findings).To(HaveEach(HaveField("Module", BeElementOf(modules))))
		},
	)

	It(
		"Should return a validation error for an unknown gate usage",
		func(ctx SpecContext) {
			Expect(exports.Check(ctx, exports.Config{
				Modules: modules,
				GOOS:    []string{"linux"},
				Base:    GinkgoT().TempDir(),
				Gated:   []exports.Gate{{Module: "testdata/a", Usage: "dead"}},
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`gate usage "dead"`)),
			))
		},
	)

	It(
		"Should return a validation error for an absolute module",
		func(ctx SpecContext) {
			Expect(exports.Check(ctx, exports.Config{
				Modules: []string{MustSucceed(filepath.Abs("testdata/a"))},
				GOOS:    []string{"linux"},
				Base:    GinkgoT().TempDir(),
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("must be relative")),
			))
		},
	)
})

var _ = Describe("Finding", func() {
	Describe("String", func() {
		It("Should name the receiver of a method", func() {
			f := exports.Finding{
				Position: token.Position{Filename: "a.go", Line: 3},
				Kind:     "method",
				Receiver: "Box",
				Name:     "Read",
			}
			Expect(f.String()).To(Equal(
				"a.go:3: method Box.Read is exported but no other package uses it",
			))
		})

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
