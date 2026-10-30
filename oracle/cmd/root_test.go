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
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/cmd"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// capture returns a pipe to stand in for a standard stream and a function that
// closes it and returns what was written to it.
func capture() (*os.File, func() string) {
	GinkgoHelper()
	r, w := MustSucceed2(os.Pipe())
	printed := make(chan []byte, 1)
	go func() {
		defer GinkgoRecover()
		printed <- MustSucceed(io.ReadAll(r))
	}()
	return w, func() string {
		GinkgoHelper()
		Expect(w.Close()).To(Succeed())
		var out []byte
		Eventually(printed).Should(Receive(&out))
		Expect(r.Close()).To(Succeed())
		return string(out)
	}
}

// run executes oracle with args and returns what it printed to stdout and stderr and
// its exit code.
func run(args ...string) (string, string, int) {
	GinkgoHelper()
	stdout, stderr := os.Stdout, os.Stderr
	var readStdout, readStderr func() string
	os.Stdout, readStdout = capture()
	os.Stderr, readStderr = capture()
	code := cmd.Execute(args)
	os.Stdout, os.Stderr = stdout, stderr
	return readStdout(), readStderr(), code
}

// succeed runs oracle with args, expects exit code 0, and returns what it printed to
// stdout.
func succeed(args ...string) string {
	GinkgoHelper()
	stdout, stderr, code := run(args...)
	Expect(code).To(BeZero(), stdout+stderr)
	return stdout
}

// fail runs oracle with args, expects exit code code, and returns what it printed to
// stderr.
func fail(code int, args ...string) string {
	GinkgoHelper()
	stdout, stderr, actual := run(args...)
	Expect(actual).To(Equal(code), stdout+stderr)
	return stderr
}

// failGate runs oracle check with args, expects the gate failure code, expects
// nothing on stderr, and returns the report printed to stdout.
func failGate(code int, args ...string) string {
	GinkgoHelper()
	stdout, stderr, actual := run(append([]string{"check"}, args...)...)
	Expect(actual).To(Equal(code), stdout+stderr)
	Expect(stderr).To(BeEmpty())
	return stdout
}

// printOnce matches output that contains msg exactly once.
func printOnce(msg string) types.GomegaMatcher {
	return WithTransform(
		func(out string) int { return strings.Count(out, msg) },
		Equal(1),
	)
}

// setupMiniRepo creates a temp directory that looks like a minimal oracle repo:
//   - .git/ directory (so paths.RepoRoot finds it)
//   - schemas/*.oracle files
//
// It changes the working directory to the temp dir and returns a cleanup function
// that restores the original working directory.
func setupMiniRepo(schemas map[string]string) (string, func()) {
	GinkgoHelper()
	origDir := MustSucceed(os.Getwd())
	repoDir := MustSucceed(os.MkdirTemp("", "oracle-test-repo"))

	// Create .git so paths.RepoRoot() finds this as the repo root.
	Expect(os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755)).To(Succeed())

	// Create schema files.
	schemasDir := filepath.Join(repoDir, "schemas")
	Expect(os.MkdirAll(schemasDir, 0o755)).To(Succeed())
	for name, content := range schemas {
		path := filepath.Join(schemasDir, name)
		Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
	}

	// Create a minimal license template so format.Default can build a
	// formatter registry. Required by the check command, which builds
	// the registry to run the generated-drift gate.
	licenseDir := filepath.Join(repoDir, "licenses", "headers")
	Expect(os.MkdirAll(licenseDir, 0o755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(licenseDir, "template.txt"),
		[]byte("Copyright {{YEAR}} Test Inc.\n"), 0o644)).To(Succeed())

	// cd into the repo so paths.RepoRoot() finds it.
	Expect(os.Chdir(repoDir)).To(Succeed())

	return repoDir, func() {
		Expect(os.Chdir(origDir)).To(Succeed())
		Expect(os.RemoveAll(repoDir)).To(Succeed())
	}
}

var _ = Describe("help", func() {
	It("Should describe oracle and list every subcommand and the verbose flag", func() {
		Expect(succeed("--help")).To(SatisfyAll(
			ContainSubstring("Schema-first code generation"),
			ContainSubstring("check"),
			ContainSubstring("fmt"),
			ContainSubstring("lsp"),
			ContainSubstring("migrate"),
			ContainSubstring("sync"),
			ContainSubstring("-v, --verbose"),
		))
	})

	DescribeTable(
		"Should print one error and exit with 1 for a usage error",
		func(msg string, args ...string) {
			Expect(fail(1, args...)).To(SatisfyAll(printOnce(msg), printOnce("✗")))
		},
		Entry("an unknown subcommand",
			`unknown command "nonexistent" for "oracle"`, "nonexistent"),
		Entry("a misspelled subcommand", "Did you mean this?\n\tsync", "sinc"),
		Entry("an unknown flag", "unknown flag: --bogus", "sync", "--bogus"),
		Entry("an extra argument",
			`unknown command "extra" for "oracle surface"`, "surface", "extra"),
	)
})

var _ = Describe("check command", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{
			"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
		})
	})

	AfterAll(func() { cleanup() })

	It("should validate well-formed schemas", func() {
		succeed("check")
	})
})

var _ = Describe("commands with no schemas", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{})
	})

	AfterAll(func() { cleanup() })

	DescribeTable("Should print one error when no schema files exist",
		func(command string) {
			Expect(fail(1, command)).To(printOnce("no schema files found"))
		},
		Entry("check", "check"),
		Entry("fmt", "fmt"),
		Entry("sync", "sync"),
	)
})

var _ = Describe("fmt command", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{
			"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
		})
	})

	AfterAll(func() { cleanup() })

	It("should format schema files without error", func() {
		succeed("fmt")
	})

	It("should pass check mode when already formatted", func() {
		succeed("fmt", "--check")
	})
})

var _ = Describe("fmt command with nested schema folders", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		// Schemas live in subdirectories (arc/synnax/x); the no-arg fmt default must
		// recurse to find all of them.
		_, cleanup := setupMiniRepo(map[string]string{
			// Intentionally unformatted in a nested folder.
			"synnax/user.oracle": "User struct {key uuid\nname   string}",
			"x/telem.oracle":     "Rate struct {\n    hz float64\n}\n",
		})
		DeferCleanup(func() { cleanup() })
	})

	It("discovers and flags unformatted schemas in subdirectories", func() {
		Expect(
			fail(1, "fmt", "--check"),
		).To(printOnce("file(s) need formatting"))
	})

	It("formats nested schemas in place", func() {
		succeed("fmt")

		succeed("fmt", "--check")
	})
})

var _ = Describe("fmt command with explicit file arguments", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		// Two unformatted schemas in different subfolders. Passing an explicit path
		// must format only that file and must not trigger recursive discovery of the
		// other.
		_, cleanup := setupMiniRepo(map[string]string{
			"synnax/user.oracle": "User struct {key uuid\nname   string}",
			"x/other.oracle":     "Other struct {a uuid\nb   string}",
		})
		DeferCleanup(func() { cleanup() })
	})

	It("formats only the explicitly named file", func() {
		succeed("fmt", "schemas/synnax/user.oracle")

		succeed("fmt", "--check", "schemas/synnax/user.oracle")
	})

	It("leaves files not named in the arguments untouched", func() {
		Expect(
			fail(1, "fmt", "--check", "schemas/x/other.oracle"),
		).To(printOnce("file(s) need formatting"))
	})
})

var _ = Describe("fmt command discovery error", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		repoDir, cleanup := setupMiniRepo(map[string]string{
			"synnax/user.oracle": "User struct {\n    key uuid\n}\n",
		})
		DeferCleanup(cleanup)
		locked := filepath.Join(repoDir, "schemas", "locked")
		Expect(os.MkdirAll(locked, 0o755)).To(Succeed())
		DenyDirRead(locked)
	})

	It("propagates the recursive-discovery error when no arguments are given", func() {
		Expect(fail(1, "fmt")).To(printOnce("walk schema directory"))
	})
})

var _ = Describe("fmt command argument error", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup := setupMiniRepo(map[string]string{
			"synnax/user.oracle": "User struct {\n    key uuid\n}\n",
		})
		DeferCleanup(func() { cleanup() })
	})

	It("propagates the glob error for a malformed pattern argument", func() {
		Expect(fail(1, "fmt", "schemas/[")).To(printOnce("invalid glob pattern"))
	})
})

var _ = Describe("fmt command with unformatted schemas", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{
			// Intentionally poorly formatted (extra spaces, no newline).
			"user.oracle": "User struct {key uuid\nname   string}",
		})
	})

	AfterAll(func() { cleanup() })

	It("should fail check mode when schemas need formatting", func() {
		Expect(fail(1, "fmt", "--check")).To(printOnce("1 file(s) need formatting"))
	})
})

var _ = Describe("migrate command with nested schema folders", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		// Schemas live in subdirectories (synnax/x) after the folder restructure. The
		// bare migrate command must recurse to discover them rather than relying on a
		// flat schemas/*.oracle glob, which would find nothing and fail.
		repoDir, cleanup := setupMiniRepo(map[string]string{
			"synnax/user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
			"x/telem.oracle":     "Rate struct {\n    hz float64\n}\n",
		})
		svcDir := filepath.Join(repoDir, "core", "pkg", "service", "user")
		Expect(os.MkdirAll(svcDir, 0o755)).To(Succeed())
		DeferCleanup(func() { cleanup() })
	})

	It("discovers schemas in subdirectories instead of failing the flat glob", func() {
		Expect(fail(1, "migrate")).To(SatisfyAll(
			printOnce("name the resources to bump"),
			Not(ContainSubstring("no schema files found")),
		))
	})
})

var _ = Describe("fmt command with file patterns", func() {
	const (
		unformatted = "A struct {key uuid}"
		formatted   = "A struct {\n    key uuid\n}\n"
	)
	var repoDir string

	BeforeEach(func() {
		var cleanup func()
		repoDir, cleanup = setupMiniRepo(map[string]string{
			"a.oracle": unformatted,
			"b.oracle": unformatted,
		})
		DeferCleanup(cleanup)
	})

	schema := func(name string) string {
		GinkgoHelper()
		return string(MustSucceed(os.ReadFile(filepath.Join(repoDir, "schemas", name))))
	}

	DescribeTable(
		"Should format every file a pattern matches",
		func(pattern func() string) {
			succeed("fmt", pattern())
			Expect(schema("a.oracle")).To(Equal(formatted))
			Expect(schema("b.oracle")).To(Equal(formatted))
		},
		Entry("relative to the repository root", func() string {
			return "schemas/*.oracle"
		}),
		Entry("absolute", func() string {
			return filepath.Join(repoDir, "schemas", "*.oracle")
		}),
	)

	It("Should fail when no file matches a pattern", func() {
		Expect(fail(1, "fmt", "nonexistent/*.xyz")).
			To(printOnce("no schema files found"))
	})
})

var _ = Describe("fmt --diff flag", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{
			"user.oracle": "User struct {key uuid\nname   string}",
		})
	})

	AfterAll(func() { cleanup() })

	It("should report changes without writing in diff mode", func() {
		succeed("fmt", "--diff")
	})
})

var _ = Describe("check command with bad schema", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{
			"bad.oracle": "this is not valid oracle syntax {{{{",
		})
	})

	AfterAll(func() { cleanup() })

	It("should fail validation on invalid schemas", func() {
		Expect(failGate(check.FailureCodes["analyze"])).
			To(printOnce("1 gate(s) failed"))
	})

	It("should print one error when surface cannot analyze the schemas", func() {
		Expect(fail(1, "surface")).To(printOnce("analysis failed"))
	})
})
