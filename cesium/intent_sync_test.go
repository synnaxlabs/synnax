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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

var errSync = errors.New("sync failed")

// failSync is a file system whose files named name fail their Sync while fail is set.
type failSync struct {
	xfs.FS
	name string
	fail *atomic.Bool
}

func (f failSync) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || name != f.name {
		return file, err
	}
	return failSyncFile{File: file, fail: f.fail}, nil
}

// Sub returns the sub file system at name, whose files named f.name fail as f's do.
func (f failSync) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return failSync{FS: sub, name: f.name, fail: f.fail}, nil
}

type failSyncFile struct {
	xfs.File
	fail *atomic.Bool
}

func (f failSyncFile) Sync() error {
	if f.fail.Load() {
		return errSync
	}
	return f.File.Sync()
}

var _ = Describe("Delete intent sync", func() {
	It("Should refuse every Intend after a failed sync of the intents file",
		func(ctx SpecContext) {
			var fail atomic.Bool
			db := mustOpenDBOnFS(
				ctx,
				failSync{FS: xfs.NewMem(), name: "delete-intents", fail: &fail},
			)
			first, second := GenerateChannelKey(), GenerateChannelKey()
			for _, k := range []cesium.ChannelKey{first, second} {
				Expect(db.CreateChannel(ctx, cesium.Channel{
					Key: k, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				})).To(Succeed())
			}
			fail.Store(true)
			Expect(db.Intend([]cesium.ChannelKey{first})).To(MatchError(errSync))
			// After a failed sync, the bytes of the file are unknown, and a later sync
			// can pass without making them durable.
			fail.Store(false)
			Expect(db.Intend([]cesium.ChannelKey{second})).To(SatisfyAll(
				MatchError(errSync),
				MatchError(ContainSubstring("until a restart")),
			))
		},
	)
})
