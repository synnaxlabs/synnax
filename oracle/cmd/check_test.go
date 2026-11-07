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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/check"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("oracle check end-to-end", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "check-e2e"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		// Bare git repo so paths.RepoRoot resolves.
		Expect(os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(repoRoot, "schemas"), 0o755)).To(Succeed())
	})

	writeSchema := func(name, body string) {
		GinkgoHelper()
		path := filepath.Join(repoRoot, "schemas", name+".oracle")
		Expect(os.WriteFile(path, []byte(body), 0o644)).To(Succeed())
	}

	runOracleCheck := func(args ...string) (string, int) {
		GinkgoHelper()
		bin := buildOracleBinary()
		proc := exec.Command(bin, append([]string{"check"}, args...)...)
		proc.Dir = repoRoot
		out, _ := proc.CombinedOutput()
		return string(out), proc.ProcessState.ExitCode()
	}

	It("returns exit 11 (analyzer errors) for an invalid schema", func() {
		// Two struct definitions with the same name in the same file.
		// The analyzer rejects this as a duplicate-definition error,
		// not a warning, so the gate fails with the analyze exit code.
		writeSchema("bad", `
@go output "x/go/bad"
Thing struct { name string }
Thing struct { other string }
`)
		_, code := runOracleCheck("--gates=analyze")
		Expect(code).To(Equal(check.FailureCodes["analyze"]))
	})

	It("escalates analyzer warnings to errors with --warnings-as-errors", func() {
		// Unresolved type is a warning by default; --warnings-as-errors
		// must promote it to a failing error.
		writeSchema("warn", `
@go output "x/go/warn"
Thing struct {
    other Missing
}
`)
		_, code := runOracleCheck("--gates=analyze", "--warnings-as-errors")
		Expect(code).To(Equal(check.FailureCodes["analyze"]))
	})

	It("exits 10 when a schema is not canonically formatted", func() {
		// A trivial schema; if formatter has any opinion, the bytes
		// won't match.
		writeSchema("widget", `
@go output  "x/go/widget"
Thing struct { name string }
`)
		_, code := runOracleCheck("--gates=format")
		Expect(code).To(Equal(check.FailureCodes["format"]))
	})

	It("emits valid JSON when --format=json", func() {
		writeSchema("good", `
@go output "x/go/good"
Thing struct {
    name string
}
`)
		out, _ := runOracleCheck("--gates=format,analyze", "--format=json")
		Expect(out).To(ContainSubstring(`"gates":`))
		Expect(out).To(ContainSubstring(`"status":`))
	})
})

var _ = Describe("check command flag paths", func() {
	var cleanup func()

	AfterEach(func() { cleanup() })

	Describe("with a well-formed schema", func() {
		BeforeEach(func() {
			_, cleanup = setupMiniRepo(map[string]string{
				"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
			})
		})

		It("rejects an invalid --format value", func() {
			Expect(
				fail(1, "check", "--format=yaml"),
			).To(printOnce(`invalid --format "yaml"`))
		})

		It("runs only the requested gates", func() {
			succeed("check", "--gates=format,analyze")
		})

		It("renders a JSON report without error", func() {
			succeed("check", "--gates=format,analyze", "--format=json")
		})

		It("renders a verbose report without error", func() {
			succeed("check", "-v")
		})
	})

	Describe("with format drift", func() {
		BeforeEach(func() {
			_, cleanup = setupMiniRepo(map[string]string{
				"user.oracle": "User struct {key uuid\nname   string}",
			})
		})

		It("returns the gate-failure exit error", func() {
			Expect(
				failGate(check.FailureCodes["format"], "--gates=format"),
			).To(ContainSubstring("1 gate(s) failed"))
		})

		It("includes diffs without error when --diff is set", func() {
			Expect(
				failGate(check.FailureCodes["format"], "--gates=format", "--diff"),
			).To(ContainSubstring("1 gate(s) failed"))
		})
	})

	Describe("with an analyzer warning", func() {
		BeforeEach(func() {
			_, cleanup = setupMiniRepo(map[string]string{
				"warn.oracle": "@go output \"x/go/warn\"\n" +
					"Thing struct {\n    other Missing\n}\n",
			})
		})

		It("escalates warnings with --warnings-as-errors", func() {
			succeed("check", "--gates=analyze")

			Expect(failGate(
				check.FailureCodes["analyze"],
				"--gates=analyze", "--warnings-as-errors",
			)).To(ContainSubstring("1 gate(s) failed"))
		})
	})
})

var _ = Describe("frozen gate", func() {
	const (
		liveHead = "@go output \"core/pkg/service/doc\"\n\n"
		base     = "Base struct {\n    name string\n\n    @go marshal\n}\n\n"
		v0Source = base + "Doc struct {\n    key uuid @key\n    base Base\n\n" +
			"    @go marshal\n}\n"
		v1Source = "Base = v0.Base\n\nDoc struct {\n    key uuid @key\n" +
			"    base Base\n    title string\n\n    @go marshal\n}\n"
		liveDoc = "Doc struct {\n    key uuid @key\n    base Base\n" +
			"    title string\n\n    @go marshal\n}\n"
		ruledBase = "Base struct {\n    name string {\n        @validate required\n" +
			"    }\n\n    @go marshal\n}\n\n"
		frozenPackage = "core/pkg/service/doc/versions/v0"
	)
	var (
		repoDir string
		cleanup func()
	)
	AfterEach(func() { cleanup() })
	write := func(rel, content string) {
		GinkgoHelper()
		Expect(os.WriteFile(
			filepath.Join(repoDir, rel), []byte(content), 0o644,
		)).To(Succeed())
	}
	// The mini repo has no buf configuration, so sync exits 1 after the sums are
	// written.
	syncRepo := func(args ...string) {
		GinkgoHelper()
		Expect(fail(1, append([]string{"sync"}, args...)...)).
			To(ContainSubstring("buf generate"))
	}
	gateCode := check.FailureCodes["frozen"]

	Describe("with a frozen version", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/versions/doc/v0.oracle": v0Source,
				"synnax/versions/doc/v1.oracle": v1Source,
				"synnax/doc.oracle":             liveHead + base + liveDoc,
			})
			syncRepo()
		})

		It("passes after a sync", func() {
			succeed("check", "--gates=frozen")
		})

		It("reports a malformed sum file in the gate report", func() {
			write("schemas/synnax/frozen/doc.sum", "<<<<<<< HEAD\n")
			succeed("check", "--gates=analyze")
			Expect(failGate(gateCode, "--gates=generated,frozen")).
				To(ContainSubstring("schemas/synnax/frozen/doc.sum"))
		})

		It("fails on a hand edit of a frozen file", func() {
			path := filepath.Join(repoDir, frozenPackage, "types.gen.go")
			content := string(MustSucceed(os.ReadFile(path)))
			Expect(os.WriteFile(path, []byte(content+"// edit\n"), 0o644)).To(Succeed())
			Expect(failGate(gateCode, "--gates=frozen")).
				To(ContainSubstring("frozen package changed"))
		})

		It("fails when a generated change reaches a frozen file", func() {
			write("schemas/synnax/doc.oracle", liveHead+ruledBase+liveDoc)
			syncRepo()
			Expect(failGate(gateCode, "--gates=frozen")).
				To(ContainSubstring(frozenPackage))
		})

		It("keeps failing on a later sync, and passes after a refreeze", func() {
			write("schemas/synnax/doc.oracle", liveHead+ruledBase+liveDoc)
			syncRepo()
			syncRepo()
			failGate(gateCode, "--gates=frozen")
			syncRepo("--refreeze")
			succeed("check", "--gates=frozen")
		})

		It("accepts a change to the version file by itself", func() {
			write(
				"schemas/synnax/versions/doc/v0.oracle",
				"// Edited.\n"+v0Source,
			)
			syncRepo()
			succeed("check", "--gates=frozen")
		})
	})

	Describe("with a mint on top of a current version", func() {
		const ruledDoc = "Doc struct {\n    key uuid @key\n    name string {\n" +
			"        @validate required\n    }\n\n    @go marshal\n}\n"
		const plainDoc = "Doc struct {\n    key uuid @key\n    name string\n\n" +
			"    @go marshal\n}\n"
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/versions/doc/v0.oracle": plainDoc,
				"synnax/doc.oracle":             liveHead + ruledDoc,
			})
			syncRepo()
			write(
				"schemas/synnax/versions/doc/v1.oracle",
				"Doc struct extends v0.Doc {\n    title string\n}\n",
			)
			write("schemas/synnax/doc.oracle", liveHead+strings.Replace(
				ruledDoc, "\n\n    @go", "\n    title string\n\n    @go", 1,
			))
			syncRepo()
		})

		It("fails when the demoted package loses a live rule", func() {
			Expect(failGate(gateCode, "--gates=frozen")).
				To(ContainSubstring(frozenPackage))
		})

		It("passes once the version file records the rule", func() {
			write("schemas/synnax/versions/doc/v0.oracle", ruledDoc)
			syncRepo()
			succeed("check", "--gates=frozen")
		})
	})
})

// buildOracleBinary compiles the oracle binary once per test run and
// caches the path. Cmd-level e2e tests exercise the real binary so flag
// parsing, exit codes, and output go through the same paths users hit.
var oracleBinaryPath string

func buildOracleBinary() string {
	GinkgoHelper()
	if oracleBinaryPath != "" {
		return oracleBinaryPath
	}
	dir := MustSucceed(os.MkdirTemp("", "oracle-bin"))
	bin := filepath.Join(dir, "oracle")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	// -buildvcs=false keeps the build working inside a Git worktree, where VCS stamping
	// fails ("error obtaining VCS status: exit status 128").
	proc := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".")
	proc.Dir = MustSucceed(findOracleModuleRoot())
	Expect(proc.CombinedOutput()).ToNot(BeNil())
	oracleBinaryPath = bin
	DeferCleanup(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
		oracleBinaryPath = ""
	})
	return bin
}

// findOracleModuleRoot walks upward from cwd looking for the oracle
// module root (where go.mod lives at oracle/go.mod). Tests run from
// the cmd/ directory, so the module root is the parent.
func findOracleModuleRoot() (string, error) {
	GinkgoHelper()
	cwd := MustSucceed(os.Getwd())
	cur := cwd
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", os.ErrNotExist
		}
		cur = parent
	}
}
