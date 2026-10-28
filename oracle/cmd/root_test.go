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
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/cmd"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	_, err := root.ExecuteC()
	return buf.String(), err
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

var _ = Describe("New", func() {
	It("should create a root command with the correct use string", func() {
		root := cmd.New()
		Expect(root.Use).To(Equal("oracle"))
	})

	It("should register all subcommands", func() {
		root := cmd.New()
		names := make([]string, 0, len(root.Commands()))
		for _, sub := range root.Commands() {
			names = append(names, sub.Name())
		}
		Expect(names).To(ContainElements("check", "fmt", "lsp", "migrate", "sync"))
	})

	It("should have the verbose persistent flag", func() {
		root := cmd.New()
		flag := root.PersistentFlags().Lookup("verbose")
		Expect(flag).NotTo(BeNil())
		Expect(flag.Shorthand).To(Equal("v"))
	})

	It("should print help when run with --help", func() {
		root := cmd.New()
		output := MustSucceed(executeCommand(root, "--help"))
		Expect(output).To(ContainSubstring("Schema-first code generation"))
	})

	It("should return an error for unknown subcommands", func() {
		root := cmd.New()
		Expect(
			executeCommand(root, "nonexistent"),
		).Error().
			To(MatchError(ContainSubstring(`unknown command "nonexistent"`)))
	})

	It("should produce independent command trees on each call", func() {
		cmd1 := cmd.New()
		cmd2 := cmd.New()
		Expect(cmd1).NotTo(BeIdenticalTo(cmd2))
		Expect(cmd1.Commands()).To(HaveLen(len(cmd2.Commands())))
	})
})

var _ = Describe("Execute", func() {
	BeforeEach(func() {
		_, cleanup := setupMiniRepo(map[string]string{
			"user.oracle": "User struct {key uuid\nname   string}",
		})
		DeferCleanup(cleanup)
	})

	It("Should return 0 when the command succeeds", func() {
		Expect(cmd.Execute([]string{"--help"})).To(Equal(0))
	})

	It("Should return the code of the gate that failed", func() {
		Expect(cmd.Execute([]string{"check", "--gates=format"})).
			To(Equal(check.FailureCodes["format"]))
	})

	It("Should return 1 for any other failure", func() {
		Expect(cmd.Execute([]string{"nonexistent"})).To(Equal(1))
	})
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
		root := cmd.New()
		MustSucceed(executeCommand(root, "check"))
	})
})

var _ = Describe("check command with no schemas", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo(map[string]string{})
	})

	AfterAll(func() { cleanup() })

	It("should error when no schema files exist", func() {
		root := cmd.New()
		Expect(
			executeCommand(root, "check"),
		).Error().
			To(MatchError(ContainSubstring("no schema files found")))
	})
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
		root := cmd.New()
		MustSucceed(executeCommand(root, "fmt"))
	})

	It("should pass check mode when already formatted", func() {
		root := cmd.New()
		MustSucceed(executeCommand(root, "fmt", "--check"))
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
		root := cmd.New()
		Expect(executeCommand(root, "fmt", "--check")).Error().
			To(MatchError(ContainSubstring("files need formatting")))
	})

	It("formats nested schemas in place", func() {
		root := cmd.New()
		Expect(executeCommand(root, "fmt")).To(Equal(""))

		root = cmd.New()
		Expect(executeCommand(root, "fmt", "--check")).To(Equal(""))
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
		root := cmd.New()
		Expect(executeCommand(root, "fmt", "schemas/synnax/user.oracle")).
			To(Equal(""))

		root = cmd.New()
		Expect(executeCommand(root, "fmt", "--check", "schemas/synnax/user.oracle")).
			To(Equal(""))
	})

	It("leaves files not named in the arguments untouched", func() {
		root := cmd.New()
		Expect(
			executeCommand(root, "fmt", "--check", "schemas/x/other.oracle"),
		).Error().
			To(MatchError(ContainSubstring("files need formatting")))
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
		root := cmd.New()
		Expect(executeCommand(root, "fmt")).Error().
			To(MatchError(ContainSubstring("walk schema directory")))
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
		root := cmd.New()
		Expect(executeCommand(root, "fmt", "schemas/[")).Error().
			To(MatchError(ContainSubstring("invalid glob pattern")))
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
		root := cmd.New()
		Expect(
			executeCommand(root, "fmt", "--check"),
		).Error().
			To(MatchError(ContainSubstring("files need formatting")))
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
		root := cmd.New()
		Expect(executeCommand(root, "migrate")).Error().
			NotTo(MatchError(ContainSubstring("no schema files found")))
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
			MustSucceed(executeCommand(cmd.New(), "fmt", pattern()))
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

	It("Should format only the files a pattern matches", func() {
		MustSucceed(executeCommand(cmd.New(), "fmt", "schemas/a.oracle"))
		Expect(schema("a.oracle")).To(Equal(formatted))
		Expect(schema("b.oracle")).To(Equal(unformatted))
	})

	It("Should fail when no file matches a pattern", func() {
		Expect(executeCommand(cmd.New(), "fmt", "nonexistent/*.xyz")).Error().
			To(MatchError("no schema files found"))
	})

	It("Should reject a malformed pattern", func() {
		Expect(executeCommand(cmd.New(), "fmt", "schemas/[")).Error().
			To(MatchError(ContainSubstring("invalid glob pattern")))
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
		root := cmd.New()
		MustSucceed(executeCommand(root, "fmt", "--diff"))
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
		root := cmd.New()
		Expect(
			executeCommand(root, "check"),
		).Error().
			To(MatchError(ContainSubstring("1 gate(s) failed")))
	})
})
