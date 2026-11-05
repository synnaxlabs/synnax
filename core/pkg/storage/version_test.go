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
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage"
	. "github.com/synnaxlabs/x/testutil"
)

// openAt opens a Layer on dir as a Core of the given version, then closes it.
func openAt(ctx context.Context, dir, version string) {
	GinkgoHelper()
	Expect(openLayer(ctx, dir, version).Close()).To(Succeed())
}

func recordedVersion(dir string) string {
	GinkgoHelper()
	return string(MustSucceed(os.ReadFile(filepath.Join(dir, "VERSION"))))
}

// entries returns the names in dir, or nothing when dir does not exist.
func entries(dir string) []string {
	GinkgoHelper()
	found, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	Expect(err).ToNot(HaveOccurred())
	names := make([]string, len(found))
	for i, e := range found {
		names[i] = e.Name()
	}
	return names
}

func newDataDir() string {
	GinkgoHelper()
	tempDir := MustSucceed(os.MkdirTemp("", "synnax-storage-version"))
	DeferCleanup(func() { Expect(os.RemoveAll(tempDir)).To(Succeed()) })
	return filepath.Join(tempDir, "data")
}

var _ = Describe("Recorded version", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It("Should record the version in a new data directory", func(ctx SpecContext) {
		openAt(ctx, dir, "0.60.0")
		Expect(recordedVersion(dir)).To(Equal("0.60.0"))
		Expect(entries(filepath.Join(dir, "backups"))).To(BeEmpty())
	})

	It("Should back up data without a recorded version", func(ctx SpecContext) {
		openAt(ctx, dir, "0.0.0")
		Expect(filepath.Join(dir, "VERSION")).ToNot(BeAnExistingFile())
		openAt(ctx, dir, "0.60.0")
		Expect(recordedVersion(dir)).To(Equal("0.60.0"))
		Expect(entries(filepath.Join(dir, "backups"))).
			To(ConsistOf(HaveSuffix("-unknown")))
	})

	DescribeTable(
		"Opening data that an older or equal version recorded",
		func(ctx SpecContext, recorded, current string, backedUp bool) {
			openAt(ctx, dir, recorded)
			openAt(ctx, dir, current)
			Expect(recordedVersion(dir)).To(Equal(current))
			backups := entries(filepath.Join(dir, "backups"))
			if backedUp {
				Expect(backups).To(ConsistOf(HaveSuffix("-" + recorded)))
			} else {
				Expect(backups).To(BeEmpty())
			}
		},
		Entry("same version", "0.60.0", "0.60.0", false),
		Entry("older patch", "0.60.0", "0.60.1", true),
		Entry("older minor", "0.59.3", "0.60.0", true),
		Entry("candidate of the release", "0.60.0-rc.1", "0.60.0", true),
		Entry("earlier candidate", "0.60.0-rc.9", "0.60.0-rc.10", true),
		Entry("earlier numeric pre-release", "0.60.0-1", "0.60.0-2", true),
	)

	DescribeTable(
		"Opening data that a newer version recorded",
		func(ctx SpecContext, recorded, current string) {
			openAt(ctx, dir, recorded)
			Expect(storage.OpenLayer(
				ctx,
				storage.LayerConfig{Dirname: dir, Version: current},
			)).Error().To(MatchError("Data was last opened by Core " + recorded +
				". Run Core " + recorded + " or newer, or restore a backup with " +
				"`synnax restore`."))
			Expect(recordedVersion(dir)).To(Equal(recorded))
			Expect(entries(filepath.Join(dir, "backups"))).To(BeEmpty())
		},
		Entry("newer patch", "0.60.1", "0.60.0"),
		Entry("newer minor", "0.61.0", "0.60.5"),
		Entry("release of the candidate", "0.60.0", "0.60.0-rc.3"),
		Entry("later candidate", "0.60.0-rc.10", "0.60.0-rc.9"),
	)

	DescribeTable(
		"Opening as a dev Core",
		func(ctx SpecContext, dev string) {
			openAt(ctx, dir, dev)
			Expect(filepath.Join(dir, "VERSION")).ToNot(BeAnExistingFile())
			openAt(ctx, dir, "0.60.1")
			openAt(ctx, dir, dev)
			Expect(recordedVersion(dir)).To(Equal("0.60.1"))
			Expect(entries(filepath.Join(dir, "backups"))).
				To(ConsistOf(HaveSuffix("-unknown")))
		},
		Entry("unstamped", "0.0.0"),
		Entry("dev build", "0.0.0-dev.0.60.1.abc1234"),
		Entry("dev build with a numeric commit", "0.0.0-dev.0123456"),
	)

	It("Should not open data with an invalid recorded version", func(ctx SpecContext) {
		Expect(os.MkdirAll(dir, 0o700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "VERSION"), []byte("garbage"), 0o600)).
			To(Succeed())
		Expect(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		)).Error().To(MatchError(ContainSubstring(`invalid version "garbage"`)))
	})

	It("Should not open when the backup fails", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		// A file in place of the backups directory makes every copy fail.
		Expect(os.WriteFile(filepath.Join(dir, "backups"), nil, 0o600)).To(Succeed())
		Expect(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		)).Error().To(MatchError(ContainSubstring("failed to back up metadata")))
		Expect(recordedVersion(dir)).To(Equal("0.59.0"))
		Expect(os.Remove(filepath.Join(dir, "backups"))).To(Succeed())
		openAt(ctx, dir, "0.60.0")
		Expect(entries(filepath.Join(dir, "backups"))).
			To(ConsistOf(HaveSuffix("-0.59.0")))
	})
})

var _ = Describe("ListBackups", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It("Should return an error for a missing data directory", func() {
		Expect(storage.ListBackups(dir)).Error().To(MatchError(fs.ErrNotExist))
	})

	It("Should return nothing when there are no backups", func() {
		Expect(os.MkdirAll(dir, 0o700)).To(Succeed())
		Expect(MustSucceed(storage.ListBackups(dir))).To(BeEmpty())
	})

	It("Should list only complete backups, oldest first", func() {
		backups := filepath.Join(dir, "backups")
		for _, name := range []string{
			"0000000002-0.60.0-rc.1",
			"0000000001-unknown",
			"0000000003-0.60.0.partial",
			"orphaned-4",
		} {
			Expect(os.MkdirAll(filepath.Join(backups, name), 0o700)).To(Succeed())
		}
		Expect(os.WriteFile(filepath.Join(backups, "0000000005-0.61.0"), nil, 0o600)).
			To(Succeed())
		Expect(MustSucceed(storage.ListBackups(dir))).To(Equal([]storage.Backup{
			{Name: "0000000001-unknown", Time: time.Unix(1, 0), Version: "unknown"},
			{
				Name:    "0000000002-0.60.0-rc.1",
				Time:    time.Unix(2, 0),
				Version: "0.60.0-rc.1",
			},
		}))
	})
})
