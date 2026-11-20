// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var errSync = errors.New("sync failed")

var _ = Describe("Files", func() {
	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var fs *opFS
			BeforeEach(func() { fs = newOpFS(makeFS()) })

			name := checkpoint.Name(generation)
			exists := func(name string) bool {
				GinkgoHelper()
				return MustSucceed(fs.Exists(name))
			}

			Describe("Write", func() {
				It(
					"Should write the body, sync, sync the root, then commit the "+
						"header "+
						"sector",
					func() {
						b := checkpoint.Encode(db, generation, sample(2))
						Expect(checkpoint.Write(fs, generation, b)).To(Succeed())
						Expect(fs.Ops()).To(Equal([]string{
							"truncate " + name + " " + itoa(len(b)),
							"write " + name + " 4096 " + itoa(len(b)-4096),
							"sync " + name,
							`syncdir ""`,
							"write " + name + " 0 512",
							"sync " + name,
						}))
						h, s := MustSucceed2(checkpoint.Read(fs, generation))
						Expect(h.Sequence).To(Equal(uint64(77)))
						Expect(s.Channels).To(HaveLen(3))
					},
				)

				DescribeTable(
					"Should remove the head of a failed attempt and sync the root",
					func(failSync int) {
						fs.failSync = failSync
						b := checkpoint.Encode(db, generation, sample(2))
						Expect(
							checkpoint.Write(fs, generation, b),
						).To(MatchError(errSync))
						Expect(exists(name)).To(BeFalse())
						ops := fs.Ops()
						Expect(ops[len(ops)-2:]).
							To(Equal([]string{"remove " + name, `syncdir ""`}))
						syncs := 0
						for _, op := range ops {
							if op == "sync "+name {
								syncs++
							}
						}
						Expect(
							syncs,
						).To(Equal(failSync-1), "no sync after the failed one")
					},
					Entry("at the body sync", 1),
					Entry("at the commit sync", 2),
				)

				It("Should refuse a head that exists", func() {
					b := checkpoint.Encode(db, generation, sample(2))
					Expect(checkpoint.Write(fs, generation, b)).To(Succeed())
					Expect(checkpoint.Write(fs, generation, b)).
						To(MatchError(ContainSubstring("exists")))
					_, s := MustSucceed2(checkpoint.Read(fs, generation))
					Expect(s.Channels).To(HaveLen(3))
				})
			})

			Describe("Read", func() {
				var b []byte
				BeforeEach(func() {
					b = checkpoint.Encode(db, generation, sample(4000))
					Expect(checkpoint.Write(fs, generation, b)).To(Succeed())
				})

				DescribeTable(
					"Should pass when each bad range reads well a second time",
					func(offs ...int64) {
						for i, off := range offs {
							if off < 0 {
								offs[i] = int64(len(b)) + off
							}
							fs.Flip(offs[i], 1)
						}
						_, s := MustSucceed2(checkpoint.Read(fs, generation))
						Expect(s.Channels).To(HaveLen(3))
						for _, off := range offs {
							Expect(fs.Reads(off)).To(Equal(2))
						}
					},
					Entry("header sector", int64(30)),
					Entry("second chunk", int64(4096+65_536+9)),
					Entry("chunk table, at its end", int64(-1)),
					Entry("header sector, then a chunk", int64(30), int64(4096+7)),
				)

				DescribeTable(
					"Should refuse a range that fails on both reads",
					func(off int64, msg string) {
						fs.Flip(off, -1)
						Expect(readErr(fs, generation)).To(And(
							MatchError(place.ErrCorrupt),
							MatchError(ContainSubstring(msg)),
						))
						Expect(fs.Reads(off)).To(Equal(2))
					},
					Entry("header sector", int64(30), "header sector fails"),
					Entry("first chunk", int64(4096), "chunk at 4096 fails"),
				)

				It(
					"Should report a zero header sector as incomplete after two reads",
					func() {
						f := MustSucceed(fs.Open(name, os.O_RDWR))
						Expect(f.WriteAt(make([]byte, 512), 0)).To(Equal(512))
						Expect(f.Close()).To(Succeed())
						fs.Flip(3, 0)
						Expect(
							readErr(fs, generation),
						).To(MatchError(checkpoint.ErrIncomplete))
						Expect(fs.Reads(3)).To(Equal(2))
					},
				)
			})

			Describe("Prepare", func() {
				It(
					"Should make a missing log of its size and remove a leftover head",
					func() {
						f := MustSucceed(
							fs.Open(checkpoint.Name(4), os.O_CREATE|os.O_RDWR),
						)
						Expect(f.Close()).To(Succeed())
						Expect(checkpoint.Prepare(fs, 4, 3*log.BlockSize)).To(Succeed())
						Expect(exists(checkpoint.Name(4))).To(BeFalse())
						info := MustSucceed(fs.Stat(log.Name(4)))
						Expect(info.Size()).To(Equal(int64(3 * log.BlockSize)))
					},
				)

				It("Should keep a log of its size and remake a short one", func() {
					Expect(checkpoint.Prepare(fs, 4, 3*log.BlockSize)).To(Succeed())
					writes := len(fs.Ops())
					Expect(checkpoint.Prepare(fs, 4, 3*log.BlockSize)).To(Succeed())
					Expect(fs.Ops()).To(HaveLen(writes+1), "only the head removal")
					Expect(checkpoint.Prepare(fs, 4, 4*log.BlockSize)).To(Succeed())
					info := MustSucceed(fs.Stat(log.Name(4)))
					Expect(info.Size()).To(Equal(int64(4 * log.BlockSize)))
				})
			})

			Describe("Clean", func() {
				create := func(names ...string) {
					GinkgoHelper()
					for _, n := range names {
						f := MustSucceed(fs.Open(n, os.O_CREATE|os.O_RDWR))
						Expect(f.Close()).To(Succeed())
					}
				}
				left := func() []string {
					GinkgoHelper()
					entries := MustSucceed(fs.List(""))
					names := make([]string, 0, len(entries))
					for _, e := range entries {
						names = append(names, e.Name())
					}
					return names
				}
				head, lg := checkpoint.Name, log.Name

				It(
					"Should keep the newest older head and every log from it on",
					func() {
						create(head(1), head(2), head(3), lg(1), lg(2), lg(3), lg(4))
						Expect(checkpoint.Clean(fs, 3)).To(Succeed())
						Expect(
							left(),
						).To(ConsistOf(head(2), head(3), lg(2), lg(3), lg(4)))
						Expect(fs.Ops()[len(fs.Ops())-1]).To(Equal(`syncdir ""`))
					},
				)

				It(
					"Should keep an older head across a generation with no head",
					func() {
						create(head(1), head(3), lg(1), lg(2), lg(3))
						Expect(checkpoint.Clean(fs, 3)).To(Succeed())
						Expect(
							left(),
						).To(ConsistOf(head(1), head(3), lg(1), lg(2), lg(3)))
					},
				)

				It("Should read only names of 16 hex digits as generations", func() {
					create(head(1), "3.head", head(4), lg(1), lg(2), lg(3), lg(4))
					Expect(checkpoint.Clean(fs, 4)).To(Succeed())
					Expect(
						left(),
					).To(ConsistOf(head(1), "3.head", head(4), lg(1), lg(2),
						lg(3), lg(4)))
				})

				It("Should keep no older head whose logs are not all there", func() {
					create(head(1), head(2), head(4), lg(1), lg(3), lg(4), "meta.json",
						head(1)+".damaged")
					Expect(checkpoint.Clean(fs, 4)).To(Succeed())
					Expect(left()).
						To(ConsistOf(head(4), lg(4), "meta.json", head(1)+".damaged"))
				})
			})
		})
	}
})

func readErr(fs xfs.FS, gen uint64) error {
	_, _, err := checkpoint.Read(fs, gen)
	return err
}

func itoa(n int) string { return strconv.Itoa(n) }
