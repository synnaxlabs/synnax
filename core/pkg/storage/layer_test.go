// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("storage", func() {
	Describe("Open", func() {
		var (
			tempDir string
			cfg     storage.Config
		)
		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "synnax-test")
			Expect(err).ToNot(HaveOccurred())
			cfg = storage.Config{Dirname: filepath.Join(tempDir, "storage")}
			ShouldNotLeakGoroutines()
		})
		AfterEach(func() { Expect(os.RemoveAll(tempDir)).ToNot(HaveOccurred()) })
		Describe("Acquiring a lock", func() {
			It("Should return an error if the lock is already acquired", func() {
				store, err := storage.Open(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				_, err = storage.Open(ctx, cfg)
				Expect(err).To(HaveOccurred())
				Expect(store.Close()).To(Succeed())
			})
		})
		Describe("Permissions", func() {
			// These two tests are failing on Windows because VFS cannot create a
			// directory on Windows with custom permissions – all directories are created
			// with 777.
			if !strings.HasPrefix(runtime.GOOS, "windows") {
				Describe("Name Directory", func() {
					It("Should set the correct permissions on the storage directory", func() {
						cfg.Perm = storage.DefaultConfig.Perm
						store, err := storage.Open(ctx, cfg)
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
						Expect(os.Mkdir(p, xfs.UserR)).To(Succeed())
						cfg.Dirname = p
					})
					AfterEach(func() { Expect(os.RemoveAll(p)).To(Succeed()) })
					It("Should return an error if the directory exists but has insufficient permissions", func() {
						// use os.Stat to check the dir permissions
						cfg.Perm = xfs.UserRWX
						_, err := storage.Open(ctx, cfg)
						Expect(err).To(HaveOccurred())
					})
				})
			}
		})
		Describe("In-Memory", func() {
			It("Should open a memory backed version of storage", func() {
				cfg.InMemory = new(true)
				store, err := storage.Open(ctx, cfg)
				Expect(err).NotTo(HaveOccurred())
				Expect(store.Close()).To(Succeed())
			})
		})
	})
	Describe("ServiceConfig", func() {
		DescribeTable("Validate", func(
			spec func(cfg storage.Config) storage.Config,
			contains string,
		) {
			iCfg := storage.DefaultConfig
			iCfg.Dirname = "foo"
			err := spec(iCfg).Validate()
			if contains == "" {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(contains))
			}
		},
			Entry("Directory not set",
				func(cfg storage.Config) storage.Config {
					cfg.Dirname = ""
					*cfg.InMemory = false
					return cfg
				},
				"dirname",
			),
			Entry("Directory not set, mem-backed",
				func(cfg storage.Config) storage.Config {
					cfg.Dirname = ""
					*cfg.InMemory = true
					return cfg
				},
				"",
			),
			Entry("Invalid key-value engine",
				func(cfg storage.Config) storage.Config {
					cfg.KVEngine = 12
					return cfg
				},
				"key-value engine",
			),
			Entry("Invalid permissions",
				func(cfg storage.Config) storage.Config {
					cfg.Perm = 0
					return cfg
				},
				"permissions",
			),
		)
	})
})
