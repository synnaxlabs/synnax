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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	metaIndexCh = cesium.Channel{
		Key:      1,
		Name:     "time",
		IsIndex:  true,
		DataType: telem.TimestampT,
	}
	metaDataCh = cesium.Channel{
		Key:      2,
		Name:     "data",
		Index:    1,
		DataType: telem.Float64T,
	}
	metaAddedCh = cesium.Channel{
		Key:      3,
		Name:     "added",
		Index:    1,
		DataType: telem.Float64T,
	}
)

// writeSamples writes three samples to each of the channels, starting at start seconds.
// The first key must be the index channel.
func writeSamples(
	ctx context.Context,
	db *cesium.DB,
	start telem.TimeStamp,
	keys ...cesium.ChannelKey,
) {
	GinkgoHelper()
	series := make([]telem.Series, len(keys))
	series[0] = telem.NewSeriesSecondsTSV(start, start+1, start+2)
	for i := 1; i < len(keys); i++ {
		series[i] = telem.NewSeriesV(1.0, 2.0, 3.0)
	}
	Expect(db.Write(ctx, start*telem.SecondTS, telem.MultiFrame(keys, series))).
		To(Succeed())
}

func entryNames(fs xfs.FS, dir string) []string {
	GinkgoHelper()
	return lo.Map(
		MustSucceed(fs.List(dir)),
		func(info xfs.FileInfo, _ int) string { return info.Name() },
	)
}

func readAll(fs xfs.FS, name string) []byte {
	GinkgoHelper()
	f := MustSucceed(fs.Open(name, os.O_RDONLY))
	data := MustSucceed(io.ReadAll(f))
	Expect(f.Close()).To(Succeed())
	return data
}

var _ = Describe("Meta backup", func() {
	var fs, backup, orphans xfs.FS
	BeforeEach(func() {
		root := xfs.NewMem()
		fs = MustSucceed(root.Sub("db"))
		backup = MustSucceed(root.Sub("backup"))
		orphans = MustSucceed(root.Sub("orphans"))
	})

	Describe("BackupMeta", func() {
		It(
			"Should copy the metadata of every channel and no telemetry",
			func(ctx SpecContext) {
				db := openDBOnFS(ctx, fs)
				Expect(db.CreateChannel(ctx, metaIndexCh, metaDataCh)).To(Succeed())
				writeSamples(ctx, db, 1, metaIndexCh.Key, metaDataCh.Key)
				Expect(db.Close()).To(Succeed())
				Expect(entryNames(fs, "2")).To(ContainElement(Not(Equal("meta.json"))))

				Expect(cesium.BackupMeta(fs, backup)).To(Succeed())
				Expect(entryNames(backup, "")).To(ConsistOf("1", "2"))
				Expect(entryNames(backup, "1")).To(ConsistOf("meta.json"))
				Expect(entryNames(backup, "2")).To(ConsistOf("meta.json"))
				Expect(readAll(backup, "2/meta.json")).
					To(Equal(readAll(fs, "2/meta.json")))
			},
		)

		It(
			"Should skip deleted channels and entries that are not channels",
			func(ctx SpecContext) {
				db := openDBOnFS(ctx, fs)
				Expect(db.CreateChannel(ctx, metaIndexCh)).To(Succeed())
				Expect(db.Close()).To(Succeed())
				deleted := MustSucceed(fs.Sub("4-DELETE-123"))
				Expect(MustSucceed(deleted.Open("meta.json", os.O_CREATE)).Close()).
					To(Succeed())
				MustSucceed(fs.Sub("notes"))
				Expect(MustSucceed(fs.Open("5", os.O_CREATE)).Close()).To(Succeed())

				Expect(cesium.BackupMeta(fs, backup)).To(Succeed())
				Expect(entryNames(backup, "")).To(ConsistOf("1"))
			},
		)
	})

	Describe("RestoreMeta", func() {
		It(
			"Should put back the metadata and move channels the backup does not know",
			func(ctx SpecContext) {
				db := openDBOnFS(ctx, fs)
				Expect(db.CreateChannel(ctx, metaIndexCh, metaDataCh)).To(Succeed())
				writeSamples(ctx, db, 1, metaIndexCh.Key, metaDataCh.Key)
				Expect(db.Close()).To(Succeed())
				Expect(cesium.BackupMeta(fs, backup)).To(Succeed())

				db = openDBOnFS(ctx, fs)
				Expect(db.RenameChannels(
					ctx,
					map[cesium.ChannelKey]string{metaDataCh.Key: "renamed"},
				)).To(Succeed())
				Expect(db.CreateChannel(ctx, metaAddedCh)).To(Succeed())
				writeSamples(
					ctx,
					db,
					10,
					metaIndexCh.Key,
					metaDataCh.Key,
					metaAddedCh.Key,
				)
				Expect(db.Close()).To(Succeed())

				Expect(cesium.RestoreMeta(fs, backup, orphans)).
					To(Equal([]cesium.ChannelKey{metaAddedCh.Key}))
				Expect(entryNames(fs, "")).To(ConsistOf("1", "2"))
				Expect(entryNames(orphans, "")).To(ConsistOf("3"))
				Expect(entryNames(orphans, "3")).To(ContainElement("meta.json"))

				db = mustOpenDBOnFS(ctx, fs)
				Expect(MustSucceed(db.RetrieveChannel(ctx, metaDataCh.Key)).Name).
					To(Equal("data"))
				Expect(db.RetrieveChannel(ctx, metaAddedCh.Key)).Error().
					To(MatchError(cesium.ErrChannelNotFound))
				frame := MustSucceed(db.Read(ctx, telem.TimeRangeMax, metaDataCh.Key))
				Expect(frame.Len()).To(BeEquivalentTo(6))
				Expect(db.CreateChannel(ctx, metaAddedCh)).To(Succeed())
			},
		)

		It(
			"Should move nothing when the backup has every channel",
			func(ctx SpecContext) {
				db := openDBOnFS(ctx, fs)
				Expect(db.CreateChannel(ctx, metaIndexCh)).To(Succeed())
				Expect(db.Close()).To(Succeed())
				Expect(cesium.BackupMeta(fs, backup)).To(Succeed())

				Expect(cesium.RestoreMeta(fs, backup, orphans)).To(BeEmpty())
				Expect(entryNames(orphans, "")).To(BeEmpty())
			},
		)
	})
})
