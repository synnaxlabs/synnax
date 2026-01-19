// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/version"
)

// Test types to simulate migrations
type ResourceV0 struct {
	Name string
}

func (e ResourceV0) GetVersion() version.Counter { return 0 }

type ResourceV1 struct {
	Title string
}

func (e ResourceV1) GetVersion() version.Counter { return 1 }

type ResourceV2 struct {
	Title       string
	Description string
}

func (e ResourceV2) GetVersion() version.Counter { return 2 }

var _ = Describe("Migrate", func() {
	Describe("NewMigrator", func() {
		var (
			migrations migrate.Migrations
			defaultV2  ResourceV2
		)

		BeforeEach(func() {
			// Create V0 to V1 migration
			migrateV0toV1 := migrate.CreateMigration(migrate.MigrationConfig[ResourceV0, ResourceV1]{
				Name: "resource",
				Migrate: func(ctx migrate.Context, v0 ResourceV0) (ResourceV1, error) {
					return ResourceV1{
						Title: v0.Name,
					}, nil
				},
			})

			// Create V1 to V2 migration
			migrateV1toV2 := migrate.CreateMigration(migrate.MigrationConfig[ResourceV1, ResourceV2]{
				Name: "resource",
				Migrate: func(ctx migrate.Context, v1 ResourceV1) (ResourceV2, error) {
					return ResourceV2{
						Title:       v1.Title,
						Description: "",
					}, nil
				},
			})

			migrations = migrate.Migrations{
				1: migrateV1toV2,
				0: migrateV0toV1,
			}

			defaultV2 = ResourceV2{
				Title:       "",
				Description: "",
			}
		})

		It("should migrate an resource from v0 to v2", func() {
			resource := ResourceV0{Name: "foo"}
			migrator := migrate.NewMigrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			Expect(result).To(Equal(ResourceV2{
				Title:       "foo",
				Description: "",
			}))
		})

		It("should migrate an resource from v1 to v2", func() {
			resource := ResourceV1{Title: "foo"}
			migrator := migrate.NewMigrator(migrate.MigratorConfig[ResourceV1, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			Expect(result).To(Equal(ResourceV2{
				Title:       "foo",
				Description: "",
			}))
		})

		It("should not migrate an resource that is already at v2", func() {
			resource := ResourceV2{
				Title:       "foo",
				Description: "bar",
			}
			migrator := migrate.NewMigrator(migrate.MigratorConfig[ResourceV2, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			Expect(result).To(Equal(resource))
		})

		It("should return default when migration fails", func() {
			migrator := migrate.NewMigrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: migrate.Migrations{},
				Default:    defaultV2,
			})

			resource := ResourceV0{Name: "foo"}
			result := migrator(resource)
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle migration errors gracefully", func() {
			// Create a migration that always fails
			failingMigration := migrate.CreateMigration(migrate.MigrationConfig[ResourceV0, ResourceV1]{
				Name: "resource",
				Migrate: func(ctx migrate.Context, v0 ResourceV0) (ResourceV1, error) {
					return ResourceV1{}, errors.Newf("intentional migration failure")
				},
			})

			failingMigrations := migrate.Migrations{
				0: failingMigration,
			}

			migrator := migrate.NewMigrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: failingMigrations,
				Default:    defaultV2,
			})

			resource := ResourceV0{Name: "foo"}
			result := migrator(resource)
			// Should return default when migration fails
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle missing migration for specific version", func() {
			// Create migrations map with a gap
			gappedMigrations := migrate.Migrations{
				1: migrations[1], // Keep V1 to V2 migration
			}

			migrator := migrate.NewMigrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: gappedMigrations,
				Default:    defaultV2,
			})

			resource := ResourceV0{Name: "foo"}
			result := migrator(resource)
			// Should return default when no migration path exists
			Expect(result).To(Equal(defaultV2))
		})
	})
})
