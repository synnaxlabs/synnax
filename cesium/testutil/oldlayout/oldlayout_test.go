// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package oldlayout_test

import (
	"context"
	"path"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/testutil/oldlayout"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// files returns the bytes of each file under fs by its path.
func files(fs xfs.FS) map[string][]byte {
	GinkgoHelper()
	out := make(map[string][]byte)
	var walk func(dir string)
	walk = func(dir string) {
		for _, info := range MustSucceed(fs.List(dir)) {
			name := path.Join(dir, info.Name())
			if info.IsDir() {
				walk(name)
				continue
			}
			out[name] = MustSucceed(xfs.ReadFile(fs, name))
		}
	}
	walk("")
	return out
}

// fixture is the root that Cesium at core/v0.62.0 wrote with a file size cap of 48
// bytes: an index (1), an int32 key (2), and a string key (3) timed by it, 40 samples
// in commits of 5. Sample i has the stamp i + 1 seconds, the int32 i + 1, and a string
// of i % 7 + 1 bytes of x.
func fixture() xfs.FS {
	GinkgoHelper()
	return MustSucceed(xfs.Default.Sub("testdata/v0.62.0"))
}

// writeOld writes the samples of the fixture through the old open path of Cesium to
// fs.
func writeOld(ctx context.Context, fs xfs.FS) {
	GinkgoHelper()
	db := MustSucceed(cesium.Open(
		ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithFileSizeCap(48*telem.Byte),
	))
	Expect(db.CreateChannel(
		ctx,
		cesium.Channel{Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT},
		cesium.Channel{Key: 2, Name: "i32", Index: 1, DataType: telem.Int32T},
		cesium.Channel{Key: 3, Name: "str", Index: 1, DataType: telem.StringT},
	)).To(Succeed())
	const n = 40
	stamps, ints, strs := make(
		[]telem.TimeStamp,
		n,
	), make(
		[]int32,
		n,
	), make(
		[]string,
		n,
	)
	for i := range n {
		stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
		ints[i] = int32(i + 1)
		strs[i] = strings.Repeat("x", i%7+1)
	}
	keys := []cesium.ChannelKey{1, 2, 3}
	w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
		Channels: keys,
		Start:    stamps[0],
	}))
	for c := 0; c < n; c += 5 {
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps[c : c+5]),
			telem.NewSeries(ints[c : c+5]),
			telem.NewSeries(strs[c : c+5]),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
	}
	Expect(w.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
}

var _ = Describe("Old layout", func() {
	It("Should read the channels, counters, pointers, and files of the fixture",
		func(ctx SpecContext) {
			keys := MustSucceed(oldlayout.Read(ctx, fixture()))
			Expect(keys).To(HaveLen(3))
			idx, i32 := keys[0], keys[1]
			Expect(idx.Channel.Name).To(Equal("time"))
			Expect(idx.Channel.IsIndex).To(BeTrue())
			Expect(idx.Counter).To(Equal(uint32(5)))
			Expect(idx.Pointers).To(HaveLen(4))
			Expect(idx.Pointers[0]).To(Equal(oldlayout.Pointer{
				Start: telem.SecondTS,
				End:   10*telem.SecondTS + 1,
				File:  1,
				Size:  80,
			}))
			Expect(idx.Files).To(HaveLen(5))
			Expect(idx.Files[5]).To(BeEmpty())
			Expect(i32.Channel.Index).To(Equal(cesium.ChannelKey(1)))
			Expect(i32.Pointers).To(HaveLen(3))
			Expect(i32.Pointers[2].Size).To(Equal(uint32(40)))
		})

	It("Should write the fixture again byte for byte", func(ctx SpecContext) {
		mem := xfs.NewMem()
		keys := MustSucceed(oldlayout.Read(ctx, fixture()))
		Expect(oldlayout.Write(mem, keys...)).To(Succeed())
		Expect(files(mem)).To(Equal(files(fixture())))
	})

	It("Should write a root of the old open path again byte for byte",
		func(ctx SpecContext) {
			old, mem := xfs.NewMem(), xfs.NewMem()
			writeOld(ctx, old)
			keys := MustSucceed(oldlayout.Read(ctx, old))
			Expect(oldlayout.Write(mem, keys...)).To(Succeed())
			want := files(old)
			// The old open path also writes delete-intents, which v0.62 lacks.
			Expect(want).To(HaveKey("delete-intents"))
			delete(want, "delete-intents")
			Expect(files(mem)).To(Equal(want))
			Expect(files(mem)).To(Equal(files(fixture())))
		})

	It("Should write the leftovers of an old compaction again byte for byte",
		func(ctx SpecContext) {
			old, mem := xfs.NewMem(), xfs.NewMem()
			keys := MustSucceed(oldlayout.Read(ctx, fixture()))
			keys[1].Leftovers["2.domain_temp"] = keys[1].Files[2]
			keys[1].Leftovers["2.domain_gc"] = []byte{1, 2, 3}
			delete(keys[1].Files, 2)
			Expect(oldlayout.Write(old, keys...)).To(Succeed())
			again := MustSucceed(oldlayout.Read(ctx, old))
			Expect(again[1].Leftovers).To(HaveLen(2))
			Expect(again[1].Files).ToNot(HaveKey(uint16(2)))
			Expect(oldlayout.Write(mem, again...)).To(Succeed())
			Expect(files(mem)).To(Equal(files(old)))
		})

	It("Should sync nothing, so a crash can lose every write", func(ctx SpecContext) {
		disk := MustSucceed(fault.New(fault.Config{}))
		keys := MustSucceed(oldlayout.Read(ctx, fixture()))
		Expect(oldlayout.Write(disk, keys...)).To(Succeed())
		image := MustSucceed(disk.Crash(fault.CrashConfig{RunKey: 1}))
		Expect(files(image)).To(BeEmpty())
		disk.Writeback()
		Expect(files(MustSucceed(disk.Crash(fault.CrashConfig{RunKey: 1})))).
			To(Equal(files(fixture())))
	})
})
