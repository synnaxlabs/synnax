// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/backup"
	. "github.com/synnaxlabs/x/testutil"
)

// fixtureLayout describes a Synnax-shaped data directory used in the tests below.
// Keys are paths relative to the source directory; values are the file contents.
var fixtureLayout = map[string]string{
	"LOCK":                          "lock",
	"kv/CURRENT":                    "kv-current",
	"kv/000001.log":                 "kv-log",
	"kv/MANIFEST-000002":            "kv-manifest",
	"cesium/1/meta.json":            `{"key":1}`,
	"cesium/1/index.domain":         "index-bytes",
	"cesium/1/counter.domain":       "counter-bytes",
	"cesium/1/1.domain":             "data-bytes-1",
	"cesium/1/2.domain":             "data-bytes-2",
	"cesium/2/meta.json":            `{"key":2}`,
	"cesium/2/1.domain":             "data-bytes-3",
	"cesium/counter.domain":         "root-counter",
	"cesium/some-other-config.json": "{}",
}

func writeFixture(root string) {
	for rel, content := range fixtureLayout {
		full := filepath.Join(root, rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
}

func readBackup(root string) map[string]string {
	out := map[string]string{}
	Expect(filepath.WalkDir(
		root,
		func(path string, d os.DirEntry, walkErr error) error {
			Expect(walkErr).To(Succeed())
			if d.IsDir() {
				return nil
			}
			rel := MustSucceed(filepath.Rel(root, path))
			out[filepath.ToSlash(rel)] = string(MustSucceed(os.ReadFile(path)))
			return nil
		},
	)).To(Succeed())
	return out
}

func mustExist(path string) {
	_ = MustSucceed(os.Stat(path))
}

func mustNotExist(path string) {
	_, err := os.Stat(path)
	Expect(os.IsNotExist(err)).To(BeTrue())
}

var _ = Describe("Backup", func() {
	var (
		base string
		src  string
		dst  string
	)
	BeforeEach(func() {
		base = MustSucceed(os.MkdirTemp("", "synnax-backup-"))
		src = filepath.Join(base, "src")
		dst = filepath.Join(base, "dst")
		Expect(os.MkdirAll(src, 0o755)).To(Succeed())
		writeFixture(src)
		DeferCleanup(func() {
			Expect(os.RemoveAll(base)).To(Succeed())
		})
	})

	Describe("default copy", func() {
		It("Should copy the entire data directory minus the LOCK file", func() {
			Expect(backup.Backup(backup.Config{Src: src, Dst: dst})).To(Succeed())
			result := readBackup(dst)
			expected := map[string]string{}
			for k, v := range fixtureLayout {
				if k == backup.LockFile {
					continue
				}
				expected[k] = v
			}
			Expect(result).To(Equal(expected))
		})

		It("Should preserve file contents byte-for-byte", func() {
			Expect(backup.Backup(backup.Config{Src: src, Dst: dst})).To(Succeed())
			data := MustSucceed(os.ReadFile(filepath.Join(dst, "cesium/1/1.domain")))
			Expect(string(data)).To(Equal("data-bytes-1"))
		})

		It("Should always exclude the LOCK file", func() {
			Expect(backup.Backup(backup.Config{Src: src, Dst: dst})).To(Succeed())
			mustNotExist(filepath.Join(dst, backup.LockFile))
		})
	})

	Describe("--no-data", func() {
		It("Should skip channel data files but keep configuration", func() {
			Expect(backup.Backup(backup.Config{
				Src:    src,
				Dst:    dst,
				NoData: true,
			})).To(Succeed())
			result := readBackup(dst)
			Expect(result).To(HaveKey("kv/CURRENT"))
			Expect(result).To(HaveKey("kv/000001.log"))
			Expect(result).To(HaveKey("cesium/1/meta.json"))
			Expect(result).To(HaveKey("cesium/2/meta.json"))
			Expect(result).To(HaveKey("cesium/some-other-config.json"))
			Expect(result).ToNot(HaveKey("cesium/1/1.domain"))
			Expect(result).ToNot(HaveKey("cesium/1/2.domain"))
			Expect(result).ToNot(HaveKey("cesium/1/index.domain"))
			Expect(result).ToNot(HaveKey("cesium/1/counter.domain"))
			Expect(result).ToNot(HaveKey("cesium/2/1.domain"))
			Expect(result).ToNot(HaveKey("cesium/counter.domain"))
			Expect(result).ToNot(HaveKey(backup.LockFile))
		})

		It("Should not skip .domain files outside the cesium subtree", func() {
			Expect(os.WriteFile(
				filepath.Join(src, "kv", "stray.domain"),
				[]byte("not-cesium"),
				0o644,
			)).To(Succeed())
			Expect(backup.Backup(backup.Config{
				Src:    src,
				Dst:    dst,
				NoData: true,
			})).To(Succeed())
			data := MustSucceed(os.ReadFile(filepath.Join(dst, "kv", "stray.domain")))
			Expect(string(data)).To(Equal("not-cesium"))
		})
	})

	Describe("validation", func() {
		It("Should reject an empty src", func() {
			Expect(backup.Backup(backup.Config{Dst: dst})).Error().
				To(MatchError(ContainSubstring("src is required")))
		})

		It("Should reject an empty dst", func() {
			Expect(backup.Backup(backup.Config{Src: src})).Error().
				To(MatchError(ContainSubstring("dst is required")))
		})

		It("Should reject a non-existent src", func() {
			Expect(backup.Backup(backup.Config{
				Src: filepath.Join(base, "missing"),
				Dst: dst,
			})).Error().To(MatchError(ContainSubstring("stat src")))
		})

		It("Should reject when src is not a directory", func() {
			file := filepath.Join(base, "afile")
			Expect(os.WriteFile(file, []byte("x"), 0o644)).To(Succeed())
			Expect(backup.Backup(backup.Config{Src: file, Dst: dst})).Error().
				To(MatchError(ContainSubstring("is not a directory")))
		})

		It("Should reject when src and dst are the same path", func() {
			Expect(backup.Backup(backup.Config{Src: src, Dst: src})).Error().
				To(MatchError(ContainSubstring("must not be the same path")))
		})

		It("Should reject when dst is inside src", func() {
			inside := filepath.Join(src, "inner")
			Expect(backup.Backup(backup.Config{Src: src, Dst: inside})).Error().
				To(MatchError(ContainSubstring("is inside src")))
		})

		It("Should reject when dst exists and overwrite is false", func() {
			Expect(os.MkdirAll(dst, 0o755)).To(Succeed())
			Expect(backup.Backup(backup.Config{Src: src, Dst: dst})).Error().
				To(MatchError(ContainSubstring("already exists")))
		})

		It("Should write into an existing dst when overwrite is true", func() {
			Expect(os.MkdirAll(dst, 0o755)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(dst, "preexisting.txt"),
				[]byte("keep"),
				0o644,
			)).To(Succeed())
			Expect(backup.Backup(backup.Config{
				Src:       src,
				Dst:       dst,
				Overwrite: true,
			})).To(Succeed())
			Expect(string(MustSucceed(os.ReadFile(
				filepath.Join(dst, "preexisting.txt"),
			)))).To(Equal("keep"))
			Expect(string(MustSucceed(os.ReadFile(
				filepath.Join(dst, "kv/CURRENT"),
			)))).To(Equal("kv-current"))
		})
	})

	Describe("Cmd", func() {
		AfterEach(func() {
			viper.Reset()
		})

		It("Should run via the cobra command using --data", func() {
			backup.Cmd.SetArgs([]string{"--data", src, dst})
			Expect(backup.Cmd.Execute()).To(Succeed())
			mustExist(filepath.Join(dst, "kv", "CURRENT"))
		})

		It("Should respect the --no-data flag", func() {
			backup.Cmd.SetArgs([]string{"--data", src, "--no-data", dst})
			Expect(backup.Cmd.Execute()).To(Succeed())
			mustNotExist(filepath.Join(dst, "cesium", "1", "1.domain"))
			mustExist(filepath.Join(dst, "cesium", "1", "meta.json"))
		})

		It("Should read the source from viper when --data is omitted", func() {
			viper.Set(backup.FlagData, src)
			backup.Cmd.SetArgs([]string{dst})
			Expect(backup.Cmd.Execute()).To(Succeed())
			mustExist(filepath.Join(dst, "kv", "CURRENT"))
		})
	})
})
