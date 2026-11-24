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
	"encoding/binary"
	"hash/crc32"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// placeSum is the checksum of the RFC's place rule: CRC32C over the 32-byte place,
// then the covered bytes, with the fields written out here and not taken from the
// engine.
func placeSum(
	db, file, offset uint64,
	kind, code uint8,
	version uint16,
	clock uint32,
	covered []byte,
) uint32 {
	place := make([]byte, 32, 32+len(covered))
	binary.LittleEndian.PutUint64(place[0:], db)
	binary.LittleEndian.PutUint64(place[8:], file)
	binary.LittleEndian.PutUint64(place[16:], offset)
	place[24], place[25] = kind, code
	binary.LittleEndian.PutUint16(place[26:], version)
	binary.LittleEndian.PutUint32(place[28:], clock)
	return crc32.Checksum(
		append(place, covered...),
		crc32.MakeTable(crc32.Castagnoli),
	)
}

var _ = Describe("Check: the place of each segment block", func() {
	It("Should check each full block of a segment with the identity, offset, type, "+
		"version 1, and clock in its place", func(ctx SpecContext) {
		const (
			block   = 65536
			trailer = 4
			n       = 20000
		)
		fs := xfs.NewMem()
		db := MustSucceed(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
		))
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		chs := []cesium.Channel{
			{Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT},
			{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		}
		Expect(db.CreateChannel(ctx, chs...)).To(Succeed())
		stamps, values := make([]telem.TimeStamp, n), make([]int64, n)
		for i := range n {
			stamps[i], values[i] = telem.TimeStamp(i+1)*telem.SecondTS, int64(i)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    stamps[0],
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(
			keys,
			[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(values)},
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		stop()
		Expect(db.Close()).To(Succeed())
		// The database identity is bytes 16 to 23 of the log header.
		header := make([]byte, 24)
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDONLY))
		MustSucceed(f.ReadAt(header, 0))
		Expect(f.Close()).To(Succeed())
		identity := binary.LittleEndian.Uint64(header[16:])
		checked := 0
		for _, ch := range chs {
			dir := strconv.FormatUint(uint64(ch.Key), 10)
			for _, info := range MustSucceed(fs.List(dir)) {
				hex, ok := strings.CutSuffix(info.Name(), ".seg")
				if !ok {
					continue
				}
				segment := MustSucceed(strconv.ParseUint(hex, 16, 64))
				b := MustSucceed(xfs.ReadFile(MustSucceed(fs.Sub(dir)), info.Name()))
				for at := 0; at+block+trailer <= len(b); at += block + trailer {
					want := placeSum(
						identity,
						segment,
						uint64(at),
						1,
						uint8(telem.CodeOf(ch.DataType)),
						1,
						idx,
						b[at:at+block],
					)
					Expect(binary.LittleEndian.Uint32(b[at+block:])).To(
						Equal(want),
						"block at %d of segment %d of key %d", at, segment, ch.Key,
					)
					checked++
				}
			}
		}
		// Each key holds 2 full blocks of its 160,000 bytes.
		Expect(checked).To(Equal(4))
	})
})
