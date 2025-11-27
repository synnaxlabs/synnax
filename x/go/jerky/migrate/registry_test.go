// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/jerky/migrate"
)

var _ = Describe("Registry", func() {
	Describe("GetMigration", func() {
		It("should find migration for valid version range", func() {
			registry := &migrate.Registry{
				TypeName:       "User",
				CurrentVersion: 3,
				Migrations: []migrate.Migration{
					{FromVersion: 0, ToVersion: 1, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
					{FromVersion: 1, ToVersion: 2, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
					{FromVersion: 2, ToVersion: 3, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
				},
			}

			migration := registry.GetMigration(1, 2)
			Expect(migration).ToNot(BeNil())
			Expect(migration.FromVersion).To(Equal(1))
			Expect(migration.ToVersion).To(Equal(2))
		})

		It("should return nil for non-existent migration", func() {
			registry := &migrate.Registry{
				TypeName:       "User",
				CurrentVersion: 2,
				Migrations: []migrate.Migration{
					{FromVersion: 0, ToVersion: 1, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
					{FromVersion: 1, ToVersion: 2, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
				},
			}

			migration := registry.GetMigration(5, 6)
			Expect(migration).To(BeNil())
		})
	})

	Describe("Migration chain", func() {
		It("should support sequential migrations", func() {
			callOrder := []int{}

			registry := &migrate.Registry{
				TypeName:       "User",
				CurrentVersion: 3,
				Migrations: []migrate.Migration{
					{
						FromVersion: 0,
						ToVersion:   1,
						Migrate: func(data []byte) ([]byte, error) {
							callOrder = append(callOrder, 1)
							return append(data, 'a'), nil
						},
					},
					{
						FromVersion: 1,
						ToVersion:   2,
						Migrate: func(data []byte) ([]byte, error) {
							callOrder = append(callOrder, 2)
							return append(data, 'b'), nil
						},
					},
					{
						FromVersion: 2,
						ToVersion:   3,
						Migrate: func(data []byte) ([]byte, error) {
							callOrder = append(callOrder, 3)
							return append(data, 'c'), nil
						},
					},
				},
			}

			// Simulate running migrations from version 0 to 3
			data := []byte("start")
			for v := 0; v < registry.CurrentVersion; v++ {
				migration := registry.GetMigration(v, v+1)
				Expect(migration).ToNot(BeNil())
				var err error
				data, err = migration.Migrate(data)
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(callOrder).To(Equal([]int{1, 2, 3}))
			Expect(string(data)).To(Equal("startabc"))
		})
	})

	Describe("MigrateToLatest", func() {
		var registry *migrate.Registry
		var callOrder []int

		BeforeEach(func() {
			callOrder = []int{}
			registry = &migrate.Registry{
				TypeName:       "User",
				CurrentVersion: 3,
				Migrations: []migrate.Migration{
					{
						FromVersion: 0,
						ToVersion:   1,
						Migrate: func(data []byte) ([]byte, error) {
							callOrder = append(callOrder, 1)
							return append(data, 'a'), nil
						},
					},
					{
						FromVersion: 1,
						ToVersion:   2,
						Migrate: func(data []byte) ([]byte, error) {
							callOrder = append(callOrder, 2)
							return append(data, 'b'), nil
						},
					},
					{
						FromVersion: 2,
						ToVersion:   3,
						Migrate: func(data []byte) ([]byte, error) {
							callOrder = append(callOrder, 3)
							return append(data, 'c'), nil
						},
					},
				},
			}
		})

		It("should migrate from version 0 to latest", func() {
			data := []byte("start")
			result, err := registry.MigrateToLatest(data, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(callOrder).To(Equal([]int{1, 2, 3}))
			Expect(string(result)).To(Equal("startabc"))
		})

		It("should migrate from version 1 to latest", func() {
			data := []byte("start")
			result, err := registry.MigrateToLatest(data, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(callOrder).To(Equal([]int{2, 3}))
			Expect(string(result)).To(Equal("startbc"))
		})

		It("should return data unchanged when already at latest version", func() {
			data := []byte("start")
			result, err := registry.MigrateToLatest(data, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(callOrder).To(BeEmpty())
			Expect(string(result)).To(Equal("start"))
		})

		It("should return error when migration is missing", func() {
			// Create registry with a gap in migrations
			gappedRegistry := &migrate.Registry{
				TypeName:       "User",
				CurrentVersion: 3,
				Migrations: []migrate.Migration{
					{FromVersion: 0, ToVersion: 1, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
					// Missing 1 -> 2
					{FromVersion: 2, ToVersion: 3, Migrate: func(data []byte) ([]byte, error) { return data, nil }},
				},
			}

			data := []byte("start")
			_, err := gappedRegistry.MigrateToLatest(data, 0)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no migration found"))
		})
	})
})
