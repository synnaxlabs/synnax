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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cockroachdb/pebble/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/testutil/oldlayout"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// eventFS calls onEvent after each call that changes what a crash keeps: a create, a
// truncate, a sync, a remove, a rename, and a directory sync.
type eventFS struct {
	xfs.FS
	onEvent func()
}

func (f eventFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		f.onEvent()
	}
	return eventFile{File: file, onEvent: f.onEvent}, nil
}

func (f eventFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	f.onEvent()
	return eventFS{FS: sub, onEvent: f.onEvent}, nil
}

func (f eventFS) Remove(name string) error {
	defer f.onEvent()
	return f.FS.Remove(name)
}

func (f eventFS) Rename(from, to string) error {
	defer f.onEvent()
	return f.FS.Rename(from, to)
}

func (f eventFS) SyncDir(name string) error {
	defer f.onEvent()
	return f.FS.SyncDir(name)
}

type eventFile struct {
	xfs.File
	onEvent func()
}

func (f eventFile) Truncate(size int64) error {
	defer f.onEvent()
	return f.File.Truncate(size)
}

func (f eventFile) Sync() error {
	defer f.onEvent()
	return f.File.Sync()
}

// openOnLog opens a Layer with Cesium on the pointer log, commits the update that the
// open starts, and runs the Layer. The returned close stops the run and closes it.
func openOnLog(
	ctx context.Context,
	cfg storage.LayerConfig,
) (*storage.Layer, func() error, error) {
	cfg.PointerLogEnabled = new(true)
	l, err := storage.OpenLayer(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	if l.Update != nil {
		if err = l.Update.Commit(); err != nil {
			return nil, nil, errors.Join(err, l.Close())
		}
	}
	stop := GoRun(ctx, l.Run)
	return l, func() error { return errors.Join(stop(), l.Close()) }, nil
}

// storedIdentity returns the Cesium identity that the key-value store of l holds.
func storedIdentity(l *storage.Layer) (uint64, error) {
	b, closer, err := l.KV.Get([]byte("cesium.identity"))
	if err != nil {
		return 0, err
	}
	if len(b) != 8 {
		return 0, errors.Join(errors.Newf("%d bytes", len(b)), closer.Close())
	}
	return binary.LittleEndian.Uint64(b), closer.Close()
}

// samplesOf returns the bytes of each channel that a read of l returns.
func samplesOf(
	ctx context.Context,
	l *storage.Layer,
) (map[cesium.ChannelKey][]byte, error) {
	fr, err := l.TS.Read(ctx, telem.TimeRangeMax, indexCh.Key, dataCh.Key)
	if err != nil {
		return nil, err
	}
	data := make(map[cesium.ChannelKey][]byte)
	for _, key := range []cesium.ChannelKey{indexCh.Key, dataCh.Key} {
		for _, s := range fr.Get(key).Series {
			data[key] = append(data[key], s.Data...)
		}
	}
	return data, nil
}

// undoneIn returns the names in the Cesium directory of fs, at dir, that an undo moved
// aside.
func undoneIn(fs xfs.FS, dir string) ([]string, error) {
	infos, err := fs.List(filepath.Join(dir, "cesium"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, info := range infos {
		if strings.HasSuffix(info.Name(), ".undone") {
			names = append(names, info.Name())
		}
	}
	return names, nil
}

// writeOld writes the data directory that a Core of v0.62.0 leaves in dir of fs after
// write from each of starts to indexCh and dataCh: its empty key-value store, its
// VERSION, and the old layout of Cesium, with one file for each key and one pointer for
// each write. It returns the bytes of each channel that a read returns.
func writeOld(fs xfs.FS, dir string, starts ...int) map[cesium.ChannelKey][]byte {
	GinkgoHelper()
	// Sub makes the directory, which the open of the store needs.
	MustSucceed(fs.Sub(filepath.Join(dir, "kv")))
	kv := MustSucceed(pebble.Open(filepath.Join(dir, "kv"), &pebble.Options{
		FS:                 pebblefs.New(fs),
		FormatMajorVersion: pebble.FormatNewest,
	}))
	Expect(kv.Close()).To(Succeed())
	data := MustSucceed(fs.Sub(dir))
	Expect(xfs.WriteFile(data, "VERSION", []byte("0.62.0"))).To(Succeed())
	keys := []oldlayout.Key{
		{Channel: indexCh, Counter: 1},
		{Channel: dataCh, Counter: 1},
	}
	var stamps, values []byte
	for _, start := range starts {
		tr := telem.TimeRange{
			Start: telem.TimeStamp(start) * telem.SecondTS,
			End:   telem.TimeStamp(start+2)*telem.SecondTS + 1,
		}
		for i := range keys {
			keys[i].Pointers = append(keys[i].Pointers, oldlayout.Pointer{
				TimeRange: tr,
				File:      1,
				Offset:    uint32(len(stamps)),
				Size:      24,
			})
		}
		stamps = append(stamps, telem.NewSeriesSecondsTSV(
			telem.TimeStamp(start),
			telem.TimeStamp(start+1),
			telem.TimeStamp(start+2),
		).Data...)
		values = append(values, telem.NewSeriesV(1.0, 2.0, 3.0).Data...)
	}
	keys[0].Files = map[uint16][]byte{1: stamps}
	keys[1].Files = map[uint16][]byte{1: values}
	Expect(oldlayout.Write(MustSucceed(data.Sub("cesium")), keys...)).To(Succeed())
	return map[cesium.ChannelKey][]byte{indexCh.Key: stamps, dataCh.Key: values}
}

var _ = Describe("Cesium identity", func() {
	It("Should store the identity of a new database and open with it", func(
		ctx SpecContext,
	) {
		dir := newDataDir()
		cfg := storage.LayerConfig{Dirname: dir, Version: "0.63.0"}
		l, closeLayer := MustSucceed2(openOnLog(ctx, cfg))
		id := l.TS.Identity()
		Expect(id).ToNot(BeZero())
		Expect(storedIdentity(l)).To(Equal(id))
		Expect(l.TS.CreateChannel(ctx, indexCh, dataCh)).To(Succeed())
		write(ctx, l, 1, indexCh.Key, dataCh.Key)
		written := MustSucceed(samplesOf(ctx, l))
		Expect(closeLayer()).To(Succeed())
		l, closeLayer = MustSucceed2(openOnLog(ctx, cfg))
		Expect(l.TS.Identity()).To(Equal(id))
		Expect(samplesOf(ctx, l)).To(Equal(written))
		Expect(closeLayer()).To(Succeed())
		Expect(undoneIn(xfs.Default, dir)).To(BeEmpty())
	})

	It("Should convert the old layout again after an undo of the update", func(
		ctx SpecContext,
	) {
		dir := newDataDir()
		before := writeOld(xfs.Default, dir, 1)
		cfg := storage.LayerConfig{Dirname: dir, Version: "0.63.0"}
		l, closeLayer := MustSucceed2(openOnLog(ctx, cfg))
		id := l.TS.Identity()
		write(ctx, l, 100, indexCh.Key, dataCh.Key)
		Expect(closeLayer()).To(Succeed())
		Expect(newBackups(dir).LosesTelemetry(onlyBackup(dir))).To(BeTrue())
		Expect(MustSucceed(newBackups(dir).Undo()).Backup.Version).To(Equal("0.62.0"))
		l, closeLayer = MustSucceed2(openOnLog(ctx, cfg))
		again := l.TS.Identity()
		Expect(again).ToNot(Equal(id))
		Expect(storedIdentity(l)).To(Equal(again))
		Expect(samplesOf(ctx, l)).To(Equal(before))
		Expect(closeLayer()).To(Succeed())
		Expect(undoneIn(xfs.Default, dir)).
			To(ContainElement(fmt.Sprintf("0000000000000001.head.%016x.undone", id)))
	})

	It("Should back up no Cesium file of the pointer log, and keep every Cesium file "+
		"at the undo", func(ctx SpecContext) {
		dir := newDataDir()
		l, closeLayer := MustSucceed2(openOnLog(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.63.0"},
		))
		id := l.TS.Identity()
		Expect(l.TS.CreateChannel(ctx, indexCh, dataCh)).To(Succeed())
		write(ctx, l, 1, indexCh.Key, dataCh.Key)
		Expect(closeLayer()).To(Succeed())
		l, closeLayer = MustSucceed2(openOnLog(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.64.0"},
		))
		write(ctx, l, 100, indexCh.Key, dataCh.Key)
		written := MustSucceed(samplesOf(ctx, l))
		Expect(closeLayer()).To(Succeed())
		Expect(archiveEntries(archiveFile(dir, onlyBackup(dir)))).
			To(HaveEach(HavePrefix("kv/")))
		files := entries(filepath.Join(dir, "cesium"))
		Expect(newBackups(dir).LosesTelemetry(onlyBackup(dir))).To(BeFalse())
		Expect(MustSucceed(newBackups(dir).Undo()).OrphanedDir).To(BeEmpty())
		Expect(entries(filepath.Join(dir, "cesium"))).To(Equal(files))
		l, closeLayer = MustSucceed2(openOnLog(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.63.0"},
		))
		Expect(l.TS.Identity()).To(Equal(id))
		Expect(samplesOf(ctx, l)).To(Equal(written))
		Expect(closeLayer()).To(Succeed())
	})

	It("Should convert again with every sample after a crash at each step of the "+
		"first open, between the commit point and the identity write included",
		func(ctx SpecContext) {
			disk := MustSucceed(fault.New(fault.Config{}))
			cfg := func(fs xfs.FS) storage.LayerConfig {
				return storage.LayerConfig{
					Dirname:              "data",
					Version:              "0.0.0",
					KVFS:                 pebblefs.New(fs),
					TSFS:                 fs,
					UpdateBackupDisabled: new(true),
				}
			}
			held := writeOld(disk, "data", 1, 100)
			// The old layout never syncs. Writeback makes its files durable, as time
			// does.
			disk.Writeback()
			var (
				mu       sync.Mutex
				taking   = true
				images   int
				aside    int
				failures []string
			)
			check := func(image *fault.FS) error {
				l, closeLayer, err := openOnLog(ctx, cfg(image))
				if err != nil {
					return err
				}
				data, err := samplesOf(ctx, l)
				id, idErr := storedIdentity(l)
				if err = errors.Join(err, idErr, closeLayer()); err != nil {
					return err
				}
				if !maps.EqualFunc(data, held, bytes.Equal) {
					return errors.New("the read differs from the old layout")
				}
				if id != l.TS.Identity() {
					return errors.Newf("stored identity %x, database %x", id,
						l.TS.Identity())
				}
				undone, err := undoneIn(image, "data")
				if len(undone) > 0 {
					aside++
				}
				return err
			}
			take := func() {
				mu.Lock()
				defer mu.Unlock()
				if !taking {
					return
				}
				image, err := disk.Crash(fault.CrashConfig{
					RunKey:            uint64(images),
					EntrySurvivalRate: 0.5,
					DataSurvivalRate:  0.5,
				})
				if err == nil {
					err = check(image)
				}
				if err != nil {
					failures = append(
						failures,
						fmt.Sprintf("image %d: %v", images, err),
					)
				}
				images++
			}
			_, closeLayer := MustSucceed2(
				openOnLog(ctx, cfg(eventFS{FS: disk, onEvent: take})),
			)
			mu.Lock()
			taking = false
			mu.Unlock()
			Expect(closeLayer()).To(Succeed())
			Expect(failures).To(BeEmpty())
			Expect(images).To(BeNumerically(">", 10))
			Expect(aside).To(BeNumerically(">", 0))
		})
})
