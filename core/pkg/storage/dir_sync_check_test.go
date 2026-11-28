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
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

// dirPrefix is the key prefix of the records of the directory engine.
var dirPrefix = []byte("dir/")

func dirKey(id uint64) []byte {
	return binary.BigEndian.AppendUint64(bytes.Clone(dirPrefix), id)
}

// dirEngine keeps each commit as one key of the key-value store of a storage layer,
// with both stores of the layer on one disk under a storage directory that the first
// open makes.
type dirEngine struct {
	layer *storage.Layer
	stop  func() error
}

func openDirLayer(
	ctx context.Context,
	fs xfs.FS,
	dirname string,
) (*storage.Layer, error) {
	logger, err := alamos.NewLogger(alamos.LoggerConfig{ZapLogger: zap.NewNop()})
	if err != nil {
		return nil, err
	}
	return storage.OpenLayer(ctx, storage.LayerConfig{
		Instrumentation: alamos.New("crash", alamos.WithLogger(logger)),
		Dirname:         dirname,
		Version:         "0.0.0",
		KVFS:            pebblefs.New(fs),
		TSFS:            fs,
	})
}

func openDirEngine(dirname string) func(context.Context, xfs.FS) (crash.Engine, error) {
	return func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
		layer, err := openDirLayer(ctx, fs, dirname)
		if err != nil {
			return nil, err
		}
		return &dirEngine{layer: layer, stop: GoRun(ctx, layer.Run)}, nil
	}
}

func (e *dirEngine) Commit(
	ctx context.Context,
	r crash.Record,
	want crash.Ack,
) (crash.Ack, error) {
	if err := e.layer.KV.Set(ctx, dirKey(r.ID), r.Payload); err != nil {
		return 0, err
	}
	return want, nil
}

func (e *dirEngine) Recovered(context.Context) ([]crash.Record, error) {
	iter, err := e.layer.KV.OpenIterator(kv.IterPrefix(dirPrefix))
	if err != nil {
		return nil, err
	}
	var records []crash.Record
	for ok := iter.First(); ok; ok = iter.Next() {
		id := binary.BigEndian.Uint64(iter.Key()[len(dirPrefix):])
		records = append(records, crash.Record{
			ID:      id,
			Payload: bytes.Clone(iter.Value()),
		})
	}
	return records, errors.Join(iter.Error(), iter.Close())
}

func (e *dirEngine) Close() error { return errors.Join(e.stop(), e.layer.Close()) }

var _ = Describe("Storage directory check", func() {
	DescribeTable("Should keep the stores of a new storage directory after a crash "+
		"that keeps only durable state",
		func(ctx SpecContext, dirname string) {
			disk := MustSucceed(fault.New(fault.Config{}))
			layer := MustSucceed(openDirLayer(ctx, disk, dirname))
			stop := GoRun(ctx, layer.Run)
			Expect(layer.KV.Set(ctx, dirKey(1), []byte{7})).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(errors.Join(stop(), layer.Close())).To(Succeed())
			dirs := []string{
				dirname,
				path.Join(dirname, "kv"),
				path.Join(dirname, "cesium"),
			}
			for _, dir := range dirs {
				Expect(image.Exists(dir)).To(BeTrue(), dir)
			}
			layer = MustSucceed(openDirLayer(ctx, image, dirname))
			stop = GoRun(ctx, layer.Run)
			value, closer := MustSucceed2(layer.KV.Get(dirKey(1)))
			Expect(value).To(Equal([]byte{7}))
			Expect(errors.Join(closer.Close(), stop(), layer.Close())).To(Succeed())
		},
		Entry("in the root of the disk", "data"),
		Entry("with a new parent", "parent/data"),
	)
})

// Each entry holds when a storage layer opens its stores in a storage directory that
// the first open makes, and each commit is one synced key of the key-value store. A
// partial crash can leave the key-value directory with no Pebble marker, and the
// migration check of the layer then reads the OS file system, not KVFS. A failed sync
// or write of Pebble's log ends the process. So only Strict runs.
var _ = DescribeTable("Storage directory crash check",
	func(ctx SpecContext, dirname string, profile crash.Profile) {
		MustSucceed(crash.Run(ctx, crash.Config{
			Open:    openDirEngine(dirname),
			Commits: 6,
			Stride:  1,
			Profile: profile,
		}))
	},
	Entry("Strict", "data", crash.Strict),
	Entry("Strict, with a new parent", "parent/data", crash.Strict),
)
