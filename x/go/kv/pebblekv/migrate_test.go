package pebblekv_test

import (
	. "github.com/synnaxlabs/x/testutil"
	"os"
	"path/filepath"

	pebblev1 "github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/kv/pebblekv"
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
			}))

			Expect(oldDB.Set([]byte("key"), []byte("value"), nil)).To(Succeed())

			Expect(oldDB.Close()).To(Succeed())

			requiresMigration, err := pebblekv.RequiresMigration(dbPath, vfs.Default)
			Expect(err).NotTo(HaveOccurred())
			Expect(requiresMigration).To(BeTrue())

			Expect(pebblekv.Migrate(dbPath)).To(Succeed())

			newDB := MustSucceed(pebble.Open(dbPath, &pebble.Options{
				FS:                 vfs.Default,
				FormatMajorVersion: pebble.FormatNewest,
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
			}))
			Expect(oldDB.Close()).To(Succeed())

			Expect(pebblekv.Migrate(dbPath)).To(Succeed())
		})
	})
})
