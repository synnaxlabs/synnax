// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// tempDirPrefix is the prefix of every directory osFS creates in os.TempDir.
func tempDirPrefix() string {
	return fmt.Sprintf("testdata-p%d-", GinkgoParallelProcess())
}

// matchingTempDirs returns the basenames of every entry in os.TempDir that
// starts with the given prefix. Used to observe whether osFS leaves a
// directory behind after its enclosing scope exits.
func matchingTempDirs(prefix string) []string {
	GinkgoHelper()
	entries := MustSucceed(os.ReadDir(os.TempDir()))
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			out = append(out, e.Name())
		}
	}
	return out
}

var _ = Describe("FS Testutil", func() {
	Describe("memFS", func() {
		It("Should return a usable in-memory FS", func() {
			fs := FileSystems["memFS"]()
			Expect(fs).ToNot(BeNil())
			f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
		})

		It(
			"Should return a fresh FS on each call so callers cannot leak state across tests",
			func() {
				a := FileSystems["memFS"]()
				b := FileSystems["memFS"]()
				fa := MustSucceed(a.Open("only-in-a.bin", os.O_CREATE|os.O_RDWR))
				DeferClose(fa)
				Expect(MustSucceed(b.Exists("only-in-a.bin"))).To(BeFalse())
			},
		)

		It("Should leave nothing on disk", func() {
			before := matchingTempDirs(tempDirPrefix())
			FileSystems["memFS"]()
			Expect(matchingTempDirs(tempDirPrefix())).To(Equal(before))
		})
	})

	Describe("osFS", func() {
		It("Should return a usable on-disk FS", func() {
			fs := FileSystems["osFS"]()
			Expect(fs).ToNot(BeNil())
			f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
		})

		It("Should return a fresh FS rooted at its own tempdir on each call", func() {
			a := FileSystems["osFS"]()
			b := FileSystems["osFS"]()
			fa := MustSucceed(a.Open("only-in-a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(fa)
			Expect(MustSucceed(b.Exists("only-in-a.bin"))).To(BeFalse())
		})

		It("Should back the FS with a real directory under os.TempDir", func() {
			before := matchingTempDirs(tempDirPrefix())
			FileSystems["osFS"]()
			after := matchingTempDirs(tempDirPrefix())
			Expect(after).To(HaveLen(len(before) + 1))
		})

		Describe("Cleanup", Ordered, func() {
			var (
				priorDirs []string
				createdAt string
			)
			BeforeAll(func() {
				ShouldNotLeakGoroutines()
				priorDirs = matchingTempDirs(tempDirPrefix())
			})
			It("Creates the tempdir while the spec is running", func() {
				FileSystems["osFS"]()
				current := matchingTempDirs(tempDirPrefix())
				Expect(current).To(HaveLen(len(priorDirs) + 1))
				for _, name := range current {
					if !slices.Contains(priorDirs, name) {
						createdAt = filepath.Join(os.TempDir(), name)
						break
					}
				}
				Expect(createdAt).ToNot(BeEmpty())
			})
			It("Removes the tempdir before the next spec runs", func() {
				_, err := os.Stat(createdAt)
				Expect(os.IsNotExist(err)).To(BeTrue())
				Expect(matchingTempDirs(tempDirPrefix())).To(Equal(priorDirs))
			})
		})
	})

	Describe("FileSystems", func() {
		It("Should expose memFS and osFS factories", func() {
			Expect(FileSystems).To(HaveKey("memFS"))
			Expect(FileSystems).To(HaveKey("osFS"))
		})

		It("Should produce a working FS for every backend", func() {
			for fsName, openFS := range FileSystems {
				By("backend: " + fsName)
				fs := openFS()
				f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
				DeferClose(f)
				MustSucceed(f.Write([]byte("hello")))
				Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
			}
		})
	})
})
