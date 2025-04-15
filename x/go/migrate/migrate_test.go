package migrate_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/version"
)

// Test types to simulate migrations
type EntityV0 struct {
	Version version.Semantic
	Name    string
}

func (e EntityV0) GetVersion() version.Semantic { return e.Version }

type EntityV1 struct {
	Version version.Semantic
	Title   string
}

func (e EntityV1) GetVersion() version.Semantic { return e.Version }

type EntityV2 struct {
	Version     version.Semantic
	Title       string
	Description string
}

func (e EntityV2) GetVersion() version.Semantic { return e.Version }

var _ = Describe("Migrate", func() {
	Describe("Migrator", func() {
		var (
			migrations migrate.Migrations
			defaultV2  EntityV2
		)

		BeforeEach(func() {
			// Create V0 to V1 migration
			migrateV0toV1 := migrate.CreateMigration(migrate.MigrationConfig[EntityV0, EntityV1]{
				Name: "entity",
				Migrate: func(ctx migrate.Context, v0 EntityV0) (EntityV1, error) {
					return EntityV1{
						Version: "1.0.0",
						Title:   v0.Name,
					}, nil
				},
			})

			// Create V1 to V2 migration
			migrateV1toV2 := migrate.CreateMigration(migrate.MigrationConfig[EntityV1, EntityV2]{
				Name: "entity",
				Migrate: func(ctx migrate.Context, v1 EntityV1) (EntityV2, error) {
					return EntityV2{
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

			defaultV2 = EntityV2{
				Version:     "2.0.0",
				Title:       "",
				Description: "",
			}
		})

		It("should migrate an entity from v0 to v2", func() {
			entity := EntityV0{Version: "0.0.0", Name: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV0, EntityV2]{
				Name:       "entity",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(entity)
			Expect(result).To(Equal(EntityV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "",
			}))
		})

		It("should migrate an entity from v1 to v2", func() {
			entity := EntityV1{Version: "1.0.0", Title: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV1, EntityV2]{
				Name:       "entity",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(entity)
			Expect(result).To(Equal(EntityV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "",
			}))
		})

		It("should not migrate an entity that is already at v2", func() {
			entity := EntityV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "bar",
			}
			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV2, EntityV2]{
				Name:       "entity",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(entity)
			Expect(result).To(Equal(entity))
		})

		It("should return default when migration fails", func() {
			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV0, EntityV2]{
				Name:       "entity",
				Migrations: migrate.Migrations{},
				Default:    defaultV2,
			})

			entity := EntityV0{Version: "0.0.0", Name: "foo"}
			result := migrator(entity)
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle empty version with default version", func() {
			entity := EntityV0{Version: "", Name: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV0, EntityV2]{
				Name:           "entity",
				Migrations:     migrations,
				Default:        defaultV2,
				DefaultVersion: "0.0.0",
			})

			result := migrator(entity)
			Expect(result).To(Equal(EntityV2{
				Version:     "2.0.0",
				Title:       "foo",
				Description: "",
			}))
		})

		It("should handle migration errors gracefully", func() {
			// Create a migration that always fails
			failingMigration := migrate.CreateMigration(migrate.MigrationConfig[EntityV0, EntityV1]{
				Name: "entity",
				Migrate: func(ctx migrate.Context, v0 EntityV0) (EntityV1, error) {
					return EntityV1{}, fmt.Errorf("intentional migration failure")
				},
			})

			failingMigrations := migrate.Migrations{
				"0.0.0": failingMigration,
			}

			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV0, EntityV2]{
				Name:       "entity",
				Migrations: failingMigrations,
				Default:    defaultV2,
			})

			entity := EntityV0{Version: "0.0.0", Name: "foo"}
			result := migrator(entity)
			// Should return default when migration fails
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle invalid version formats", func() {
			entity := EntityV0{Version: "invalid.version", Name: "foo"}
			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV0, EntityV2]{
				Name:       "entity",
				Migrations: migrations,
				Default:    defaultV2,
			})

			result := migrator(entity)
			// Should return default when version is invalid
			Expect(result).To(Equal(defaultV2))
		})

		It("should handle missing migration for specific version", func() {
			// Create migrations map with a gap
			gappedMigrations := migrate.Migrations{
				// Missing "0.0.0" -> "1.0.0" migration
				"1.0.0": migrations["1.0.0"], // Keep V1 to V2 migration
			}

			migrator := migrate.Migrator(migrate.MigratorConfig[EntityV0, EntityV2]{
				Name:       "entity",
				Migrations: gappedMigrations,
				Default:    defaultV2,
			})

			entity := EntityV0{Version: "0.0.0", Name: "foo"}
			result := migrator(entity)
			// Should return default when no migration path exists
			Expect(result).To(Equal(defaultV2))
		})
	})
})
