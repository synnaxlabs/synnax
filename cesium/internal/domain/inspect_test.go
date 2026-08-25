// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/domain"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Inspect", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				db *domain.DB
				fs xfs.FS
			)
			BeforeEach(func() {
				fs = openFS()
				db = MustSucceed(domain.Open(domain.Config{
					FS:              fs,
					Instrumentation: PanicLogger(),
				}))
			})
			AfterEach(func() {
				Expect(db.Close()).To(Succeed())
			})

			readIndex := func() []byte {
				GinkgoHelper()
				f := MustSucceed(fs.Open(domain.IndexFileName, os.O_RDONLY))
				defer func() { Expect(f.Close()).To(Succeed()) }()
				info := MustSucceed(f.Stat())
				b := make([]byte, info.Size())
				MustSucceed(f.ReadAt(b, 0))
				return b
			}

			Describe("DecodeRecords", func() {
				It("Should decode the persisted index", func(ctx SpecContext) {
					Expect(domain.Write(
						ctx,
						db,
						(10 * telem.SecondTS).Range(15*telem.SecondTS+1),
						[]byte{10, 11, 12, 13, 14, 15},
					)).To(Succeed())
					Expect(domain.Write(
						ctx,
						db,
						(20 * telem.SecondTS).Range(23*telem.SecondTS+1),
						[]byte{20, 21, 22, 23},
					)).To(Succeed())
					records := domain.DecodeRecords(readIndex())
					Expect(records).To(HaveLen(2))
					Expect(records[0].TimeRange).To(Equal(
						(10 * telem.SecondTS).Range(15*telem.SecondTS + 1),
					))
					Expect(records[0].Size).To(Equal(uint32(6)))
					Expect(records[1].TimeRange).To(Equal(
						(20 * telem.SecondTS).Range(23*telem.SecondTS + 1),
					))
					Expect(records[1].Size).To(Equal(uint32(4)))
					Expect(records[1].FileKey).ToNot(BeZero())
				})

				It("Should drop a trailing partial record", func(ctx SpecContext) {
					Expect(domain.Write(
						ctx,
						db,
						(10 * telem.SecondTS).Range(15*telem.SecondTS+1),
						[]byte{10, 11, 12},
					)).To(Succeed())
					b := readIndex()
					Expect(b).To(HaveLen(domain.RecordSize))
					records := domain.DecodeRecords(b[:domain.RecordSize-3])
					Expect(records).To(BeEmpty())
				})

				It("Should decode empty input to no records", func() {
					Expect(domain.DecodeRecords(nil)).To(BeEmpty())
				})
			})

			Describe("DataFileName", func() {
				It("Should name domain files by their key", func() {
					Expect(domain.DataFileName(3)).To(Equal("3" + domain.Extension))
					Expect(domain.IndexFileName).To(Equal("index" + domain.Extension))
					Expect(domain.CounterFileName).To(
						Equal("counter" + domain.Extension),
					)
				})
			})
		})
	}
})
