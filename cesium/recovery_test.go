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
	"encoding/binary"
	"math/rand/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Recovery", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })
	open := func(ctx context.Context) (*cesium.DB, error) {
		return cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		)
	}
	// cycle opens the database, runs it, and closes it.
	cycle := func(ctx context.Context) {
		GinkgoHelper()
		db := MustSucceed(open(ctx))
		stop := StartRun(db.Run)
		stop()
		Expect(db.Close()).To(Succeed())
	}
	// frameOffsets returns the offsets of the frames of log 1, which hold only marks.
	frameOffsets := func() []int64 {
		GinkgoHelper()
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		var offsets []int64
		b := make([]byte, 4)
		for off := int64(log.FramesStart); off < 1<<20; off += log.BlockSize {
			MustSucceed(f.ReadAt(b, off))
			if binary.LittleEndian.Uint32(b) == 0x464C5043 {
				offsets = append(offsets, off)
			}
		}
		return offsets
	}
	flip := func(off int64) {
		GinkgoHelper()
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		b := make([]byte, 1)
		MustSucceed(f.ReadAt(b, off))
		b[0] ^= 0xFF
		MustSucceed(f.WriteAt(b, off))
		Expect(f.Close()).To(Succeed())
	}

	It("Should open a new database in an empty root, and open it again", func(
		ctx SpecContext,
	) {
		cycle(ctx)
		Expect(fs.Exists("0000000000000001.head")).To(BeTrue())
		Expect(fs.Exists(log.Name(1))).To(BeTrue())
		cycle(ctx)
		Expect(frameOffsets()).To(HaveLen(2))
	})

	It("Should close a database that never ran", func(ctx SpecContext) {
		db := MustSucceed(open(ctx))
		Expect(db.Close()).To(Succeed())
		cycle(ctx)
	})

	It("Should refuse a root that holds the old layout", func(ctx SpecContext) {
		old := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fs)))
		Expect(old.CreateChannel(ctx, cesium.Channel{
			Key:      1,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(old.Close()).To(Succeed())
		Expect(open(ctx)).Error().To(MatchError(cesium.ErrOldLayout))
	})

	It("Should cut a torn tail after the last frame", func(ctx SpecContext) {
		cycle(ctx)
		last := frameOffsets()[0]
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		MustSucceed(f.WriteAt([]byte("torn bytes of a frame"), last+log.BlockSize))
		Expect(f.Close()).To(Succeed())
		cycle(ctx)
	})

	It("Should refuse a log with a valid frame after a bad one", func(ctx SpecContext) {
		cycle(ctx)
		cycle(ctx)
		offsets := frameOffsets()
		Expect(offsets).To(HaveLen(2))
		flip(offsets[0] + 8)
		Expect(open(ctx)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("holds a valid frame at")),
		))
	})

	It("Should start a new database over a head 1 that is not complete and a log 1 "+
		"with no frame", func(ctx SpecContext) {
		cycle(ctx)
		f := MustSucceed(fs.Open("0000000000000001.head", os.O_RDWR))
		MustSucceed(f.WriteAt(make([]byte, 512), 0))
		Expect(f.Close()).To(Succeed())
		g := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		MustSucceed(g.WriteAt(make([]byte, 1<<20), 0))
		Expect(g.Close()).To(Succeed())
		cycle(ctx)
	})

	It("Should refuse a head 1 that is not complete when log 1 holds a frame", func(
		ctx SpecContext,
	) {
		cycle(ctx)
		f := MustSucceed(fs.Open("0000000000000001.head", os.O_RDWR))
		MustSucceed(f.WriteAt(make([]byte, 512), 0))
		Expect(f.Close()).To(Succeed())
		Expect(open(ctx)).Error().To(MatchError(place.ErrCorrupt))
	})
})
