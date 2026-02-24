// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package snapshot_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/snapshot"
	. "github.com/synnaxlabs/x/testutil"
)

func writeOracleFile(dir, name, content string) {
	Expect(os.MkdirAll(dir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)).To(Succeed())
}

var _ = Describe("Snapshot", func() {
	var (
		schemasDir   string
		snapshotsDir string
	)

	BeforeEach(func() {
		tmp := GinkgoT().TempDir()
		schemasDir = filepath.Join(tmp, "schemas")
		snapshotsDir = filepath.Join(tmp, "schemas", ".snapshots")
		Expect(os.MkdirAll(schemasDir, 0755)).To(Succeed())
	})

	Describe("LatestVersion", func() {
		It("Should return 0 when snapshots dir does not exist", func() {
			v := MustSucceed(snapshot.LatestVersion(
				filepath.Join(GinkgoT().TempDir(), "nonexistent"),
			))
			Expect(v).To(Equal(0))
		})

		It("Should return 0 when snapshots dir is empty", func() {
			Expect(os.MkdirAll(snapshotsDir, 0755)).To(Succeed())
			v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
			Expect(v).To(Equal(0))
		})

		It("Should return 1 when only v1 exists", func() {
			Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v1"), 0755)).To(Succeed())
			v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
			Expect(v).To(Equal(1))
		})

		It("Should return the highest version when multiple exist", func() {
			Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v1"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v3"), 0755)).To(Succeed())
			v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
			Expect(v).To(Equal(3))
		})

		It("Should ignore non-version directories", func() {
			Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v2"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(snapshotsDir, "notes"), 0755)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(snapshotsDir, ".DS_Store"), []byte{}, 0644,
			)).To(Succeed())
			v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
			Expect(v).To(Equal(2))
		})
	})

	Describe("Create", func() {
		It("Should copy all .oracle files into v<N> subdirectory", func() {
			writeOracleFile(schemasDir, "foo.oracle", "type Foo struct {}")
			writeOracleFile(schemasDir, "bar.oracle", "type Bar struct {}")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			Expect(filepath.Join(snapshotsDir, "v1", "foo.oracle")).To(BeAnExistingFile())
			Expect(filepath.Join(snapshotsDir, "v1", "bar.oracle")).To(BeAnExistingFile())
		})

		It("Should create the snapshots directory if it does not exist", func() {
			writeOracleFile(schemasDir, "foo.oracle", "content")
			newSnapDir := filepath.Join(GinkgoT().TempDir(), "deep", "nested", ".snapshots")
			Expect(snapshot.Create(schemasDir, newSnapDir, 1)).To(Succeed())
			Expect(filepath.Join(newSnapDir, "v1", "foo.oracle")).To(BeAnExistingFile())
		})

		It("Should preserve file contents byte-for-byte", func() {
			content := "Key = uuid\n\nSchematic struct {\n\tkey Key { @key }\n}\n"
			writeOracleFile(schemasDir, "schematic.oracle", content)
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			data := MustSucceed(os.ReadFile(
				filepath.Join(snapshotsDir, "v1", "schematic.oracle"),
			))
			Expect(string(data)).To(Equal(content))
		})

		It("Should only copy .oracle files", func() {
			writeOracleFile(schemasDir, "foo.oracle", "oracle content")
			Expect(os.WriteFile(
				filepath.Join(schemasDir, "readme.md"), []byte("markdown"), 0644,
			)).To(Succeed())
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			Expect(filepath.Join(snapshotsDir, "v1", "foo.oracle")).To(BeAnExistingFile())
			Expect(filepath.Join(snapshotsDir, "v1", "readme.md")).ToNot(BeAnExistingFile())
		})

		It("Should handle an empty schemas directory without error", func() {
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			Expect(filepath.Join(snapshotsDir, "v1")).To(BeADirectory())
		})
	})

	Describe("Check", func() {
		It("Should return nil when schemas match the latest snapshot", func() {
			writeOracleFile(schemasDir, "foo.oracle", "content A")
			writeOracleFile(schemasDir, "bar.oracle", "content B")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			Expect(snapshot.Check(schemasDir, snapshotsDir)).To(Succeed())
		})

		It("Should return error when no snapshots exist", func() {
			writeOracleFile(schemasDir, "foo.oracle", "content")
			err := snapshot.Check(schemasDir, snapshotsDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no snapshots found"))
		})

		It("Should return error when content differs", func() {
			writeOracleFile(schemasDir, "foo.oracle", "original")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			writeOracleFile(schemasDir, "foo.oracle", "modified")
			err := snapshot.Check(schemasDir, snapshotsDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("modified: foo.oracle"))
		})

		It("Should return error when a new .oracle file is added", func() {
			writeOracleFile(schemasDir, "foo.oracle", "content")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			writeOracleFile(schemasDir, "new.oracle", "new content")
			err := snapshot.Check(schemasDir, snapshotsDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("added: new.oracle"))
		})

		It("Should return error when an .oracle file is removed", func() {
			writeOracleFile(schemasDir, "foo.oracle", "content")
			writeOracleFile(schemasDir, "bar.oracle", "content")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			Expect(os.Remove(filepath.Join(schemasDir, "bar.oracle"))).To(Succeed())
			err := snapshot.Check(schemasDir, snapshotsDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("removed: bar.oracle"))
		})

		It("Should compare against the latest snapshot, not earlier ones", func() {
			writeOracleFile(schemasDir, "foo.oracle", "v1 content")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			writeOracleFile(schemasDir, "foo.oracle", "v2 content")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 2)).To(Succeed())
			Expect(snapshot.Check(schemasDir, snapshotsDir)).To(Succeed())
		})
	})

	Describe("Create then Check", func() {
		It("Should pass check immediately after creating a snapshot", func() {
			writeOracleFile(schemasDir, "a.oracle", "aaa")
			writeOracleFile(schemasDir, "b.oracle", "bbb")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			Expect(snapshot.Check(schemasDir, snapshotsDir)).To(Succeed())
		})

		It("Should fail check after modifying a schema post-snapshot", func() {
			writeOracleFile(schemasDir, "a.oracle", "original")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			writeOracleFile(schemasDir, "a.oracle", "changed")
			Expect(snapshot.Check(schemasDir, snapshotsDir)).To(HaveOccurred())
		})

		It("Should pass check after creating a new snapshot post-modification", func() {
			writeOracleFile(schemasDir, "a.oracle", "original")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())
			writeOracleFile(schemasDir, "a.oracle", "changed")
			Expect(snapshot.Create(schemasDir, snapshotsDir, 2)).To(Succeed())
			Expect(snapshot.Check(schemasDir, snapshotsDir)).To(Succeed())
		})
	})
})
