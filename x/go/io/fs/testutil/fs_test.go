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
	"os"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// matchingTempDirs returns the basenames of every entry in os.TempDir that
// starts with the given prefix. Used to observe whether OpenOS leaves a
// directory behind after its enclosing scope exits.
func matchingTempDirs(prefix string) []string {
	entries, err := os.ReadDir(os.TempDir())
	Expect(err).ToNot(HaveOccurred())
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			out = append(out, e.Name())
		}
	}
	return out
}

var _ = Describe("FS Testutil", func() {
	Describe("OpenMem", func() {
		It("Should return a usable in-memory FS", func() {
			fs := OpenMem()
			Expect(fs).ToNot(BeNil())
			f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
		})

		It("Should return a fresh FS on each call so callers cannot leak state across tests", func() {
			a := OpenMem()
			b := OpenMem()
			fa := MustSucceed(a.Open("only-in-a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(fa)
			Expect(MustSucceed(b.Exists("only-in-a.bin"))).To(BeFalse())
		})

		It("Should leave nothing on disk", func() {
			before := matchingTempDirs("testdata-")
			OpenMem()
			Expect(matchingTempDirs("testdata-")).To(Equal(before))
		})
	})

	Describe("OpenOS", func() {
		It("Should return a usable on-disk FS", func() {
			fs := OpenOS()
			Expect(fs).ToNot(BeNil())
			f := MustSucceed(fs.Open("a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(f)
			MustSucceed(f.Write([]byte("hello")))
			Expect(MustSucceed(fs.Exists("a.bin"))).To(BeTrue())
		})

		It("Should return a fresh FS rooted at its own tempdir on each call", func() {
			a := OpenOS()
			b := OpenOS()
			fa := MustSucceed(a.Open("only-in-a.bin", os.O_CREATE|os.O_RDWR))
			DeferClose(fa)
			Expect(MustSucceed(b.Exists("only-in-a.bin"))).To(BeFalse())
		})

		It("Should back the FS with a real directory under os.TempDir", func() {
			before := matchingTempDirs("testdata-")
			OpenOS()
			after := matchingTempDirs("testdata-")
			Expect(len(after)).To(Equal(len(before) + 1))
		})

		Describe("Cleanup", Ordered, func() {
			var (
				priorDirs []string
				createdAt string
			)
			BeforeAll(func() {
				priorDirs = matchingTempDirs("testdata-")
			})
			It("Creates the tempdir while the spec is running", func() {
				OpenOS()
				current := matchingTempDirs("testdata-")
				Expect(len(current)).To(Equal(len(priorDirs) + 1))
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
				Expect(matchingTempDirs("testdata-")).To(Equal(priorDirs))
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

		It("Should bind memFS to OpenMem and osFS to OpenOS", func() {
			before := matchingTempDirs("testdata-")
			FileSystems["memFS"]()
			Expect(matchingTempDirs("testdata-")).To(Equal(before))

			FileSystems["osFS"]()
			Expect(len(matchingTempDirs("testdata-"))).To(Equal(len(before) + 1))
		})
	})

	Describe("Factory type", func() {
		It("Should be assignable from any FileSystems value", func() {
			var f Factory = OpenMem
			Expect(f()).ToNot(BeNil())
			f = OpenOS
			Expect(f()).ToNot(BeNil())
			f = FileSystems["memFS"]
			Expect(f()).ToNot(BeNil())
		})
	})
})
