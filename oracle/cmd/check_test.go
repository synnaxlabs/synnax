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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

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
		out, err := proc.CombinedOutput()
		Expect(err).To(Or(Succeed(), BeAssignableToTypeOf(&exec.ExitError{})))
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
			var repoDir string
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
			})
			releaseMiniRepo(repoDir)
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

var _ = Describe("check against the shipped release", func() {
	const (
		channel = `Channel struct {
    name string

    @go marshal
}
`
		keyedChannel = `Channel struct {
    name string
    key  uint32

    @go marshal
}
`
		chainDir = "schemas/synnax/versions/channel"
	)
	var (
		repoDir string
		cleanup func()
	)

	AfterEach(func() { cleanup() })

	git := func(args ...string) {
		GinkgoHelper()
		proc := exec.Command("git", append([]string{
			"-c", "user.name=Test", "-c", "user.email=test@example.com",
			"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false",
			// Checkouts keep LF, as the Synnax repository's .gitattributes forces.
			"-c", "core.autocrlf=false",
		}, args...)...)
		proc.Dir = repoDir
		out, err := proc.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), string(out))
	}

	// writeChain writes one version file per body and the live file that projects
	// the last one.
	writeChain := func(bodies ...string) {
		GinkgoHelper()
		for n, body := range bodies {
			Expect(os.WriteFile(
				filepath.Join(repoDir, chainDir, fmt.Sprintf("v%d.oracle", n)),
				[]byte(body), 0o644,
			)).To(Succeed())
		}
		header := fmt.Sprintf("// Copyright %d Test Inc.\n\n", time.Now().Year())
		Expect(os.WriteFile(
			filepath.Join(repoDir, "schemas", "synnax", "channel.oracle"),
			[]byte(header+bodies[len(bodies)-1]), 0o644,
		)).To(Succeed())
	}

	commit := func(bodies ...string) {
		GinkgoHelper()
		writeChain(bodies...)
		git("add", "-A")
		git("commit", "-q", "-m", "commit")
	}

	shipped := func(tag string) string {
		return "schemas/synnax/channel v1 is gone, which " + tag + " shipped"
	}

	Describe("outside a Git repository", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/versions/channel/v0.oracle": channel,
			})
			writeChain(channel)
		})

		It("Should fail when it cannot list the release tags", func() {
			Expect(
				fail(1, "check", "--gates=versions"),
			).To(And(
				printOnce("list the Core release tags"),
				ContainSubstring("not a git repository"),
			))
		})
	})

	Describe("in a Git repository", func() {
		BeforeEach(func() {
			repoDir, cleanup = setupMiniRepo(map[string]string{
				"synnax/versions/channel/v0.oracle": channel,
			})
			Expect(os.RemoveAll(filepath.Join(repoDir, ".git"))).To(Succeed())
			git("init", "-q", "-b", "main")
			commit(channel)
		})

		It("Should fail when HEAD descends only from release candidates", func() {
			git("tag", "core/v0.1.0-rc.1")
			Expect(
				fail(1, "check", "--gates=versions"),
			).To(printOnce("HEAD descends from no stable Core release tag"))
		})

		It("Should fail a stored shape that changes after a release", func() {
			git("tag", "core/v0.1.0")
			succeed("check", "--gates=versions")
			writeChain(keyedChannel)
			Expect(failGate(check.FailureCodes["versions"], "--gates=versions")).To(
				ContainSubstring("which core/v0.1.0 shipped"),
			)
		})

		It("Should fail a release whose version files are all gone", func() {
			git("tag", "core/v0.1.0")
			Expect(os.RemoveAll(filepath.Join(repoDir, "schemas/synnax/versions"))).
				To(Succeed())
			Expect(failGate(check.FailureCodes["versions"], "--gates=versions")).To(
				ContainSubstring("channel v0 is gone, which core/v0.1.0"),
			)
		})

		It("Should compare main with the newest patch on its newest train", func() {
			git("tag", "core/v0.1.0")
			git("checkout", "-q", "-b", "release/0.1")
			commit(channel, keyedChannel)
			git("tag", "core/v0.1.1")
			git("checkout", "-q", "main")
			Expect(failGate(check.FailureCodes["versions"], "--gates=versions")).To(
				ContainSubstring(shipped("core/v0.1.1")),
			)
		})

		It("Should compare a release branch with the newest patch on its train",
			func() {
				git("tag", "core/v0.1.0")
				git("branch", "release/0.1")
				commit(channel, keyedChannel)
				git("tag", "core/v0.2.0")
				git("checkout", "-q", "release/0.1")
				succeed("check", "--gates=versions")
			},
		)
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
