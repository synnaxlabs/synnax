// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"os"
	"path/filepath"

	pebblev1 "github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/kv/pebblekv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "pebblekv-migrate-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	Context("RequiresMigration", func() {
		It("should return false for a new database", func() {
			dbPath := filepath.Join(tempDir, "new-db")
			db, err := pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(db.Close()).To(Succeed())

			requiresMigration := MustSucceed(pebblekv.RequiresMigration(dbPath, vfs.Default))
			Expect(requiresMigration).To(BeFalse())
		})

		It("should return true for an old database", func() {
			dbPath := filepath.Join(tempDir, "old-db")

			oldDB := MustSucceed(pebblev1.Open(dbPath, &pebblev1.Options{
				FormatMajorVersion: pebblev1.FormatDefault, // Use oldest supported format
				Logger:             pebblekv.NewNoopLogger(),
			}))
			Expect(oldDB.Close()).To(Succeed())

			requiresMigration := MustSucceed(pebblekv.RequiresMigration(dbPath, vfs.Default))
			Expect(requiresMigration).To(BeTrue())
		})

		It("should return false for non-existent database", func() {
			nonExistentPath := filepath.Join(tempDir, "non-existent")

			required := MustSucceed(pebblekv.RequiresMigration(nonExistentPath, vfs.Default))
			Expect(required).To(BeFalse())
		})

	})

	Context("Migrate", func() {
		It("should successfully migrate an old database", func() {
			dbPath := filepath.Join(tempDir, "migrate-db")

			oldDB := MustSucceed(pebblev1.Open(dbPath, &pebblev1.Options{
				FormatMajorVersion: pebblev1.FormatDefault, // Use oldest supported format
				Logger:             pebblekv.NewNoopLogger(),
			}))

			Expect(oldDB.Set([]byte("key"), []byte("value"), nil)).To(Succeed())

			Expect(oldDB.Close()).To(Succeed())

			requiresMigration, err := pebblekv.RequiresMigration(dbPath, vfs.Default)
			Expect(err).NotTo(HaveOccurred())
			Expect(requiresMigration).To(BeTrue())

			Expect(pebblekv.Migrate(dbPath, alamos.Instrumentation{})).To(Succeed())

			newDB := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			}))

			val, closer := MustSucceed2(newDB.Get([]byte("key")))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(val)).To(Equal("value"))
			Expect(closer.Close()).To(Succeed())
			Expect(newDB.Close()).To(Succeed())

			requiresMigration = MustSucceed(pebblekv.RequiresMigration(dbPath, vfs.Default))
			Expect(err).NotTo(HaveOccurred())
			Expect(requiresMigration).To(BeFalse())

		})

		It("should handle migration of empty database", func() {
			dbPath := filepath.Join(tempDir, "empty-migrate-db")

			oldDB := MustSucceed(pebblev1.Open(dbPath, &pebblev1.Options{
				FormatMajorVersion: pebblev1.FormatDefault,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			Expect(oldDB.Close()).To(Succeed())

			Expect(pebblekv.Migrate(dbPath, alamos.Instrumentation{})).To(Succeed())
		})

		It("should handle database already at v2 newest format", func() {
			dbPath := filepath.Join(tempDir, "v2-newest-db")

			// Create a database with v2's newest format
			db := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			Expect(db.Set([]byte("test-key"), []byte("test-value"), nil)).To(Succeed())
			Expect(db.Close()).To(Succeed())

			// Should not require migration
			requiresMigration := MustSucceed(pebblekv.RequiresMigration(dbPath, vfs.Default))
			Expect(requiresMigration).To(BeFalse())

			// Migration should still succeed (no-op)
			Expect(pebblekv.Migrate(dbPath, alamos.Instrumentation{})).To(Succeed())

			// Verify data is still accessible
			verifyDB := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			val, closer := MustSucceed2(verifyDB.Get([]byte("test-key")))
			Expect(string(val)).To(Equal("test-value"))
			Expect(closer.Close()).To(Succeed())
			Expect(verifyDB.Close()).To(Succeed())
		})

		It("should handle database with v2 supported format but not newest", func() {
			dbPath := filepath.Join(tempDir, "v2-supported-db")

			// Create a database with an older v2 supported format We'll use
			// FormatMinSupported to create a database with the minimum v2 format
			db := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatMinSupported,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			Expect(db.Set([]byte("data-key"), []byte("data-value"), nil)).To(Succeed())
			Expect(db.Close()).To(Succeed())

			// Migration should succeed and potentially upgrade format
			Expect(pebblekv.Migrate(dbPath, alamos.Instrumentation{})).To(Succeed())

			// Verify data is still accessible and database works with newest format
			verifyDB := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			val, closer := MustSucceed2(verifyDB.Get([]byte("data-key")))
			Expect(string(val)).To(Equal("data-value"))
			Expect(closer.Close()).To(Succeed())
			Expect(verifyDB.Close()).To(Succeed())
		})

		It("should handle database with v1 newest format", func() {
			dbPath := filepath.Join(tempDir, "v1-newest-db")

			// Create a database with v1's newest format
			oldDB := MustSucceed(pebblev1.Open(dbPath, &pebblev1.Options{
				FormatMajorVersion: pebblev1.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			Expect(oldDB.Set([]byte("v1-key"), []byte("v1-value"), nil)).To(Succeed())
			Expect(oldDB.Close()).To(Succeed())

			// Should require migration (v1 newest to v2 newest)
			requiresMigration := MustSucceed(pebblekv.RequiresMigration(dbPath, vfs.Default))
			Expect(requiresMigration).To(BeTrue())

			// Migration should succeed
			Expect(pebblekv.Migrate(dbPath, alamos.Instrumentation{})).To(Succeed())

			// Verify data is accessible with v2
			verifyDB := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
				Logger:             pebblekv.NewNoopLogger(),
			}))
			val, closer := MustSucceed2(verifyDB.Get([]byte("v1-key")))
			Expect(string(val)).To(Equal("v1-value"))
			Expect(closer.Close()).To(Succeed())
			Expect(verifyDB.Close()).To(Succeed())

			// Should no longer require migration
			requiresMigration = MustSucceed(pebblekv.RequiresMigration(dbPath, vfs.Default))
			Expect(requiresMigration).To(BeFalse())
		})

		It("should verify format versions during migration", func() {
			dbPath := filepath.Join(tempDir, "format-verification-db")

			// Create very old format database
			oldDB := MustSucceed(pebblev1.Open(dbPath, &pebblev1.Options{
				FormatMajorVersion: pebblev1.FormatMostCompatible, // Very old format
				Logger:             pebblekv.NewNoopLogger(),
			}))
			Expect(oldDB.Close()).To(Succeed())

			// Verify initial format is old
			dbDesc := MustSucceed(pebble.Peek(dbPath, vfs.Default))
			initialFormat := dbDesc.FormatMajorVersion
			Expect(uint64(initialFormat) < uint64(pebble.FormatMinSupported)).To(BeTrue())

			// Migrate
			Expect(pebblekv.Migrate(dbPath, alamos.Instrumentation{})).To(Succeed())

			// Verify final format is newest
			dbDesc = MustSucceed(pebble.Peek(dbPath, vfs.Default))
			Expect(dbDesc.FormatMajorVersion).To(Equal(pebble.FormatNewest))
		})
	})
})
