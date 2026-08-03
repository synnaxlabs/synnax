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
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
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
//   - core/pkg/version/VERSION
//
// It changes the working directory to the temp dir and returns a cleanup function
// that restores the original working directory.
func setupMiniRepo(version string, schemas map[string]string) (string, func()) {
	GinkgoHelper()
	origDir := MustSucceed(os.Getwd())
	repoDir := MustSucceed(os.MkdirTemp("", "oracle-test-repo"))

	// Create .git so paths.RepoRoot() finds this as the repo root.
	Expect(os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)).To(Succeed())

	// Create VERSION file.
	versionDir := filepath.Join(repoDir, "core", "pkg", "version")
	Expect(os.MkdirAll(versionDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(versionDir, "VERSION"), []byte(version), 0644)).To(Succeed())

	// Create schema files.
	schemasDir := filepath.Join(repoDir, "schemas")
	Expect(os.MkdirAll(schemasDir, 0755)).To(Succeed())
	for name, content := range schemas {
		path := filepath.Join(schemasDir, name)
		Expect(os.MkdirAll(filepath.Dir(path), 0755)).To(Succeed())
		Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	}

	// Create a minimal license template so format.Default can build a
	// formatter registry. Required by the check command, which builds
	// the registry to run the generated-drift gate.
	licenseDir := filepath.Join(repoDir, "licenses", "headers")
	Expect(os.MkdirAll(licenseDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(licenseDir, "template.txt"),
		[]byte("Copyright {{YEAR}} Test Inc.\n"), 0644)).To(Succeed())

	// cd into the repo so paths.RepoRoot() finds it.
	Expect(os.Chdir(repoDir)).To(Succeed())

	return repoDir, func() {
		Expect(os.Chdir(origDir)).To(Succeed())
		Expect(os.RemoveAll(repoDir)).To(Succeed())
	}
}

var _ = Describe("NewRootCmd", func() {
	It("should create a root command with the correct use string", func() {
		cmd := NewRootCmd()
		Expect(cmd.Use).To(Equal("oracle"))
	})

	It("should register all subcommands", func() {
		cmd := NewRootCmd()
		names := make([]string, 0, len(cmd.Commands()))
		for _, sub := range cmd.Commands() {
			names = append(names, sub.Name())
		}
		Expect(names).To(ContainElements("check", "fmt", "lsp", "migrate", "snapshot", "sync"))
	})

	It("should register migrate create as a subcommand of migrate", func() {
		cmd := NewRootCmd()
		var migrateCmd *cobra.Command
		for _, sub := range cmd.Commands() {
			if sub.Name() == "migrate" {
				migrateCmd = sub
				break
			}
		}
		Expect(migrateCmd).NotTo(BeNil())
		subNames := make([]string, 0)
		for _, sub := range migrateCmd.Commands() {
			subNames = append(subNames, sub.Name())
		}
		Expect(subNames).To(ContainElement("create"))
	})

	It("should have the verbose persistent flag", func() {
		cmd := NewRootCmd()
		flag := cmd.PersistentFlags().Lookup("verbose")
		Expect(flag).NotTo(BeNil())
		Expect(flag.Shorthand).To(Equal("v"))
	})

	It("should print help when run with --help", func() {
		cmd := NewRootCmd()
		output := MustSucceed(executeCommand(cmd, "--help"))
		Expect(output).To(ContainSubstring("Schema-first code generation"))
	})

	It("should return an error for unknown subcommands", func() {
		cmd := NewRootCmd()
		_, err := executeCommand(cmd, "nonexistent")
		Expect(err).To(HaveOccurred())
	})

	It("should produce independent command trees on each call", func() {
		cmd1 := NewRootCmd()
		cmd2 := NewRootCmd()
		Expect(cmd1).NotTo(BeIdenticalTo(cmd2))
		Expect(cmd1.Commands()).To(HaveLen(len(cmd2.Commands())))
	})
})

var _ = Describe("check command", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{
			"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
		})
	})

	AfterAll(func() { cleanup() })

	It("should validate well-formed schemas", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "check"))
	})
})

var _ = Describe("check command with no schemas", Ordered, func() {
	var (
		cleanup func()
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{})
	})

	AfterAll(func() { cleanup() })

	It("should error when no schema files exist", func() {
		cmd := NewRootCmd()
		_, err := executeCommand(cmd, "check")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("fmt command", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{
			"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
		})
	})

	AfterAll(func() { cleanup() })

	It("should format schema files without error", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "fmt"))
	})

	It("should pass check mode when already formatted", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "fmt", "--check"))
	})
})

var _ = Describe("fmt command with nested schema folders", Ordered, func() {

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		// Schemas live in subdirectories (arc/synnax/x); the no-arg fmt default must
		// recurse to find all of them.
		_, cleanup := setupMiniRepo("0.53.4", map[string]string{
			// Intentionally unformatted in a nested folder.
			"synnax/user.oracle": "User struct {key uuid\nname   string}",
			"x/telem.oracle":     "Rate struct {\n    hz float64\n}\n",
		})
		DeferCleanup(func() { cleanup() })
	})

	It("discovers and flags unformatted schemas in subdirectories", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "fmt", "--check")).Error().
			To(MatchError(ContainSubstring("files need formatting")))
	})

	It("formats nested schemas in place", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "fmt")).To(Equal(""))

		cmd = NewRootCmd()
		Expect(executeCommand(cmd, "fmt", "--check")).To(Equal(""))
	})
})

var _ = Describe("fmt command with explicit file arguments", Ordered, func() {

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		// Two unformatted schemas in different subfolders. Passing an explicit path
		// must format only that file and must not trigger recursive discovery of the
		// other.
		_, cleanup := setupMiniRepo("0.53.4", map[string]string{
			"synnax/user.oracle": "User struct {key uuid\nname   string}",
			"x/other.oracle":     "Other struct {a uuid\nb   string}",
		})
		DeferCleanup(func() { cleanup() })
	})

	It("formats only the explicitly named file", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "fmt", "schemas/synnax/user.oracle")).
			To(Equal(""))

		cmd = NewRootCmd()
		Expect(executeCommand(cmd, "fmt", "--check", "schemas/synnax/user.oracle")).
			To(Equal(""))
	})

	It("leaves files not named in the arguments untouched", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "fmt", "--check", "schemas/x/other.oracle")).Error().
			To(MatchError(ContainSubstring("files need formatting")))
	})
})

var _ = Describe("fmt command discovery error", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		if os.Geteuid() == 0 {
			Skip("filesystem permissions are bypassed when running as root")
		}
		repoDir, cleanup := setupMiniRepo("0.53.4", map[string]string{
			"synnax/user.oracle": "User struct {\n    key uuid\n}\n",
		})
		locked := filepath.Join(repoDir, "schemas", "locked")
		Expect(os.MkdirAll(locked, 0755)).To(Succeed())
		Expect(os.Chmod(locked, 0000)).To(Succeed())
		DeferCleanup(func() {
			if locked != "" {
				Expect(os.Chmod(locked, 0755)).To(Succeed())
			}
			if cleanup != nil {
				cleanup()
			}
		})
	})

	It("propagates the recursive-discovery error when no arguments are given", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "fmt")).Error().
			To(MatchError(ContainSubstring("walk schema directory")))
	})
})

var _ = Describe("fmt command argument error", Ordered, func() {

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup := setupMiniRepo("0.53.4", map[string]string{
			"synnax/user.oracle": "User struct {\n    key uuid\n}\n",
		})
		DeferCleanup(func() { cleanup() })
	})

	It("propagates the glob error for a malformed pattern argument", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "fmt", "schemas/[")).Error().
			To(MatchError(ContainSubstring("invalid glob pattern")))
	})
})

var _ = Describe("fmt command with unformatted schemas", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{
			// Intentionally poorly formatted (extra spaces, no newline).
			"user.oracle": "User struct {key uuid\nname   string}",
		})
	})

	AfterAll(func() { cleanup() })

	It("should fail check mode when schemas need formatting", func() {
		cmd := NewRootCmd()
		_, err := executeCommand(cmd, "fmt", "--check")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("snapshot command", Ordered, func() {
	var (
		repoDir string
		cleanup func()
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		repoDir, cleanup = setupMiniRepo("0.53.4", map[string]string{
			"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
		})
	})

	AfterAll(func() { cleanup() })

	It("should create a snapshot at the current version", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "snapshot"))

		snapshotFile := filepath.Join(repoDir, "schemas", "snapshots", "v53", "user.oracle")
		Expect(snapshotFile).To(BeAnExistingFile())

		content := string(MustSucceed(os.ReadFile(snapshotFile)))
		Expect(content).To(ContainSubstring("User struct"))
	})
})

var _ = Describe("migrate create command", Ordered, func() {
	var (
		repoDir string
		cleanup func()
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		repoDir, cleanup = setupMiniRepo("0.53.4", map[string]string{
			"user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
		})
		// Create a service directory to act as CWD for migrate create.
		svcDir := filepath.Join(repoDir, "core", "pkg", "service", "user")
		Expect(os.MkdirAll(svcDir, 0755)).To(Succeed())
	})

	AfterAll(func() { cleanup() })

	It("should scaffold a migration file", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "migrate", "create", "add_email",
			"--service", "core/pkg/service/user"))

		migrationFile := filepath.Join(repoDir, "core", "pkg", "service", "user",
			"migrations", "v53", "add_email.go")
		Expect(migrationFile).To(BeAnExistingFile())

		content := string(MustSucceed(os.ReadFile(migrationFile)))
		Expect(content).To(ContainSubstring("package v53"))
		Expect(content).To(ContainSubstring("NewAddEmailMigration"))
		Expect(content).To(ContainSubstring("migrate.Migration"))
	})

	It("should error when migration file already exists", func() {
		cmd := NewRootCmd()
		_, err := executeCommand(cmd, "migrate", "create", "add_email",
			"--service", "core/pkg/service/user")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("migrate command with nested schema folders", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		// Schemas live in subdirectories (synnax/x) after the folder restructure. The
		// bare migrate command must recurse to discover them rather than relying on a
		// flat schemas/*.oracle glob, which would find nothing and fail.
		repoDir, cleanup := setupMiniRepo("0.53.4", map[string]string{
			"synnax/user.oracle": "User struct {\n    key  uuid\n    name string\n}\n",
			"x/telem.oracle":     "Rate struct {\n    hz float64\n}\n",
		})
		svcDir := filepath.Join(repoDir, "core", "pkg", "service", "user")
		Expect(os.MkdirAll(svcDir, 0755)).To(Succeed())
		DeferCleanup(func() { cleanup() })
	})

	It("discovers schemas in subdirectories instead of failing the flat glob", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "migrate")).Error().
			NotTo(MatchError(ContainSubstring("no schema files found")))
	})
})

var _ = Describe("migrate command with no schemas", Ordered, func() {
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup := setupMiniRepo("0.53.4", map[string]string{})
		DeferCleanup(func() { cleanup() })
	})

	It("errors when no schema files exist", func() {
		cmd := NewRootCmd()
		Expect(executeCommand(cmd, "migrate")).Error().
			To(MatchError(ContainSubstring("no schema files found")))
	})
})

var _ = Describe("migrate create with existing migrations", Ordered, func() {
	var (
		repoDir string
		cleanup func()
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		repoDir, cleanup = setupMiniRepo("0.54.0", map[string]string{
			"user.oracle": "User struct {\n    key uuid\n}\n",
		})
		svcDir := filepath.Join(repoDir, "core", "pkg", "service", "user")
		Expect(os.MkdirAll(svcDir, 0755)).To(Succeed())
		// Create a pre-existing migration version directory.
		Expect(os.MkdirAll(filepath.Join(svcDir, "migrations", "v53"), 0755)).To(Succeed())
	})

	AfterAll(func() { cleanup() })

	It("should scaffold into the latest existing version directory", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "migrate", "create", "fix_index",
			"--service", "core/pkg/service/user"))

		migrationFile := filepath.Join(repoDir, "core", "pkg", "service", "user",
			"migrations", "v53", "fix_index.go")
		Expect(migrationFile).To(BeAnExistingFile())

		content := string(MustSucceed(os.ReadFile(migrationFile)))
		Expect(content).To(ContainSubstring(`gorp.NewMigration("fix_index"`))
	})
})

var _ = Describe("expandGlobs", func() {
	var (
		repoDir string
		cleanup func()
	)

	BeforeEach(func() {
		repoDir, cleanup = setupMiniRepo("0.1.0", map[string]string{
			"a.oracle": "A struct {\n    key uuid\n}\n",
			"b.oracle": "B struct {\n    key uuid\n}\n",
		})
	})

	AfterEach(func() { cleanup() })

	It("should expand a valid glob pattern", func() {
		files := MustSucceed(expandGlobs([]string{"schemas/*.oracle"}, repoDir))
		Expect(files).To(HaveLen(2))
	})

	It("should return empty for non-matching globs", func() {
		files := MustSucceed(expandGlobs([]string{"nonexistent/*.xyz"}, repoDir))
		Expect(files).To(BeEmpty())
	})

	It("should return sorted results", func() {
		files := MustSucceed(expandGlobs([]string{"schemas/*.oracle"}, repoDir))
		for i := 1; i < len(files); i++ {
			Expect(files[i] >= files[i-1]).To(BeTrue())
		}
	})

	It("should handle absolute patterns", func() {
		pattern := filepath.Join(repoDir, "schemas", "*.oracle")
		files := MustSucceed(expandGlobs([]string{pattern}, repoDir))
		Expect(files).To(HaveLen(2))
	})
})

var _ = Describe("snapshot command without schemas", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{})
	})

	AfterAll(func() { cleanup() })

	It("should succeed even with no schema files", func() {
		cmd := NewRootCmd()
		// snapshot walks the directory; no .oracle files means nothing copied,
		// but it should not error.
		MustSucceed(executeCommand(cmd, "snapshot"))
	})
})

var _ = Describe("fmt --diff flag", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{
			"user.oracle": "User struct {key uuid\nname   string}",
		})
	})

	AfterAll(func() { cleanup() })

	It("should report changes without writing in diff mode", func() {
		cmd := NewRootCmd()
		MustSucceed(executeCommand(cmd, "fmt", "--diff"))
	})
})

var _ = Describe("check command with bad schema", Ordered, func() {
	var cleanup func()

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		_, cleanup = setupMiniRepo("0.53.4", map[string]string{
			"bad.oracle": "this is not valid oracle syntax {{{{",
		})
	})

	AfterAll(func() { cleanup() })

	It("should fail validation on invalid schemas", func() {
		cmd := NewRootCmd()
		_, err := executeCommand(cmd, "check")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("buildPluginRegistry", func() {
	It("should register all expected plugins", func() {
		registry := MustSucceed(buildPluginRegistry())
		expectedPlugins := []string{
			"ts/types", "go/types", "py/types", "pb/types",
			"cpp/types", "cpp/json", "cpp/pb", "go/pb", "go/marshal",
		}
		for _, name := range expectedPlugins {
			Expect(registry.Get(name)).NotTo(BeNil(),
				fmt.Sprintf("plugin %q should be registered", name))
		}
	})
})

var _ = Describe("syncOutputs", func() {
	var (
		tmpDir     string
		formatters *format.Registry
		cache      *format.Cache
	)

	BeforeEach(func() {
		tmpDir = MustSucceed(os.MkdirTemp("", "sync"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})
		formatters = format.NewRegistry()
		cache = format.LoadCache(tmpDir)
	})

	resultWith := func(files map[string][]plugin.File) *pipeline.Result {
		return &pipeline.Result{
			Outputs:   files,
			Deletions: map[string][]string{},
		}
	}

	It("should write new files and report them", func(ctx SpecContext) {
		result := resultWith(map[string][]plugin.File{
			"test": {{Path: "out/types.gen.go", Content: []byte("package out")}},
		})
		sr := MustSucceed(syncOutputs(ctx, result, tmpDir, formatters, cache, 1))
		Expect(sr.Written).To(HaveLen(1))
		Expect(sr.Unchanged).To(BeEmpty())

		content := string(MustSucceed(os.ReadFile(filepath.Join(tmpDir, "out", "types.gen.go"))))
		Expect(content).To(Equal("package out"))
	})

	It("should skip unchanged files", func(ctx SpecContext) {
		outDir := filepath.Join(tmpDir, "out")
		Expect(os.MkdirAll(outDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(outDir, "types.gen.go"), []byte("package out"), 0644)).To(Succeed())

		result := resultWith(map[string][]plugin.File{
			"test": {{Path: "out/types.gen.go", Content: []byte("package out")}},
		})
		sr := MustSucceed(syncOutputs(ctx, result, tmpDir, formatters, cache, 1))
		Expect(sr.Written).To(BeEmpty())
		Expect(sr.Unchanged).To(HaveLen(1))
	})

	It("should overwrite files with different content", func(ctx SpecContext) {
		outDir := filepath.Join(tmpDir, "out")
		Expect(os.MkdirAll(outDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(outDir, "types.gen.go"), []byte("old"), 0644)).To(Succeed())

		result := resultWith(map[string][]plugin.File{
			"test": {{Path: "out/types.gen.go", Content: []byte("new")}},
		})
		sr := MustSucceed(syncOutputs(ctx, result, tmpDir, formatters, cache, 1))
		Expect(sr.Written).To(HaveLen(1))
		Expect(sr.ByPlugin["test"]).To(HaveLen(1))
	})

	It("should skip via cache on second sync with identical raw bytes", func(ctx SpecContext) {
		result := resultWith(map[string][]plugin.File{
			"test": {{Path: "out/types.gen.go", Content: []byte("package out")}},
		})
		sr1 := MustSucceed(syncOutputs(ctx, result, tmpDir, formatters, cache, 1))
		Expect(sr1.Written).To(HaveLen(1))

		sr2 := MustSucceed(syncOutputs(ctx, result, tmpDir, formatters, cache, 1))
		Expect(sr2.Written).To(BeEmpty())
		Expect(sr2.Skipped).To(HaveLen(1))
	})
})
