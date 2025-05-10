package migrate_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/version"
)

// Test types to simulate migrations
type ResourceV0 struct {
	Version version.Semantic
	Name    string
}

func (e ResourceV0) GetVersion() version.Semantic { return e.Version }

type ResourceV1 struct {
	Version version.Semantic
	Title   string
}

func (e ResourceV1) GetVersion() version.Semantic { return e.Version }

type ResourceV2 struct {
	Version     version.Semantic
	Title       string
	Description string
}

func (e ResourceV2) GetVersion() version.Semantic { return e.Version }

var _ = Describe("Migrate", func() {
	Describe("Migrator", func() {
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
						Version: "1.0.0",
						Title:   v0.Name,
					}, nil
				},
			})

			// Create V1 to V2 migration
			migrateV1toV2 := migrate.CreateMigration(migrate.MigrationConfig[ResourceV1, ResourceV2]{
				Name: "resource",
				Migrate: func(ctx migrate.Context, v1 ResourceV1) (ResourceV2, error) {
					return ResourceV2{
						Version:     "2.0.0",
						Title:       v1.Title,
						Description: "",
					}, nil
				},
			})

			// Set up unified migrations map
			migrations = migrate.Migrations{
				"0.0.0": migrateV0toV1,
				"1.0.0": migrateV1toV2,
			}

			defaultV2 = ResourceV2{
				Version:     "2.0.0",
				Title:       "",
				Description: "",
			}
		})

		It("should migrate an resource from v0 to v2", func() {
			resource := ResourceV0{Version: "0.0.0", Name: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			Expect(result).To(Equal(ResourceV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "",
			}))
		})

		It("should migrate an resource from v1 to v2", func() {
			resource := ResourceV1{Version: "1.0.0", Title: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV1, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			Expect(result).To(Equal(ResourceV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "",
			}))
		})

		It("should not migrate an resource that is already at v2", func() {
			resource := ResourceV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "bar",
			}
			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV2, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			Expect(result).To(Equal(resource))
		})

		It("should return default when migration fails", func() {
			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: migrate.Migrations{},
				Default:    defaultV2,
			})

			resource := ResourceV0{Version: "0.0.0", Name: "foo"}
			result := migrator(resource)
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle empty version with default version", func() {
			resource := ResourceV0{Version: "", Name: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:           "resource",
				Migrations:     migrations,
				Default:        defaultV2,
				DefaultVersion: "0.0.0",
			})

			result := migrator(resource)
			Expect(result).To(Equal(ResourceV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "",
			}))
		})

		It("should handle migration errors gracefully", func() {
			// Create a migration that always fails
			failingMigration := migrate.CreateMigration(migrate.MigrationConfig[ResourceV0, ResourceV1]{
				Name: "resource",
				Migrate: func(ctx migrate.Context, v0 ResourceV0) (ResourceV1, error) {
					return ResourceV1{}, fmt.Errorf("intentional migration failure")
				},
			})

			failingMigrations := migrate.Migrations{
				"0.0.0": failingMigration,
			}

			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: failingMigrations,
				Default:    defaultV2,
			})

			resource := ResourceV0{Version: "0.0.0", Name: "foo"}
			result := migrator(resource)
			// Should return default when migration fails
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle invalid version formats", func() {
			resource := ResourceV0{Version: "invalid.version", Name: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(resource)
			// Should return default when version is invalid
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle missing migration for specific version", func() {
			// Create migrations map with a gap
			gappedMigrations := migrate.Migrations{
				// Missing "0.0.0" -> "1.0.0" migration
				"1.0.0": migrations["1.0.0"], // Keep V1 to V2 migration
			}

			migrator := migrate.Migrator(migrate.MigratorConfig[ResourceV0, ResourceV2]{
				Name:       "resource",
				Migrations: gappedMigrations,
				Default:    defaultV2,
			})

			resource := ResourceV0{Version: "0.0.0", Name: "foo"}
			result := migrator(resource)
			// Should return default when no migration path exists
			Expect(result).To(Equal(defaultV2))
		})
	})
})
