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
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// forgeClock returns the clock key that gives a segment of c with data type dataType
// the checksum start of c, so that both pass the same blocks.
func forgeClock(c segment.Config, dataType uint8) uint32 {
	GinkgoHelper()
	tab := crc32.MakeTable(crc32.Castagnoli)
	index := make(map[byte]byte, 256)
	for i := range 256 {
		index[byte(tab[i]>>24)] = byte(i)
	}
	Expect(index).To(HaveLen(256))
	prefix := binary.LittleEndian.AppendUint64(nil, c.DB)
	prefix = binary.LittleEndian.AppendUint64(prefix, c.ID)
	prefix = binary.LittleEndian.AppendUint64(prefix, 0)
	prefix = append(prefix, 1, dataType)
	prefix = binary.LittleEndian.AppendUint16(prefix, segment.Version)
	var steps [4]byte
	x := ^placeAt(c, 0).Start()
	for i := 3; i >= 0; i-- {
		steps[i] = index[byte(x>>24)]
		x = (x ^ tab[steps[i]]) << 8
	}
	var clock [4]byte
	state := ^crc32.Update(0, tab, prefix)
	for i, step := range steps {
		clock[i] = byte(state) ^ step
		state = tab[step] ^ state>>8
	}
	forged := c
	forged.DataType, forged.Clock = dataType, binary.LittleEndian.Uint32(clock[:])
	Expect(placeAt(forged, 0).Start()).To(Equal(placeAt(c, 0).Start()))
	return forged.Clock
}

var _ = Describe("Search", func() {
	var fs xfs.FS
	BeforeEach(func() { fs = xfs.NewMem() })
	// write writes stream to the segment file of c.
	write := func(c segment.Config, stream []byte) {
		GinkgoHelper()
		budget := MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor))
		h := MustSucceed(budget.Create(fs, "3.seg"))
		Expect(segment.NewWriter(h, c).Write(stream)).To(Succeed())
		Expect(h.Close()).To(Succeed())
	}
	// spoil changes a byte of block k of the segment file of c.
	spoil := func(c segment.Config, k int64) {
		GinkgoHelper()
		b := MustSucceed(xfs.ReadFile(fs, "3.seg"))
		b[c.FileOffset(k*segment.BlockSize)+10] ^= 0xFF
		Expect(xfs.WriteFile(fs, "3.seg", b)).To(Succeed())
	}
	// search searches the segment file among candidates.
	search := func(candidates ...segment.Config) (segment.Config, int64, string) {
		GinkgoHelper()
		f := MustSucceed(fs.Open("3.seg", os.O_RDONLY))
		defer func() { Expect(f.Close()).To(Succeed()) }()
		size := MustSucceed(f.Stat()).Size()
		found, checked, reason, err := segment.Search(f, size, candidates)
		Expect(err).ToNot(HaveOccurred())
		return found, checked, reason
	}
	// others returns c with other types and clocks.
	others := func(c segment.Config) []segment.Config {
		out := make([]segment.Config, 0, 4)
		for _, d := range []struct {
			dataType uint8
			clock    uint32
		}{{c.DataType + 1, c.Clock}, {c.DataType, c.Clock + 1}, {2, 5}, {11, 0}} {
			o := c
			o.DataType, o.Clock = d.dataType, d.clock
			out = append(out, o)
		}
		return out
	}
	fixedOthers, varOthers := others(cfg), others(varCfg)
	variable, _ := samples(70_000, 10, 120_000, 9)

	DescribeTable("Should find the type and clock of a segment among its candidates, "+
		"and count the full blocks that pass",
		func(c segment.Config, stream []byte, blocks int64) {
			write(c, stream)
			candidates := append(append(fixedOthers[:2:2], c), varOthers...)
			found, checked, reason := search(candidates...)
			Expect(reason).To(BeEmpty())
			Expect(found).To(Equal(c))
			Expect(checked).To(Equal(blocks))
		},
		Entry("of fixed density", cfg, pattern(3*segment.BlockSize+500), int64(3)),
		Entry("of variable density", varCfg, variable, int64(2)),
	)

	It("Should stop the count at the first full block that fails", func() {
		write(cfg, pattern(4*segment.BlockSize))
		spoil(cfg, 2)
		found, checked, reason := search(append(fixedOthers, cfg)...)
		Expect(reason).To(BeEmpty())
		Expect(found).To(Equal(cfg))
		Expect(checked).To(Equal(int64(2)))
	})

	It("Should find the config by a later block when the first fails, and count no "+
		"block", func() {
		write(cfg, pattern(3*segment.BlockSize))
		spoil(cfg, 0)
		found, checked, reason := search(append(fixedOthers, cfg)...)
		Expect(reason).To(BeEmpty())
		Expect(found).To(Equal(cfg))
		Expect(checked).To(BeZero())
	})

	It("Should find no config of a file with no full block", func() {
		write(cfg, pattern(segment.BlockSize-1))
		_, _, reason := search(cfg)
		Expect(reason).To(Equal("it holds no full block"))
	})

	It("Should find no config when no candidate passes", func() {
		write(cfg, pattern(2*segment.BlockSize))
		_, _, reason := search(append(fixedOthers, varOthers...)...)
		Expect(reason).To(Equal("no candidate passes a full block"))
	})

	It("Should find no config when two candidates pass the first block that passes",
		func() {
			write(cfg, pattern(2*segment.BlockSize))
			forged := cfg
			forged.DataType = 8
			forged.Clock = forgeClock(cfg, forged.DataType)
			_, _, reason := search(cfg, forged)
			Expect(reason).To(Equal(fmt.Sprintf(
				"type 4 with clock 2097153 and type 8 with clock %d both pass block 0",
				forged.Clock,
			)))
		})
})
