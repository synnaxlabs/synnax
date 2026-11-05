// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	indexCh = cesium.Channel{
		Key:      1,
		Name:     "time",
		IsIndex:  true,
		DataType: telem.TimestampT,
	}
	dataCh  = cesium.Channel{Key: 2, Name: "data", Index: 1, DataType: telem.Float64T}
	addedCh = cesium.Channel{
		Key:      3,
		Name:     "added",
		Index:    1,
		DataType: telem.Float64T,
	}
)

func openLayer(ctx context.Context, dir, version string) *storage.Layer {
	GinkgoHelper()
	return MustSucceed(storage.OpenLayer(
		ctx,
		storage.LayerConfig{Dirname: dir, Version: version},
	))
}

// write writes three samples to each of the channels, starting at start seconds.
func write(
	ctx context.Context,
	l *storage.Layer,
	start int,
	keys ...cesium.ChannelKey,
) {
	GinkgoHelper()
	series := make([]telem.Series, len(keys))
	for i, key := range keys {
		if key == indexCh.Key {
			series[i] = telem.NewSeriesSecondsTSV(
				telem.TimeStamp(start),
				telem.TimeStamp(start+1),
				telem.TimeStamp(start+2),
			)
		} else {
			series[i] = telem.NewSeriesV(1.0, 2.0, 3.0)
		}
	}
	Expect(l.TS.Write(
		ctx,
		telem.TimeStamp(start)*telem.SecondTS,
		telem.MultiFrame(keys, series),
	)).To(Succeed())
}

func readFile(path string) string {
	GinkgoHelper()
	return string(MustSucceed(os.ReadFile(path)))
}

func writeFile(path, data string) {
	GinkgoHelper()
	Expect(os.WriteFile(path, []byte(data), 0o600)).To(Succeed())
}

func perm(path string) fs.FileMode {
	GinkgoHelper()
	return MustSucceed(os.Stat(path)).Mode().Perm()
}

func onlyBackup(dir string) string {
	GinkgoHelper()
	backups := MustSucceed(storage.ListBackups(dir))
	Expect(backups).To(HaveLen(1))
	return filepath.Join(dir, "backups", backups[0].Name)
}

var _ = Describe("Backup", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It(
		"Should copy the key-value store, the channel metadata, and the Driver key",
		func(ctx SpecContext) {
			l := openLayer(ctx, dir, "0.59.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(l.TS.CreateChannel(ctx, indexCh, dataCh)).To(Succeed())
			write(ctx, l, 1, indexCh.Key, dataCh.Key)
			Expect(l.Close()).To(Succeed())
			writeFile(filepath.Join(dir, storage.DriverKeyFile), "driver key")
			openAt(ctx, dir, "0.60.0")

			backup := onlyBackup(dir)
			Expect(entries(backup)).To(ConsistOf("kv", "cesium", storage.DriverKeyFile))
			Expect(entries(filepath.Join(backup, "kv"))).ToNot(BeEmpty())
			Expect(readFile(filepath.Join(backup, storage.DriverKeyFile))).
				To(Equal("driver key"))
			Expect(perm(filepath.Join(backup, storage.DriverKeyFile))).
				To(Equal(perm(filepath.Join(dir, storage.DriverKeyFile))))
			Expect(entries(filepath.Join(backup, "cesium"))).To(HaveLen(2))
		},
	)

	It(
		"Should keep the newest complete backups and remove unfinished ones",
		func(ctx SpecContext) {
			openAt(ctx, dir, "0.0.0")
			backups := filepath.Join(dir, "backups")
			for _, name := range []string{
				"0000000001-0.57.0",
				"0000000002-0.58.0",
				"0000000003-0.59.0",
				"0000000004-0.59.1.partial",
				"orphaned-5",
			} {
				Expect(os.MkdirAll(filepath.Join(backups, name), 0o700)).To(Succeed())
			}
			openAt(ctx, dir, "0.60.0")
			Expect(entries(backups)).To(ConsistOf(
				"0000000002-0.58.0",
				"0000000003-0.59.0",
				HaveSuffix("-unknown"),
				"orphaned-5",
			))
		},
	)
})

var _ = Describe("Restore", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It(
		"Should put the metadata back and move channels the backup does not know",
		func(ctx SpecContext) {
			l := openLayer(ctx, dir, "0.59.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("before"))).To(Succeed())
			Expect(l.TS.CreateChannel(ctx, indexCh, dataCh)).To(Succeed())
			write(ctx, l, 1, indexCh.Key, dataCh.Key)
			Expect(l.Close()).To(Succeed())
			writeFile(filepath.Join(dir, storage.DriverKeyFile), "old key")

			l = openLayer(ctx, dir, "0.60.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
			Expect(l.TS.RenameChannels(
				ctx,
				map[cesium.ChannelKey]string{dataCh.Key: "renamed"},
			)).To(Succeed())
			Expect(l.TS.CreateChannel(ctx, addedCh)).To(Succeed())
			write(ctx, l, 10, indexCh.Key, dataCh.Key, addedCh.Key)
			Expect(l.Close()).To(Succeed())
			writeFile(filepath.Join(dir, storage.DriverKeyFile), "new key")

			name := filepath.Base(onlyBackup(dir))
			res := MustSucceed(storage.Restore(dir, name))
			Expect(res.Backup.Name).To(Equal(name))
			Expect(res.Backup.Version).To(Equal("0.59.0"))
			Expect(res.OrphanedDir).
				To(HavePrefix(filepath.Join(dir, "backups", "orphaned-")))
			Expect(entries(res.OrphanedDir)).To(HaveLen(1))
			Expect(recordedVersion(dir)).To(Equal("0.59.0"))
			Expect(readFile(filepath.Join(dir, storage.DriverKeyFile))).
				To(Equal("old key"))

			l = openLayer(ctx, dir, "0.59.0")
			value, closer := MustSucceed2(l.KV.Get(ctx, []byte("key")))
			Expect(string(value)).To(Equal("before"))
			Expect(closer.Close()).To(Succeed())
			Expect(MustSucceed(l.TS.RetrieveChannel(ctx, dataCh.Key)).Name).
				To(Equal("data"))
			Expect(l.TS.RetrieveChannel(ctx, addedCh.Key)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
			frame := MustSucceed(l.TS.Read(ctx, telem.TimeRangeMax, dataCh.Key))
			Expect(frame.Len()).To(BeEquivalentTo(6))
			Expect(l.TS.CreateChannel(ctx, addedCh)).To(Succeed())
			Expect(l.Close()).To(Succeed())
			Expect(MustSucceed(storage.ListBackups(dir))).To(HaveLen(1))
		},
	)

	It("Should remove the version of a backup without one", func(ctx SpecContext) {
		openAt(ctx, dir, "0.0.0")
		openAt(ctx, dir, "0.60.0")
		res := MustSucceed(storage.Restore(dir, filepath.Base(onlyBackup(dir))))
		Expect(res.Backup.Version).To(Equal(storage.UnknownVersion))
		Expect(res.OrphanedDir).To(BeEmpty())
		Expect(entries(filepath.Join(dir, "backups"))).To(ConsistOf(res.Backup.Name))
		Expect(filepath.Join(dir, "VERSION")).ToNot(BeAnExistingFile())
	})

	DescribeTable(
		"Should not open the data until an interrupted restore runs again",
		func(ctx SpecContext, interrupt func()) {
			openAt(ctx, dir, "0.59.0")
			l := openLayer(ctx, dir, "0.60.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
			Expect(l.Close()).To(Succeed())
			name := filepath.Base(onlyBackup(dir))
			writeFile(filepath.Join(dir, storage.RestoreMarkerFile), name)
			interrupt()
			before := entries(dir)
			Expect(storage.OpenLayer(
				ctx,
				storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
			)).Error().To(MatchError(ContainSubstring(
				"Run `synnax restore %s` again", name,
			)))
			Expect(entries(dir)).To(Equal(before))
			MustSucceed(storage.Restore(dir, name))
			Expect(entries(dir)).ToNot(ContainElement(
				Or(HavePrefix("kv."), Equal(storage.RestoreMarkerFile)),
			))
			Expect(entries(filepath.Join(dir, "kv"))).ToNot(ContainElement("stale"))
			l = openLayer(ctx, dir, "0.59.0")
			Expect(l.KV.Get(ctx, []byte("key"))).Error().
				To(MatchError(query.ErrNotFound))
			Expect(l.Close()).To(Succeed())
		},
		Entry("before it changed the data", func() {}),
		Entry("after it moved the live store aside", func() {
			Expect(os.Rename(filepath.Join(dir, "kv"), filepath.Join(dir, "kv.old"))).
				To(Succeed())
			Expect(os.MkdirAll(filepath.Join(dir, "kv.restore", "stale"), 0o700)).
				To(Succeed())
		}),
		Entry("before it removed the old store", func() {
			Expect(os.MkdirAll(filepath.Join(dir, "kv.old", "stale"), 0o700)).
				To(Succeed())
		}),
		Entry("while it copied the store", func() {
			Expect(os.MkdirAll(filepath.Join(dir, "kv.restore", "stale"), 0o700)).
				To(Succeed())
		}),
	)

	It("Should not open the data after a restore that failed", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		name := filepath.Base(onlyBackup(dir))
		blocked := filepath.Join(dir, "VERSION.tmp", "blocked")
		Expect(os.MkdirAll(blocked, 0o700)).To(Succeed())
		Expect(storage.Restore(dir, name)).Error().
			To(MatchError(ContainSubstring("VERSION.tmp")))
		Expect(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		)).Error().To(MatchError(ContainSubstring(
			"The restore of backup %s did not finish", name,
		)))
		Expect(os.RemoveAll(filepath.Dir(blocked))).To(Succeed())
		MustSucceed(storage.Restore(dir, name))
		openAt(ctx, dir, "0.59.0")
	})

	It("Should remove a Driver key that the backup lacks", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		writeFile(filepath.Join(dir, storage.DriverKeyFile), "new key")
		MustSucceed(storage.Restore(dir, filepath.Base(onlyBackup(dir))))
		Expect(filepath.Join(dir, storage.DriverKeyFile)).ToNot(BeAnExistingFile())
	})

	DescribeTable(
		"Should not restore a backup that is not complete",
		func(ctx SpecContext, name string) {
			openAt(ctx, dir, "0.60.0")
			partial := filepath.Join(dir, "backups", "0000000001-0.59.0.partial")
			Expect(os.MkdirAll(filepath.Join(partial, "kv"), 0o700)).To(Succeed())
			Expect(storage.Restore(dir, name)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(recordedVersion(dir)).To(Equal("0.60.0"))
			Expect(filepath.Join(dir, storage.RestoreMarkerFile)).
				ToNot(BeAnExistingFile())
		},
		Entry("missing", "0000000001-0.59.0"),
		Entry("unfinished", "0000000001-0.59.0.partial"),
	)

	It("Should not restore while a Core runs on the data", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		name := filepath.Base(onlyBackup(dir))
		MustOpen(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		))
		Expect(storage.Restore(dir, name)).Error().
			To(MatchError(ContainSubstring("Failed to acquire lock")))
	})

	It("Should return an error for a missing data directory", func() {
		Expect(storage.Restore(dir, "0000000001-0.59.0")).Error().
			To(MatchError(fs.ErrNotExist))
	})
})
