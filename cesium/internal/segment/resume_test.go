// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resume", func() {
	var (
		fs     xfs.FS
		budget *segment.Budget
	)
	BeforeEach(func() {
		fs = xfs.NewMem()
		budget = MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor))
	})
	read := func(name string) []byte {
		GinkgoHelper()
		f := MustSucceed(fs.Open(name, os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		return MustSucceed(io.ReadAll(f))
	}
	// write writes each piece to the segment name of c through its own writer: a new
	// one first, then each later one resumed at the commit of the one before.
	write := func(c segment.Config, name string, pieces ...[]byte) []byte {
		GinkgoHelper()
		h := MustSucceed(budget.CreateHeld(fs, name))
		w := segment.NewWriter(h, c)
		for i, p := range pieces {
			if i > 0 {
				length, crc := w.Commit()
				Expect(h.Close()).To(Succeed())
				h = MustSucceed(budget.OpenHeld(fs, name))
				w = MustSucceed(segment.Resume(h, c, int64(length), crc))
			}
			Expect(w.Write(p)).To(Succeed())
		}
		w.Commit()
		Expect(h.Close()).To(Succeed())
		return read(name)
	}
	// split cuts stream at each of cuts.
	split := func(stream []byte, cuts ...int64) [][]byte {
		var pieces [][]byte
		prev := int64(0)
		for _, cut := range cuts {
			pieces = append(pieces, stream[prev:cut])
			prev = cut
		}
		return append(pieces, stream[prev:])
	}
	fixed := pattern(3*segment.BlockSize + 40)
	variable, starts := samples(
		10,
		3*segment.BlockSize,
		7,
		segment.BlockSize-40,
		5,
		600,
		2*segment.BlockSize,
	)

	DescribeTable(
		"Should write the same file as a writer that never stops",
		func(c segment.Config, stream []byte, cuts ...int64) {
			Expect(write(c, "resumed.seg", split(stream, cuts...)...)).
				To(Equal(write(c, "whole.seg", stream)))
		},
		Entry("fixed density, inside a block", cfg, fixed, int64(10), int64(4000)),
		Entry("fixed density, at a block edge", cfg, fixed, int64(segment.BlockSize)),
		Entry(
			"fixed density, across blocks",
			cfg,
			fixed,
			int64(segment.BlockSize+5),
			int64(3*segment.BlockSize+1),
		),
		Entry("variable density, after a short sample", varCfg, variable, starts[1]),
		Entry(
			"variable density, after a sample that spans blocks with no start",
			varCfg,
			variable,
			starts[2],
			starts[4],
		),
		Entry(
			"variable density, after a block whose first start has samples before it",
			varCfg,
			variable,
			starts[5],
		),
		Entry(
			"variable density, in each unfinished block",
			varCfg,
			variable,
			starts[1],
			starts[2],
			starts[3],
			starts[4],
			starts[5],
		),
	)

	// committed writes stream to the segment name of c and returns its commit.
	committed := func(c segment.Config, name string, stream []byte) (int64, uint32) {
		GinkgoHelper()
		h := MustSucceed(budget.CreateHeld(fs, name))
		w := segment.NewWriter(h, c)
		Expect(w.Write(stream)).To(Succeed())
		length, crc := w.Commit()
		Expect(h.Close()).To(Succeed())
		return int64(length), crc
	}
	// flip flips a byte of the segment name of c at stream offset off.
	flip := func(c segment.Config, name string, off int64) {
		GinkgoHelper()
		f := MustSucceed(fs.Open(name, os.O_RDWR))
		b := make([]byte, 1)
		MustSucceed(f.ReadAt(b, c.FileOffset(off)))
		b[0] ^= 0xFF
		MustSucceed(f.WriteAt(b, c.FileOffset(off)))
		Expect(f.Close()).To(Succeed())
	}

	It("Should refuse a tail whose committed bytes fail its checksum", func() {
		length, crc := committed(cfg, "3.seg", fixed)
		flip(cfg, "3.seg", length-1)
		h := MustSucceed(budget.OpenHeld(fs, "3.seg"))
		Expect(segment.Resume(h, cfg, length, crc)).Error().
			To(MatchError(place.ErrCorrupt))
		Expect(h.Close()).To(Succeed())
	})

	It("Should refuse a variable segment whose block of the last start fails", func() {
		length, crc := committed(varCfg, "3.seg", variable[:starts[5]])
		// A body byte of a sample in the walk's first block: no sample length changes.
		flip(varCfg, "3.seg", starts[3]+4)
		h := MustSucceed(budget.OpenHeld(fs, "3.seg"))
		Expect(segment.Resume(h, varCfg, length, crc)).Error().
			To(MatchError(place.ErrCorrupt))
		Expect(h.Close()).To(Succeed())
	})
})
