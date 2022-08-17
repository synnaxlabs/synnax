package storage_test

import (
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/fsutil"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
)

var _ = Describe("StorageKey", func() {
	Describe("Open", func() {
		var (
			tempDir string
			cfg     storage.Config
		)
		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "delta-test")
			Expect(err).ToNot(HaveOccurred())
			cfg = storage.Config{Dirname: filepath.Join(tempDir, "storage")}
		})
		AfterEach(func() { Expect(os.RemoveAll(tempDir)).ToNot(HaveOccurred()) })
		Describe("Acquiring a lock", func() {
			It("Should return an error if the lock is already acquired", func() {
				store, err := storage.Open(cfg)
				Expect(err).NotTo(HaveOccurred())
				_, err = storage.Open(cfg)
				Expect(err).To(HaveOccurred())
				Expect(store.Close()).To(Succeed())
			})
		})
		Describe("Permissions", func() {
			Describe("New Directory", func() {
				It("Should set the correct permissions on the storage directory", func() {
					cfg.Perm = storage.DefaultConfig().Perm
					store, err := storage.Open(cfg)
					Expect(err).NotTo(HaveOccurred())
					stat, err := os.Stat(cfg.Dirname)
					Expect(err).ToNot(HaveOccurred())
					Expect(stat.Mode().Perm()).To(Equal(cfg.Perm))
					Expect(store.Close()).To(Succeed())
				})
			})
			Describe("Existing Directory", func() {
				var p string
				BeforeEach(func() {
					p = filepath.Join(tempDir, "x-storage")
					Expect(os.Mkdir(p, fsutil.OS_USER_R)).To(Succeed())
					cfg.Dirname = p
				})
				AfterEach(func() { Expect(os.RemoveAll(p)).To(Succeed()) })
				It("Should return an error if the directory exists but has insufficient permissions", func() {
					// use os.Stat to check the dir permissions
					cfg.Perm = fsutil.OS_USER_RWX
					_, err := storage.Open(cfg)
					Expect(err).To(HaveOccurred())
				})
			})
		})
		Describe("Membacked", func() {
			It("Should open a memory backed version of storage", func() {
				cfg.MemBacked = true
				store, err := storage.Open(cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(store.Close()).To(Succeed())
			})
		})
	})
	Describe("Config", func() {
		DescribeTable("Validate", func(
			spec func(cfg storage.Config) storage.Config,
			expectedErr error,
		) {
			iCfg := storage.DefaultConfig()
			iCfg.Dirname = "foo"
			err := spec(iCfg).Validate()
			if expectedErr == nil {
				Expect(err).ToNot(HaveOccurred())
			} else {
				if !errors.Is(err, expectedErr) {
					Fail(err.Error())
				}
			}
		},
			Entry("Directory not set",
				func(cfg storage.Config) storage.Config {
					cfg.Dirname = ""
					cfg.MemBacked = false
					return cfg
				},
				errors.Newf("[storage] - dirname must be set"),
			),
			Entry("Directory not set, mem-backed",
				func(cfg storage.Config) storage.Config {
					cfg.Dirname = ""
					cfg.MemBacked = true
					return cfg
				},
				nil,
			),
			Entry("Invalid key-value engine",
				func(cfg storage.Config) storage.Config {
					cfg.KVEngine = 12
					return cfg
				},
				errors.New("[storage] - invalid key-value engine"),
			),
			Entry("Invalid time-series engine",
				func(cfg storage.Config) storage.Config {
					cfg.TSEngine = 12
					return cfg
				},
				errors.New("[storage] - invalid time-series engine"),
			),
			Entry("Invalid permissions",
				func(cfg storage.Config) storage.Config {
					cfg.Perm = 0
					return cfg
				},
				errors.New("[storage] - insufficient permission bits configured"),
			),
		)
	})
})
