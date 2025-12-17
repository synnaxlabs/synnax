// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Gorp", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("GorpRunner", func() {
		Describe("Basic migration execution", func() {
			It("Should run a single migration successfully", func() {
				executed := false
				runner := gorp.Migrator{
					Key: "test_migration_version",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = true
								return nil
							},
						},
					},
				}
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(BeTrue())
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{1}))
			})
			It("Should run multiple migrations in order", func() {
				var executionOrder []int
				runner := gorp.Migrator{
					Key: "test_migration_version_2",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionOrder = append(executionOrder, 1)
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionOrder = append(executionOrder, 2)
								return nil
							},
						},
						{
							Name: "third_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionOrder = append(executionOrder, 3)
								return nil
							},
						},
					},
				}
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionOrder).To(Equal([]int{1, 2, 3}))
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_2")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{3}))
			})

			It("Should not run migrations that are already completed", func() {
				// Run first migration
				executionCount := 0
				runner := gorp.Migrator{
					Key: "test_migration_version_3",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionCount++
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(1))

				// Run again - should not execute
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(1))
			})

			It("Should only run new migrations after partial completion", func() {
				// First run with 2 migrations
				var executed []string
				runner := gorp.Migrator{
					Key: "test_migration_version_4",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "first")
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "second")
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "second"}))

				// Second run with additional migration
				runner.Migrations = append(runner.Migrations, gorp.MigrationSpec{
					Name: "third_migration",
					Migrate: func(context.Context, gorp.Tx) error {
						executed = append(executed, "third")
						return nil
					},
				})

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "second", "third"}))

				// Verify final version
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_4")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{3}))
			})

			It("Should handle empty migration list", func() {
				runner := gorp.Migrator{
					Key:        "test_migration_version_5",
					Migrations: []gorp.MigrationSpec{},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())

				// No version key should be set
				Expect(db.Get(ctx, []byte("test_migration_version_5"))).Error().
					To(MatchError(query.NotFound))
			})
		})

		Describe("Error handling", func() {
			It("Should return error when migration fails", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_6",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "failing_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								return errors.New("test error")
							},
						},
					},
				}

				err := runner.Run(ctx, db)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("migration 1 (failing_migration) failed"))
				Expect(err.Error()).To(ContainSubstring("test error"))

				// Version should not be updated
				Expect(db.Get(ctx, []byte("test_migration_version_6"))).Error().
					To(MatchError(query.NotFound))
			})

			It("Should fail when migration count exceeds 255", func() {
				// Create 256 migrations
				migrations := make([]gorp.MigrationSpec, 256)
				for i := range migrations {
					migrations[i] = gorp.MigrationSpec{
						Name:    "migration",
						Migrate: func(context.Context, gorp.Tx) error { return nil },
					}
				}

				runner := gorp.Migrator{
					Key:        "test_migration_version_7",
					Migrations: migrations,
				}

				Expect(runner.Run(ctx, db)).Error().
					To(MatchError(gorp.ErrMigrationCountExceeded))
			})

			It("Should stop at first failing migration", func() {
				var executed []string
				runner := gorp.Migrator{
					Key: "test_migration_version_8",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "first")
								return nil
							},
						},
						{
							Name: "failing_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "second")
								return errors.New("failure")
							},
						},
						{
							Name: "third_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "third")
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).Error().
					To(MatchError(ContainSubstring("migration 2 (failing_migration) failed")))
				// First migration succeeds, second fails, third never runs
				Expect(executed).To(Equal([]string{"first", "second"}))

				// Version should be undefined because the transaction was rolled back
				Expect(db.Get(ctx, []byte("test_migration_version_8"))).Error().
					To(MatchError(query.NotFound))
			})
		})

		Describe("Force flag", func() {
			It("Should rerun all migrations when Force is true", func() {
				// First run
				executionCount := 0
				runner := gorp.Migrator{
					Key: "test_migration_version_9",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionCount++
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionCount++
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(2))

				// Second run with Force=true
				runner.Force = true
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(4))

				// Version should be updated to 2
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_9")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{2}))
			})

			It("Should run all migrations with Force even if some completed", func() {
				// Run first migration normally
				var executed []string
				runner := gorp.Migrator{
					Key: "test_migration_version_10",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "first")
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first"}))

				// Add second migration and run with Force
				runner.Migrations = append(runner.Migrations, gorp.MigrationSpec{
					Name: "second_migration",
					Migrate: func(context.Context, gorp.Tx) error {
						executed = append(executed, "second")
						return nil
					},
				})
				runner.Force = true

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "first", "second"}))
			})
		})

		Describe("Version tracking", func() {
			It("Should increment version after each successful migration", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_12",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								// Version should be undefined before the migration
								Expect(tx.Get(ctx, []byte("test_migration_version_12"))).
									Error().To(MatchError(query.NotFound))
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								// Verify version is 1 after first migration
								version, closer := MustSucceed2(
									tx.Get(ctx, []byte("test_migration_version_12")),
								)
								Expect(closer.Close()).To(Succeed())
								Expect(version).To(Equal([]byte{1}))
								return nil
							},
						},
						{
							Name: "third_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								// Verify version is 2 after second migration
								version, closer := MustSucceed2(
									tx.Get(ctx, []byte("test_migration_version_12")),
								)
								Expect(closer.Close()).To(Succeed())
								Expect(version).To(Equal([]byte{2}))
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())

				// Verify final version is 3
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_12")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{3}))
			})
		})

		Describe("Transaction behavior", func() {
			It("Should rollback all changes when a migration fails", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_13",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								// Write some data
								return tx.Set(ctx, []byte("test_key"), []byte("test_value"))
							},
						},
						{
							Name: "failing_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								return errors.New("failure")
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(HaveOccurred())

				// Data from first migration should be rolled back
				Expect(db.Get(ctx, []byte("test_key"))).Error().
					To(MatchError(query.NotFound))

				// Version should not be set
				Expect(db.Get(ctx, []byte("test_migration_version_13"))).Error().
					To(MatchError(query.NotFound))
			})

			It("Should commit all changes when all migrations succeed", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_14",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								return tx.Set(ctx, []byte("key1"), []byte("value1"))
							},
						},
						{
							Name: "second_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								return tx.Set(ctx, []byte("key2"), []byte("value2"))
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())

				// All data should be committed
				value1, closer1 := MustSucceed2(
					db.Get(ctx, []byte("key1")),
				)
				Expect(closer1.Close()).To(Succeed())
				Expect(value1).To(Equal([]byte("value1")))

				value2, closer2 := MustSucceed2(
					db.Get(ctx, []byte("key2")),
				)
				Expect(closer2.Close()).To(Succeed())
				Expect(value2).To(Equal([]byte("value2")))

				version, closer3 := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_14")),
				)
				Expect(closer3.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{2}))
			})
		})
	})
})
