// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Counter", func() {
	fileSystems := map[string]func() xfs.FS{
		"memFS": func() xfs.FS {
			return xfs.NewMem()
		},
		"osFS": func() xfs.FS { return MustSucceed(xfs.Default.Sub("./testdata")) },
	}

	for fsName, makeFS := range fileSystems {
		fsName, makeFS := fsName, makeFS
		var fsRoot, fs xfs.FS

		Context("FS:"+fsName, Ordered, func() {
			BeforeEach(func() {
				fsRoot = makeFS()
				_fs, err := fsRoot.Sub("test-spec")
				fs = _fs
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				Expect(fsRoot.Remove("test-spec")).To(Succeed())
			})
			It("Should create a new counter when the file does not exist", func() {
				f, err := fs.Open("counterfile", os.O_CREATE|os.O_EXCL|os.O_RDWR)
				Expect(err).ToNot(HaveOccurred())

				c, err := NewInt32Counter(f)
				Expect(err).ToNot(HaveOccurred())
				Expect(c.Value()).To(Equal(int32(0)))
				Expect(f.Close()).To(Succeed())
			})
			It("Should read the existing value when the file does exist", func() {
				f := MustSucceed(
					fs.Open("counterfile", os.O_CREATE|os.O_EXCL|os.O_RDWR),
				)
				Expect(f.Write([]byte{0x15, 0x1, 0x0, 0x0})).To(Equal(4))
				c := MustSucceed(NewInt32Counter(f))
				Expect(c.Value()).To(Equal(int32(277)))
				Expect(f.Close()).To(Succeed())
			})
			It("Should be concurrent-safe", func() {
				var (
					keys = make([]int32, 1000)
					wg   = sync.WaitGroup{}
					f    xfs.File
					c    *Int32Counter
				)

				f, err := fs.Open("counterfile", os.O_CREATE|os.O_EXCL|os.O_RDWR)
				Expect(err).ToNot(HaveOccurred())

				c, err = NewInt32Counter(f)
				Expect(err).ToNot(HaveOccurred())

				wg.Add(1000)
				for i := range 1000 {
					i := i
					go func() {
						defer wg.Done()
						val, err := c.Add(1)
						keys[i] = val
						Expect(err).ToNot(HaveOccurred())
					}()
				}

				wg.Wait()

				Expect(keys).To(HaveLen(1000))
				Expect(lo.Uniq(keys)).To(HaveLen(1000))
				Expect(f.Close()).To(Succeed())
			})
		})
	}
})
