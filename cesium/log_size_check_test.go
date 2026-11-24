// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: the next log of a recovery past three quarters of a small log",
	func() {
		const size = log.MinFileSize
		// recovered opens fs, which recovers it, and returns the bytes of log 2 before
		// the DB runs.
		recovered := func(ctx context.Context, fs xfs.FS) []byte {
			GinkgoHelper()
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithLogFileSize(telem.Size(size)),
			))
			b := MustSucceed(xfs.ReadFile(fs, log.Name(2)))
			Expect(errors.Join(GoRun(ctx, db.Run)(), db.Close())).To(Succeed())
			return b
		}
		// crashed returns an image, taken while the DB runs, of a root whose log 1
		// holds n Intends, or as many as it takes when n is 0. The disk has no room for
		// log 2, so the DB never switches, and the image has room.
		crashed := func(ctx SpecContext, n int) *fault.FS {
			GinkgoHelper()
			disk := MustSucceed(fault.New(fault.Config{Capacity: size + 1<<20}))
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(disk),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
				cesium.WithLogFileSize(telem.Size(size)),
			))
			stop := GoRun(ctx, db.Run)
			// Each Intend takes one block of the log.
			taken := 0
			for ; n == 0 || taken < n; taken++ {
				if err := db.Intend([]cesium.ChannelKey{1}); err != nil {
					Expect(n).To(BeZero(), "%v", err)
					break
				}
			}
			Expect(taken).To(BeNumerically(">", size/log.BlockSize*3/4))
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(errors.Join(stop(), db.Close())).To(Or(Succeed(), HaveOccurred()))
			Expect(MustSucceed(image.Exists(log.Name(2)))).To(BeFalse())
			return image
		}

		It("Should prepare the new log at the log file size when the Restart does not "+
			"fit in a full log", func(ctx SpecContext) {
			Expect(recovered(ctx, crashed(ctx, 0))).To(HaveLen(int(size)))
		})

		It("Should switch to a whole new log of the log file size",
			func(ctx SpecContext) {
				image := crashed(ctx, size/log.BlockSize*3/4+32)
				Expect(log.Prepare(image, 2, size)).To(Succeed())
				b := recovered(ctx, image)
				Expect(b).To(HaveLen(int(size)))
				// The switch writes the header of log 2.
				Expect(b[:log.HeaderSize]).ToNot(Equal(make([]byte, log.HeaderSize)))
			})
	})
