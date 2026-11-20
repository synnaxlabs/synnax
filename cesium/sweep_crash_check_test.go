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
	"fmt"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// snapFS calls onEvent after each call that changes the disk.
type snapFS struct {
	inner   xfs.FS
	onEvent func()
}

func (s *snapFS) Open(name string, flag int) (xfs.File, error) {
	f, err := s.inner.Open(name, flag)
	if err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		s.onEvent()
	}
	return &snapFile{File: f, onEvent: s.onEvent}, nil
}

func (s *snapFS) Sub(name string) (xfs.FS, error) {
	sub, err := s.inner.Sub(name)
	if err != nil {
		return nil, err
	}
	s.onEvent()
	return &snapFS{inner: sub, onEvent: s.onEvent}, nil
}

func (s *snapFS) List(name string) ([]xfs.FileInfo, error) { return s.inner.List(name) }
func (s *snapFS) Exists(name string) (bool, error) {
	return s.inner.Exists(name)
}
func (s *snapFS) Stat(name string) (xfs.FileInfo, error) { return s.inner.Stat(name) }

func (s *snapFS) Remove(name string) error {
	err := s.inner.Remove(name)
	s.onEvent()
	return err
}

func (s *snapFS) Rename(o, n string) error {
	err := s.inner.Rename(o, n)
	s.onEvent()
	return err
}

func (s *snapFS) SyncDir(name string) error {
	err := s.inner.SyncDir(name)
	s.onEvent()
	return err
}

type snapFile struct {
	xfs.File
	onEvent func()
}

func (f *snapFile) Write(p []byte) (int, error) {
	n, err := f.File.Write(p)
	f.onEvent()
	return n, err
}

func (f *snapFile) WriteAt(p []byte, off int64) (int, error) {
	n, err := f.File.WriteAt(p, off)
	f.onEvent()
	return n, err
}

func (f *snapFile) Truncate(size int64) error {
	err := f.File.Truncate(size)
	f.onEvent()
	return err
}

func (f *snapFile) Sync() error {
	err := f.File.Sync()
	f.onEvent()
	return err
}

// syncAll makes every file and directory under dir durable.
func syncAll(f *fault.FS, dir string) {
	GinkgoHelper()
	for _, i := range MustSucceed(f.List(dir)) {
		p := path.Join(dir, i.Name())
		if i.IsDir() {
			syncAll(f, p)
			continue
		}
		h := MustSucceed(f.Open(p, os.O_RDWR))
		Expect(h.Sync()).To(Succeed())
		Expect(h.Close()).To(Succeed())
	}
	Expect(f.SyncDir(dir)).To(Succeed())
}

var _ = Describe("Sweep crash check", func() {
	It("Should open after a crash in a Sweep that drops a channel with no data", func(
		ctx SpecContext,
	) {
		var failures []string
		for runKey := range uint64(48) {
			disk := MustSucceed(fault.New(fault.Config{}))
			db := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(disk)))
			idx, empty := cesium.ChannelKey(1), cesium.ChannelKey(2)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      empty,
					Name:     "empty",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
			)).To(Succeed())
			Expect(commitFrame(ctx, db, 10*telem.SecondTS, telem.UnaryFrame(
				idx,
				telem.NewSeriesSecondsTSV(10, 11, 12),
			))).To(Succeed())
			Expect(db.Close()).To(Succeed())
			syncAll(disk, "")
			var (
				images []*fault.FS
				armed  bool
			)
			wrapped := &snapFS{inner: disk, onEvent: func() {
				if !armed {
					return
				}
				images = append(images, MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:            runKey,
					EntrySurvivalRate: 0.5,
					DataSurvivalRate:  0.5,
				})))
			}}
			db = MustSucceed(cesium.Open(ctx, "", cesium.WithFS(wrapped)))
			armed = true
			Expect(keptOf(db.Sweep(ctx, []cesium.ChannelKey{idx}))).To(BeEmpty())
			armed = false
			Expect(db.Close()).To(Succeed())
			Expect(images).ToNot(BeEmpty())
			for i, image := range images {
				re, err := cesium.Open(ctx, "", cesium.WithFS(image))
				if err != nil {
					var names []string
					for _, d := range MustSucceed(image.List("")) {
						names = append(names, d.Name())
						if d.IsDir() {
							for _, f := range MustSucceed(image.List(d.Name())) {
								names = append(names, d.Name()+"/"+f.Name())
							}
						}
					}
					failures = append(failures, fmt.Sprintf(
						"run key %d, image %d: %v; files %v", runKey, i, err, names,
					))
					continue
				}
				Expect(keptOf(re.Sweep(ctx, []cesium.ChannelKey{idx}))).To(BeEmpty())
				Expect(re.Close()).To(Succeed())
			}
		}
		Expect(failures).To(BeEmpty())
	})

	It("Should find no corrupt intent after a crash in an append", func(
		ctx SpecContext,
	) {
		var failures []uint64
		for runKey := range uint64(48) {
			disk := MustSucceed(fault.New(fault.Config{}))
			var (
				images []*fault.FS
				armed  bool
			)
			wrapped := &snapFS{inner: disk, onEvent: func() {
				if !armed {
					return
				}
				images = append(images, MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:           runKey,
					DataSurvivalRate: 0.5,
				})))
			}}
			db := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(wrapped)))
			first := make([]cesium.ChannelKey, 340)
			for i := range first {
				first[i] = cesium.ChannelKey(1000 + i)
			}
			Expect(db.Intend(first)).To(Succeed())
			armed = true
			Expect(db.Intend([]cesium.ChannelKey{2000, 2001, 2002, 2003})).
				To(Succeed())
			armed = false
			Expect(db.Close()).To(Succeed())
			for _, image := range images {
				re := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(image)))
				if re.CorruptIntents() != 0 {
					failures = append(failures, runKey)
				}
				Expect(re.Close()).To(Succeed())
			}
		}
		Expect(failures).To(BeEmpty())
	})

	It("Should open and sweep again after a crash in a Sweep that drops an intended "+
		"data channel", func(ctx SpecContext) {
		var (
			failures []string
			// emptied counts the images that hold the truncate of the intents file.
			emptied int
		)
		for runKey := range uint64(48) {
			disk := MustSucceed(fault.New(fault.Config{}))
			db := MustSucceed(cesium.Open(ctx, "", cesium.WithFS(disk)))
			idx, kept, dropped := cesium.ChannelKey(1), cesium.ChannelKey(2),
				cesium.ChannelKey(3)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: kept, Name: "a", Index: idx, DataType: telem.Int64T,
				},
				cesium.Channel{
					Key: dropped, Name: "b", Index: idx, DataType: telem.Int64T,
				},
			)).To(Succeed())
			Expect(commitFrame(ctx, db, 10*telem.SecondTS, telem.MultiFrame(
				[]cesium.ChannelKey{idx, kept, dropped},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
				},
			))).To(Succeed())
			Expect(db.Intend([]cesium.ChannelKey{dropped})).To(Succeed())
			Expect(db.Close()).To(Succeed())
			syncAll(disk, "")
			var (
				images []*fault.FS
				armed  bool
			)
			wrapped := &snapFS{inner: disk, onEvent: func() {
				if !armed {
					return
				}
				images = append(images, MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:            runKey,
					EntrySurvivalRate: 0.5,
					DataSurvivalRate:  0.5,
					TornRate:          0.5,
				})))
			}}
			records := []cesium.ChannelKey{idx, kept}
			db = MustSucceed(cesium.Open(ctx, "", cesium.WithFS(wrapped)))
			armed = true
			Expect(keptOf(db.Sweep(ctx, records))).To(BeEmpty())
			armed = false
			Expect(db.Close()).To(Succeed())
			for i, image := range images {
				if MustSucceed(image.Stat("delete-intents")).Size() == 0 {
					emptied++
				}
				re, err := cesium.Open(ctx, "", cesium.WithFS(image))
				if err == nil {
					var kept2 []cesium.KeptChannel
					kept2, _, err = re.Sweep(ctx, records)
					if err == nil && len(kept2) > 0 {
						err = fmt.Errorf("kept %v", kept2)
					}
				}
				if err == nil {
					if _, rErr := re.RetrieveChannel(ctx, dropped); rErr == nil {
						err = fmt.Errorf("dropped channel still open")
					}
				}
				if err == nil {
					var fr cesium.Frame
					if fr, err = re.Read(ctx, telem.TimeRangeMax, kept); err == nil &&
						fr.Get(kept).Len() != 3 {
						err = fmt.Errorf("kept channel holds %d samples",
							fr.Get(kept).Len())
					}
				}
				if re != nil {
					Expect(re.Close()).To(Succeed())
				}
				if err != nil {
					failures = append(failures, fmt.Sprintf(
						"run key %d, image %d: %v", runKey, i, err,
					))
				}
			}
		}
		Expect(failures).To(BeEmpty())
		Expect(emptied).To(BeNumerically(">", 0))
	})
})

// keptOf returns the kept channels of a Sweep that must succeed.
func keptOf(
	kept []cesium.KeptChannel,
	_ cesium.ChannelKey,
	err error,
) []cesium.KeptChannel {
	GinkgoHelper()
	Expect(err).ToNot(HaveOccurred())
	return kept
}
