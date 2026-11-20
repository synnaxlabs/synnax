// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// eventFS calls onEvent after each call that can change the disk.
type eventFS struct {
	xfs.FS
	onEvent func()
}

func (e eventFS) Open(name string, flag int) (xfs.File, error) {
	f, err := e.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		e.onEvent()
	}
	return eventFile{File: f, onEvent: e.onEvent}, nil
}

func (e eventFS) Sub(name string) (xfs.FS, error) {
	sub, err := e.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	e.onEvent()
	return eventFS{FS: sub, onEvent: e.onEvent}, nil
}

func (e eventFS) Remove(name string) error {
	defer e.onEvent()
	return e.FS.Remove(name)
}

func (e eventFS) Rename(from, to string) error {
	defer e.onEvent()
	return e.FS.Rename(from, to)
}

func (e eventFS) SyncDir(name string) error {
	defer e.onEvent()
	return e.FS.SyncDir(name)
}

type eventFile struct {
	xfs.File
	onEvent func()
}

func (f eventFile) Write(p []byte) (int, error) {
	defer f.onEvent()
	return f.File.Write(p)
}

func (f eventFile) WriteAt(p []byte, off int64) (int, error) {
	defer f.onEvent()
	return f.File.WriteAt(p, off)
}

func (f eventFile) Truncate(size int64) error {
	defer f.onEvent()
	return f.File.Truncate(size)
}

func (f eventFile) Sync() error {
	defer f.onEvent()
	return f.File.Sync()
}

// syncAll makes each file and directory under dir durable.
func syncAll(disk *fault.FS, dir string) {
	GinkgoHelper()
	for _, info := range MustSucceed(disk.List(dir)) {
		p := path.Join(dir, info.Name())
		if info.IsDir() {
			syncAll(disk, p)
			continue
		}
		f := MustSucceed(disk.Open(p, os.O_RDWR))
		Expect(f.Sync()).To(Succeed())
		Expect(f.Close()).To(Succeed())
	}
	Expect(disk.SyncDir(dir)).To(Succeed())
}

var _ = Describe("Crash adapter, purge", func() {
	It("Should open each crash image of a purge, with the data whole or gone",
		func(ctx SpecContext) {
			const key = cesium.ChannelKey(60_000)
			db := DeferClose(gorp.Wrap(memkv.New()))
			var failures []string
			for runKey := range uint64(32) {
				disk := MustSucceed(fault.New(fault.Config{}))
				ts := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(disk)))
				Expect(ts.CreateChannel(ctx, cesium.Channel{
					Key: key, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				})).To(Succeed())
				Expect(ts.WriteSeries(
					ctx,
					key,
					10*telem.SecondTS,
					telem.NewSeriesSecondsTSV(10, 11, 12),
				)).To(Succeed())
				Expect(ts.Close()).To(Succeed())
				syncAll(disk, "")
				var (
					images []*fault.FS
					armed  bool
				)
				ts = MustSucceed(cesium.Open(ctx, "", cesium.WithFS(eventFS{
					FS: disk,
					onEvent: func() {
						if armed {
							images = append(
								images,
								MustSucceed(disk.Crash(fault.CrashConfig{
									RunKey:            runKey,
									EntrySurvivalRate: 0.5,
									DataSurvivalRate:  0.5,
								})),
							)
						}
					},
				})))
				whole := MustSucceed(ts.Sweep(ctx, nil))
				Expect(whole).To(HaveLen(1))
				armed = true
				Expect(channel.Purge(ctx, db, ts, []channel.Key{channel.Key(key)})).
					To(BeEmpty())
				armed = false
				Expect(ts.Close()).To(Succeed())
				Expect(images).ToNot(BeEmpty())
				for i, image := range images {
					re, err := cesium.Open(ctx, "", cesium.WithFS(image))
					if err != nil {
						failures = append(failures, fmt.Sprintf(
							"run key %d, image %d: %v", runKey, i, err,
						))
						continue
					}
					kept := MustSucceed(re.Sweep(ctx, nil))
					if len(kept) > 0 && (len(kept) > 1 || kept[0] != whole[0]) {
						failures = append(failures, fmt.Sprintf(
							"run key %d, image %d: kept %v, want none or %v",
							runKey, i, kept, whole,
						))
					}
					Expect(re.Close()).To(Succeed())
				}
			}
			Expect(failures).To(BeEmpty())
		},
	)
})
