// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// runStdin is exercised indirectly by every shell-out formatter, but
// these tests pin the wrapper's contract with cheap external commands
// (`cat`, `sh`) that are present on every supported platform.
//
// findPackageJSONDir / findPyProjectDir share the same walk-up logic.
// Both are tested directly here so the prettier/eslint/ruff tests can
// rely on them rather than re-implementing fixture discovery.

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestStdinRunPipesStdinToStdout(t *testing.T) {
	out, err := stdinRun{Name: "cat", Stdin: []byte("hello")}.run(context.Background())
	if err != nil {
		t.Fatalf("cat failed: %v", err)
	}
	if string(out) != "hello" {
		t.Fatalf("expected 'hello', got %q", out)
	}
}

func TestStdinRunCapturesStderrInError(t *testing.T) {
	_, err := stdinRun{
		Name: "sh",
		Args: []string{"-c", "echo BOOM 1>&2; exit 7"},
	}.run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if msg := err.Error(); !contains(msg, "BOOM") {
		t.Fatalf("expected stderr in error, got %q", msg)
	}
}

func TestStdinRunAllowExitReturnsStdoutOnNonZeroExit(t *testing.T) {
	out, err := stdinRun{
		Name:      "sh",
		Args:      []string{"-c", "echo ok; exit 3"},
		AllowExit: true,
	}.run(context.Background())
	if err != nil {
		t.Fatalf("AllowExit should swallow exit code, got %v", err)
	}
	if string(out) != "ok\n" {
		t.Fatalf("expected 'ok\\n', got %q", out)
	}
}

func TestStdinRunHonorsDir(t *testing.T) {
	dir := t.TempDir()
	// `pwd -P` resolves symlinks (e.g. macOS's /var -> /private/var) so the
	// shell-reported path matches the EvalSymlinks-resolved fixture path.
	out, err := stdinRun{
		Name: "sh",
		Args: []string{"-c", "pwd -P"},
		Dir:  dir,
	}.run(context.Background())
	if err != nil {
		t.Fatalf("sh failed: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if len(got) > 0 && got[len(got)-1] == '\n' {
		got = got[:len(got)-1]
	}
	if got != resolved {
		t.Fatalf("expected pwd=%q, got %q", resolved, got)
	}
}

func TestFindPackageJSONDirWalksUp(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg", "package.json"), `{"name":"x"}`)
	deep := filepath.Join(root, "pkg", "src", "nested", "foo.ts")
	got := findPackageJSONDir(deep)
	want := filepath.Join(root, "pkg")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFindPackageJSONDirReturnsEmptyWhenAbsent(t *testing.T) {
	root := t.TempDir()
	got := findPackageJSONDir(filepath.Join(root, "src", "foo.ts"))
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestFindPyProjectDirWalksUp(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg", "pyproject.toml"), `[project]
name = "x"
`)
	deep := filepath.Join(root, "pkg", "src", "nested", "foo.py")
	got := findPyProjectDir(deep)
	want := filepath.Join(root, "pkg")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
