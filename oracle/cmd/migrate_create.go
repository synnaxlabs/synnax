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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/errors"
)

var migrateCreateService string

func newMigrateCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a hand-written migration",
		Long:  "Creates a migration file with boilerplate, pre-wired dependency on the latest schema migration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runMigrateCreate(args[0]); err != nil {
				printError(err.Error())
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&migrateCreateService, "service", "", "Service output path (e.g. core/pkg/service/ranger)")
	return cmd
}

var migrationScaffoldTmpl = template.Must(template.New("scaffold").Parse(
	`package v{{.Version}}

import (
	"context"

	"github.com/synnaxlabs/x/gorp"
)

// New{{.PascalName}}Migration creates a hand-written migration.
// TODO: implement migration logic.
func New{{.PascalName}}Migration() migrate.Migration {
	return gorp.WithDependencies(
		gorp.NewRawMigration("{{.Name}}", func(ctx context.Context, tx gorp.Tx) error {
			// TODO: implement
			return nil
		}),
		"{{.DependsOn}}",
	)
}
`))

func findMigrationVersions(migrationsDir string) ([]int, error) {
	entries, err := os.ReadDir(migrationsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var versions []int
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "v") {
			continue
		}
		v := 0
		if _, err := fmt.Sscanf(e.Name(), "v%d", &v); err == nil {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

func runMigrateCreate(name string) (err error) {
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "must be run within a git repository")
	}

	servicePath := migrateCreateService
	if servicePath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get working directory")
		}
		rel, err := filepath.Rel(repoRoot, cwd)
		if err != nil {
			return errors.Wrap(err, "failed to compute relative path")
		}
		servicePath = rel
	}

	version, err := readCoreVersion(repoRoot)
	if err != nil {
		return errors.Wrap(err, "failed to read core version")
	}

	existingVersions, err := findMigrationVersions(filepath.Join(repoRoot, servicePath, "migrations"))
	if err != nil {
		return errors.Wrapf(err, "failed to discover existing migrations for %s", servicePath)
	}

	dependsOn := "msgpack_to_orc"
	effectiveVersion := version
	if len(existingVersions) > 0 {
		latest := existingVersions[len(existingVersions)-1]
		dependsOn = fmt.Sprintf("v%d_schema_migration", latest)
		effectiveVersion = latest
	}

	vDir := fmt.Sprintf("v%d", effectiveVersion)
	migrationDir := filepath.Join(repoRoot, servicePath, "migrations", vDir)
	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create migration directory")
	}

	outPath := filepath.Join(migrationDir, name+".go")
	if _, statErr := os.Stat(outPath); statErr == nil {
		return errors.Newf("migration file already exists: %s", outPath)
	}

	pascalName := lo.PascalCase(name)
	f, err := os.Create(outPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s", outPath)
	}
	defer func() { err = errors.Combine(err, f.Close()) }()

	if err = migrationScaffoldTmpl.Execute(f, struct {
		Version    int
		Name       string
		PascalName string
		DependsOn  string
	}{
		Version:    effectiveVersion,
		Name:       name,
		PascalName: pascalName,
		DependsOn:  dependsOn,
	}); err != nil {
		return errors.Wrapf(err, "failed to write scaffold")
	}

	relPath, _ := filepath.Rel(repoRoot, outPath)
	printDim(fmt.Sprintf("  ✏️  %s ← implement migration logic", relPath))

	pkg := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(relPath))))
	printDim(fmt.Sprintf("  🔌 Wire New%sMigration() into your gorp.OpenTable call in %s/service.go",
		pascalName, strings.TrimPrefix(servicePath, "core/pkg/service/")))
	_ = pkg

	return nil
}
