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
	"bytes"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	. "github.com/synnaxlabs/x/testutil"
)

// flipOnce flips a byte of the first read that covers at.
type flipOnce struct {
	r       *bytes.Reader
	at      int64
	flipped bool
}

func (f *flipOnce) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.r.ReadAt(p, off)
	if !f.flipped && off <= f.at && f.at < off+int64(n) {
		p[f.at-off] ^= 0xFF
		f.flipped = true
	}
	return n, err
}

var _ = Describe("Checks", func() {
	cfg := segment.Config{DB: 7, ID: 3, DataType: 4, Clock: 5, Bare: true}
	// old returns an old file of size bytes and its checks.
	old := func(size int) ([]byte, *segment.Checks) {
		b := make([]byte, size)
		r := rand.New(rand.NewPCG(1, 2))
		for i := range b {
			b[i] = byte(r.Uint32())
		}
		var sums []uint32
		for start := 0; start < size; start += segment.BlockSize {
			end := min(start+segment.BlockSize, size)
			sums = append(
				sums,
				cfg.BlockSum(int64(start/segment.BlockSize), b[start:end]),
			)
		}
		return b, MustSucceed(cfg.DecodeChecks(cfg.EncodeChecks(sums), int64(size)))
	}

	It("Should decode the checks that it encodes", func() {
		sums := []uint32{1, 2, 3}
		Expect(cfg.DecodeChecks(cfg.EncodeChecks(sums), 2*segment.BlockSize+1)).
			To(Equal(&segment.Checks{Length: 2*segment.BlockSize + 1, Sums: sums}))
	})

	DescribeTable("Should refuse a check file that does not hold the checks", func(
		change func([]byte) []byte,
		decoder segment.Config,
		msg string,
	) {
		b := change(cfg.EncodeChecks([]uint32{1, 2, 3}))
		Expect(decoder.DecodeChecks(b, 2*segment.BlockSize+1)).Error().To(SatisfyAll(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring(msg)),
		))
	},
		Entry("one sum short", func(b []byte) []byte { return b[4:] }, cfg,
			"check file of segment 3 holds 12 bytes, not 16"),
		Entry("a flipped sum", func(b []byte) []byte { b[0] ^= 1; return b }, cfg,
			"check file of segment 3 fails its checksum"),
		Entry("the file of another segment", func(b []byte) []byte { return b },
			segment.Config{DB: 7, ID: 4, Bare: true},
			"check file of segment 4 fails its checksum"),
	)

	It("Should read across blocks of a checked old file", func() {
		b, checks := old(2*segment.BlockSize + 100)
		c := cfg
		c.Checks = checks
		p := make([]byte, segment.BlockSize+50)
		Expect(c.ReadAt(bytes.NewReader(b), p, segment.BlockSize-25)).To(Succeed())
		Expect(p).To(Equal(b[segment.BlockSize-25 : 2*segment.BlockSize+25]))
		last := make([]byte, 100)
		Expect(c.ReadAt(bytes.NewReader(b), last, 2*segment.BlockSize)).To(Succeed())
		Expect(last).To(Equal(b[2*segment.BlockSize:]))
	})

	It("Should read a block again after one bad read", func() {
		b, checks := old(segment.BlockSize + 10)
		c := cfg
		c.Checks = checks
		p := make([]byte, 10)
		r := &flipOnce{r: bytes.NewReader(b), at: segment.BlockSize + 3}
		Expect(c.ReadAt(r, p, segment.BlockSize)).To(Succeed())
		Expect(p).To(Equal(b[segment.BlockSize:]))
		Expect(r.flipped).To(BeTrue())
	})

	It("Should refuse a block that fails both reads", func() {
		b, checks := old(segment.BlockSize + 10)
		c := cfg
		c.Checks = checks
		b[5] ^= 0xFF
		Expect(c.ReadAt(bytes.NewReader(b), make([]byte, 4), 0)).To(SatisfyAll(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring("block 0 of segment 3 fails its check")),
		))
	})

	It("Should refuse a block whose sums belong to another segment", func() {
		b, checks := old(100)
		c := segment.Config{DB: 7, ID: 9, DataType: 4, Clock: 5, Bare: true}
		c.Checks = checks
		Expect(c.ReadAt(bytes.NewReader(b), make([]byte, 4), 0)).
			To(MatchError(place.ErrCorrupt))
	})

	It("Should read the checked blocks of a version 0 segment in whole blocks", func() {
		b, checks := old(2*segment.BlockSize + 100)
		checked := cfg
		checked.Checks = checks
		buf := make([]byte, 3*segment.BlockSize)
		n := MustSucceed(
			checked.ReadChecked(bytes.NewReader(b), buf, 1, int64(len(b)), 0),
		)
		Expect(buf[:n]).To(Equal(b[segment.BlockSize:]))
	})

	DescribeTable("Should refuse a version 0 block that fails its check", func(
		at int64,
		checked bool,
		msg string,
	) {
		b, sums := old(2*segment.BlockSize + 100)
		c := cfg
		if checked {
			c.Checks = sums
		}
		r := &flipOnce{r: bytes.NewReader(b), at: at}
		buf := make([]byte, 3*segment.BlockSize)
		Expect(c.ReadChecked(r, buf, 0, int64(len(b)), 0)).Error().To(SatisfyAll(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring(msg)),
		))
	},
		Entry("a flipped byte", int64(segment.BlockSize+7), true,
			"block 1 of segment 3 fails its check"),
		Entry("no checks", int64(-1), false, "block 0 of segment 3 fails its check"),
	)
})
