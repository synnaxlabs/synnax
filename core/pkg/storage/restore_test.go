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

const restoreMarkerFile = "RESTORING"

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

			l = openLayer(ctx, dir, "0.60.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
			Expect(l.TS.RenameChannels(
				ctx,
				map[cesium.ChannelKey]string{dataCh.Key: "renamed"},
			)).To(Succeed())
			Expect(l.TS.CreateChannel(ctx, addedCh)).To(Succeed())
			write(ctx, l, 10, indexCh.Key, dataCh.Key, addedCh.Key)
			Expect(l.Close()).To(Succeed())

			name := filepath.Base(onlyBackup(dir))
			res := MustSucceed(newBackups(dir).Restore(name))
			Expect(res.Backup.Name).To(Equal(name))
			Expect(res.Backup.Version).To(Equal("0.59.0"))
			Expect(res.OrphanedDir).
				To(HavePrefix(filepath.Join(dir, "backups", "orphaned-")))
			Expect(entries(res.OrphanedDir)).To(HaveLen(1))
			Expect(recordedVersion(dir)).To(Equal("0.59.0"))

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
			Expect(MustSucceed(newBackups(dir).List())).To(HaveLen(1))
		},
	)

	It("Should remove the version of a backup without one", func(ctx SpecContext) {
		openAt(ctx, dir, "0.0.0")
		openAt(ctx, dir, "0.60.0")
		res := MustSucceed(newBackups(dir).Restore(filepath.Base(onlyBackup(dir))))
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
			writeFile(filepath.Join(dir, restoreMarkerFile), name)
			interrupt()
			before := entries(dir)
			Expect(storage.OpenLayer(
				ctx,
				storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
			)).Error().To(MatchError(ContainSubstring(
				"Run `synnax update undo %s` again", name,
			)))
			Expect(entries(dir)).To(Equal(before))
			MustSucceed(newBackups(dir).Restore(name))
			Expect(entries(dir)).ToNot(ContainElement(
				Or(HavePrefix("kv."), Equal(restoreMarkerFile)),
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

	It(
		"Should move channels that an interrupted restore set aside",
		func(ctx SpecContext) {
			openAt(ctx, dir, "0.59.0")
			openAt(ctx, dir, "0.60.0")
			name := filepath.Base(onlyBackup(dir))
			writeFile(filepath.Join(dir, restoreMarkerFile), name)
			Expect(os.MkdirAll(filepath.Join(dir, "cesium", "orphaned", "9"), 0o700)).
				To(Succeed())
			res := MustSucceed(newBackups(dir).Restore(name))
			Expect(
				entries(filepath.Join(dir, "cesium")),
			).ToNot(ContainElement("orphaned"))
			Expect(entries(res.OrphanedDir)).To(ConsistOf("9"))
		},
	)

	It("Should not open the data after a restore that failed", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		name := filepath.Base(onlyBackup(dir))
		blocked := filepath.Join(dir, "VERSION.tmp", "blocked")
		Expect(os.MkdirAll(blocked, 0o700)).To(Succeed())
		Expect(newBackups(dir).Restore(name)).Error().
			To(MatchError(ContainSubstring("VERSION.tmp")))
		Expect(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		)).Error().To(MatchError(ContainSubstring(
			"The restore of backup %s did not finish", name,
		)))
		Expect(os.RemoveAll(filepath.Dir(blocked))).To(Succeed())
		MustSucceed(newBackups(dir).Restore(name))
		openAt(ctx, dir, "0.59.0")
	})

	DescribeTable(
		"Should not restore a backup that is not complete",
		func(ctx SpecContext, name string) {
			openAt(ctx, dir, "0.60.0")
			partial := filepath.Join(dir, "backups", "0000000001-0.59.0.partial")
			Expect(os.MkdirAll(filepath.Join(partial, "kv"), 0o700)).To(Succeed())
			Expect(newBackups(dir).Restore(name)).Error().
				To(MatchError(query.ErrNotFound))
			Expect(recordedVersion(dir)).To(Equal("0.60.0"))
			Expect(filepath.Join(dir, restoreMarkerFile)).
				ToNot(BeAnExistingFile())
		},
		Entry("missing", "0000000001-0.59.0"),
		Entry("unfinished", "0000000001-0.59.0.partial"),
	)

	DescribeTable(
		"Should not change the data when the backup lacks a directory",
		func(ctx SpecContext, missing string) {
			openAt(ctx, dir, "0.59.0")
			l := openLayer(ctx, dir, "0.60.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
			Expect(l.TS.CreateChannel(ctx, indexCh)).To(Succeed())
			Expect(l.Close()).To(Succeed())
			backup := onlyBackup(dir)
			Expect(os.RemoveAll(filepath.Join(backup, missing))).To(Succeed())
			Expect(newBackups(dir).Restore(filepath.Base(backup))).Error().
				To(MatchError(ContainSubstring("has no %s directory", missing)))
			Expect(entries(backup)).ToNot(ContainElement(missing))
			Expect(filepath.Join(dir, restoreMarkerFile)).
				ToNot(BeAnExistingFile())
			l = openLayer(ctx, dir, "0.60.0")
			value, closer := MustSucceed2(l.KV.Get(ctx, []byte("key")))
			Expect(string(value)).To(Equal("after"))
			Expect(closer.Close()).To(Succeed())
			Expect(MustSucceed(l.TS.RetrieveChannel(ctx, indexCh.Key)).Name).
				To(Equal("time"))
			Expect(l.Close()).To(Succeed())
		},
		Entry("key-value store", "kv"),
		Entry("channel metadata", "cesium"),
	)

	It("Should not restore while a Core runs on the data", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		name := filepath.Base(onlyBackup(dir))
		MustOpen(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		))
		Expect(newBackups(dir).Restore(name)).Error().
			To(MatchError(ContainSubstring("Failed to acquire lock")))
	})
})

var _ = Describe("Undo", func() {
	var dir string
	BeforeEach(func() { dir = newDataDir() })

	It("Should restore the newest backup", func(ctx SpecContext) {
		openAt(ctx, dir, "0.58.0")
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		res := MustSucceed(newBackups(dir).Undo())
		Expect(res.Backup.Version).To(Equal("0.59.0"))
		Expect(recordedVersion(dir)).To(Equal("0.59.0"))
	})

	It("Should not undo the same update twice", func(ctx SpecContext) {
		openAt(ctx, dir, "0.59.0")
		openAt(ctx, dir, "0.60.0")
		name := MustSucceed(newBackups(dir).Undo()).Backup.Name
		Expect(newBackups(dir).Undo()).Error().To(MatchError(ContainSubstring(
			"The last update is already undone. To restore backup %s again", name,
		)))
		MustSucceed(newBackups(dir).Restore(name))
	})

	It("Should return an error when there is no backup", func(ctx SpecContext) {
		openAt(ctx, dir, "0.60.0")
		Expect(newBackups(dir).Undo()).Error().
			To(MatchError(ContainSubstring("there is no update to undo")))
	})
})
