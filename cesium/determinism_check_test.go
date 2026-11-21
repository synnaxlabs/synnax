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
	"io"
	"maps"
	"math/rand/v2"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// files returns the bytes of each file under dir of fs, by path.
func files(fs xfs.FS, dir string) map[string][]byte {
	GinkgoHelper()
	out := map[string][]byte{}
	for _, info := range MustSucceed(fs.List(dir)) {
		name := path.Join(dir, info.Name())
		if info.IsDir() {
			maps.Copy(out, files(fs, name))
			continue
		}
		f := MustSucceed(fs.Open(name, os.O_RDONLY))
		out[name] = MustSucceed(io.ReadAll(f))
		Expect(f.Close()).To(Succeed())
	}
	return out
}

var _ = Describe("Pointer log determinism check", func() {
	// run runs one workload in a new root and returns its files: eight clocks of two
	// channels each, three synced commits on each, a drop of two keys, and a reopen
	// that closes clean.
	run := func(ctx context.Context) map[string][]byte {
		GinkgoHelper()
		fs := xfs.NewMem()
		open := func() (*cesium.DB, func() error) {
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
			))
			stop := GoRun(ctx, db.Run)
			return db, func() error {
				if err := stop(); err != nil {
					return err
				}
				return db.Close()
			}
		}
		db, closeDB := open()
		for c := range cesium.ChannelKey(8) {
			idx, data := 10+2*c, 11+2*c
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: data, Name: "data", Index: idx, DataType: telem.Int64T,
				},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{idx, data},
				Clock:    idx,
				Start:    telem.SecondTS,
			}))
			for i := range telem.TimeStamp(3) {
				MustSucceed2(w.Write(ctx, telem.MultiFrame(
					[]cesium.ChannelKey{idx, data},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1+i*2, 2+i*2),
						telem.NewSeriesV[int64](int64(c), int64(i)),
					},
				), nil))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Sync()).To(Succeed())
			}
			Expect(w.Close()).To(Succeed())
		}
		Expect(db.Intend([]cesium.ChannelKey{11, 13})).To(Succeed())
		Expect(db.Drop([]cesium.ChannelKey{11, 13})).To(Succeed())
		Expect(closeDB()).To(Succeed())
		_, closeDB = open()
		Expect(closeDB()).To(Succeed())
		return files(fs, "")
	}

	It("Should write the same bytes in every file in two runs of one workload", func(
		ctx SpecContext,
	) {
		first, second := run(ctx), run(ctx)
		var differ []string
		for name, b := range first {
			if string(second[name]) != string(b) {
				differ = append(differ, name)
			}
		}
		for name := range second {
			if _, ok := first[name]; !ok {
				differ = append(differ, name)
			}
		}
		Expect(differ).To(BeEmpty())
	})
})
