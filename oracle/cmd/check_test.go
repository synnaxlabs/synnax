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
	"os"
	"os/exec"
	"path/filepath"

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
		Expect(os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(repoRoot, "schemas"), 0755)).To(Succeed())
	})

	writeSchema := func(name, body string) {
		path := filepath.Join(repoRoot, "schemas", name+".oracle")
		Expect(os.WriteFile(path, []byte(body), 0644)).To(Succeed())
	}

	runOracleCheck := func(args ...string) (string, int) {
		bin := buildOracleBinary()
		cmd := exec.Command(bin, append([]string{"check"}, args...)...)
		cmd.Dir = repoRoot
		out, _ := cmd.CombinedOutput()
		return string(out), cmd.ProcessState.ExitCode()
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
    other other.Missing
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

// buildOracleBinary compiles the oracle binary once per test run and
// caches the path. Cmd-level e2e tests exercise the real binary so flag
// parsing, exit codes, and output go through the same paths users hit.
var oracleBinaryPath string

func buildOracleBinary() string {
	if oracleBinaryPath != "" {
		return oracleBinaryPath
	}
	dir := MustSucceed(os.MkdirTemp("", "oracle-bin"))
	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = MustSucceed(findOracleModuleRoot())
	out := MustSucceed(cmd.CombinedOutput())
	_ = out
	oracleBinaryPath = bin
	DeferCleanup(func() {
		_ = os.RemoveAll(dir)
		oracleBinaryPath = ""
	})
	return bin
}

// findOracleModuleRoot walks upward from cwd looking for the oracle
// module root (where go.mod lives at oracle/go.mod). Tests run from
// the cmd/ directory, so the module root is the parent.
func findOracleModuleRoot() (string, error) {
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
