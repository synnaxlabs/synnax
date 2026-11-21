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
	"encoding/binary"
	"path"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// writes counts the writes to each file of a disk, by path from its root.
type writes struct {
	mu    sync.Mutex
	files map[string]int
}

func (w *writes) add(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.files[name]++
}

// under returns the count of writes to files under dir.
func (w *writes) under(dir string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for name, c := range w.files {
		if strings.HasPrefix(name, dir+"/") {
			n += c
		}
	}
	return n
}

// writesFS counts each write of a file that it opens, and passes every call to FS.
type writesFS struct {
	xfs.FS
	dir    string
	writes *writes
}

func (f writesFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return writesFile{File: file, name: path.Join(f.dir, name), writes: f.writes}, nil
}

func (f writesFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return writesFS{FS: sub, dir: path.Join(f.dir, name), writes: f.writes}, nil
}

type writesFile struct {
	xfs.File
	name   string
	writes *writes
}

func (f writesFile) Write(p []byte) (int, error) {
	f.writes.add(f.name)
	return f.File.Write(p)
}

func (f writesFile) WriteAt(p []byte, off int64) (int, error) {
	f.writes.add(f.name)
	return f.File.WriteAt(p, off)
}

const (
	layerIndex cesium.ChannelKey = 1
	layerData  cesium.ChannelKey = 2
)

// layerEngine maps each commit to samples of one Cesium channel and then to one key of
// the key-value store, with both stores of a storage layer on one disk.
type layerEngine struct {
	fs    xfs.FS
	layer *storage.Layer
	stop  func() error
}

func openLayerEngine(ctx context.Context, fs xfs.FS, w *writes) (*layerEngine, error) {
	counted := writesFS{FS: fs, writes: w}
	layer, err := storage.OpenLayer(ctx, storage.LayerConfig{
		Dirname: "data",
		Version: "0.0.0",
		KVFS:    pebblefs.New(counted),
		TSFS:    counted,
	})
	if err != nil {
		return nil, err
	}
	e := &layerEngine{fs: fs, layer: layer, stop: GoRun(ctx, layer.Run)}
	if _, err = layer.TS.RetrieveChannel(ctx, layerData); err == nil {
		return e, nil
	}
	if err = layer.TS.CreateChannel(
		ctx,
		cesium.Channel{
			Key: layerIndex, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		},
		cesium.Channel{
			Key: layerData, Name: "data", Index: layerIndex, DataType: telem.Uint8T,
		},
	); err != nil {
		return nil, errors.Join(err, e.Close())
	}
	return e, nil
}

func recordKey(id uint64) []byte {
	return binary.BigEndian.AppendUint64([]byte("crash/"), id)
}

func (e *layerEngine) Commit(
	ctx context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	start := telem.TimeStamp(r.ID) * telem.SecondTS
	stamps := make([]telem.TimeStamp, len(r.Payload))
	for i := range stamps {
		stamps[i] = start + telem.TimeStamp(i)
	}
	if err := e.layer.TS.Write(ctx, start, telem.MultiFrame(
		[]cesium.ChannelKey{layerIndex, layerData},
		[]telem.Series{telem.NewSeries(stamps), telem.NewSeries(r.Payload)},
	)); err != nil {
		return 0, err
	}
	// Cesium syncs no file, so the samples are durable only from here on.
	if err := crash.Writeback(e.fs); err != nil {
		return 0, err
	}
	if err := e.layer.KV.Set(ctx, recordKey(r.ID), r.Payload); err != nil {
		return 0, err
	}
	return crash.Durable, nil
}

// Recovered returns the commits whose key the key-value store holds, each with the
// payload that Cesium holds for it.
func (e *layerEngine) Recovered(ctx context.Context) ([]crash.Record, error) {
	iter, err := e.layer.KV.OpenIterator(kv.IterPrefix([]byte("crash/")))
	if err != nil {
		return nil, err
	}
	var ids []uint64
	for ok := iter.First(); ok; ok = iter.Next() {
		ids = append(ids, binary.BigEndian.Uint64(iter.Key()[len("crash/"):]))
	}
	if err = errors.Join(iter.Error(), iter.Close()); err != nil {
		return nil, err
	}
	records := make([]crash.Record, 0, len(ids))
	for _, id := range ids {
		start := telem.TimeStamp(id) * telem.SecondTS
		frame, err := e.layer.TS.Read(ctx, start.Range(start+telem.SecondTS), layerData)
		if err != nil {
			return nil, err
		}
		var payload []byte
		for _, s := range frame.SeriesSlice() {
			payload = append(payload, s.Data...)
		}
		records = append(records, crash.Record{ID: id, Payload: payload})
	}
	return records, nil
}

func (e *layerEngine) Close() error { return errors.Join(e.stop(), e.layer.Close()) }

var _ = Describe("Crash harness on the storage layer", func() {
	It("Should count the writes of the key-value store and of Cesium on one disk",
		func(ctx SpecContext) {
			// Only the writes of the workload count, not those of the recoveries.
			var (
				workload = &writes{files: map[string]int{}}
				opens    int
			)
			res, err := crash.Run(ctx, crash.Config{
				Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
					opens++
					if opens == 1 {
						return openLayerEngine(ctx, fs, workload)
					}
					return openLayerEngine(ctx, fs, &writes{files: map[string]int{}})
				},
				Commits:    4,
				Stride:     1,
				Profile:    crash.Strict,
				Exhaustive: true,
			})
			Expect(err).ToNot(HaveOccurred())
			kvWrites, tsWrites := workload.under(
				"data/kv",
			), workload.under(
				"data/cesium",
			)
			Expect(kvWrites).To(BeNumerically(">", 0))
			Expect(tsWrites).To(BeNumerically(">", 0))
			GinkgoWriter.Printf("kv %d, cesium %d, recoveries %d\n", kvWrites, tsWrites,
				res.Recoveries)
			// Each write is an event, and the run opens one crash image per event.
			Expect(res.Recoveries).To(BeNumerically(">=", kvWrites+tsWrites))
		},
	)
})
