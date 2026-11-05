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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage"
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

func writeFile(path, data string) {
	GinkgoHelper()
	Expect(os.WriteFile(path, []byte(data), 0o600)).To(Succeed())
}

func newBackups(dir string) *storage.Backups {
	GinkgoHelper()
	return MustSucceed(storage.NewBackups(storage.BackupsConfig{Dirname: dir}))
}

func onlyBackup(dir string) string {
	GinkgoHelper()
	backups := MustSucceed(newBackups(dir).List())
	Expect(backups).To(HaveLen(1))
	return filepath.Join(dir, "backups", backups[0].Name)
}

var _ = Describe("Backup", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It(
		"Should copy the key-value store and the channel metadata",
		func(ctx SpecContext) {
			l := openLayer(ctx, dir, "0.59.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("value"))).To(Succeed())
			Expect(l.TS.CreateChannel(ctx, indexCh, dataCh)).To(Succeed())
			write(ctx, l, 1, indexCh.Key, dataCh.Key)
			Expect(l.Close()).To(Succeed())
			openAt(ctx, dir, "0.60.0")

			backup := onlyBackup(dir)
			Expect(entries(backup)).To(ConsistOf("kv", "cesium"))
			Expect(entries(filepath.Join(backup, "kv"))).ToNot(BeEmpty())
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

	It(
		"Should remove unfinished backups before it backs up again",
		func(ctx SpecContext) {
			openAt(ctx, dir, "0.59.0")
			stale := filepath.Join(dir, "backups", "0000000001-0.58.0.partial")
			Expect(os.MkdirAll(stale, 0o700)).To(Succeed())
			// A directory in place of a metadata file makes the copy fail.
			Expect(os.MkdirAll(filepath.Join(dir, "cesium", "9", "meta.json"), 0o700)).
				To(Succeed())
			Expect(storage.OpenLayer(
				ctx,
				storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
			)).Error().To(MatchError(ContainSubstring("failed to back up metadata")))
			Expect(entries(filepath.Join(dir, "backups"))).
				To(ConsistOf(HaveSuffix("-0.59.0.partial")))
		},
	)
})

var _ = Describe("Backups", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It("Should return an error for a missing data directory", func() {
		Expect(storage.NewBackups(storage.BackupsConfig{Dirname: dir})).Error().
			To(MatchError(fs.ErrNotExist))
	})

	It("Should return nothing when there are no backups", func() {
		Expect(os.MkdirAll(dir, 0o700)).To(Succeed())
		Expect(MustSucceed(newBackups(dir).List())).To(BeEmpty())
	})

	It("Should list only complete backups, oldest first", func() {
		backups := filepath.Join(dir, "backups")
		for _, name := range []string{
			"0000000002-0.60.0-rc.1",
			"0000000001-unknown",
			"0000000003-0.60.0.partial",
			"orphaned-4",
		} {
			Expect(os.MkdirAll(filepath.Join(backups, name), 0o700)).To(Succeed())
		}
		Expect(os.WriteFile(filepath.Join(backups, "0000000005-0.61.0"), nil, 0o600)).
			To(Succeed())
		Expect(MustSucceed(newBackups(dir).List())).To(Equal([]storage.Backup{
			{Name: "0000000001-unknown", Time: time.Unix(1, 0), Version: "unknown"},
			{
				Name:    "0000000002-0.60.0-rc.1",
				Time:    time.Unix(2, 0),
				Version: "0.60.0-rc.1",
			},
		}))
	})
})
