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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	updateMarkerFile = "UPDATING"
	failedUpdateFile = "UPDATE_FAILED"
)

// startUpdate opens a Layer on dir as a Core of the given version without committing
// the update that the open starts.
func startUpdate(ctx context.Context, dir, version string) *storage.Layer {
	GinkgoHelper()
	l := MustSucceed(storage.OpenLayer(
		ctx,
		storage.LayerConfig{Dirname: dir, Version: version},
	))
	Expect(l.Update).ToNot(BeNil())
	return l
}

func readValue(ctx context.Context, l *storage.Layer) string {
	GinkgoHelper()
	value, closer := MustSucceed2(l.KV.Get(ctx, []byte("key")))
	defer func() { Expect(closer.Close()).To(Succeed()) }()
	return string(value)
}

func updateState(dir, version string) storage.UpdateState {
	GinkgoHelper()
	return MustSucceed(newBackups(dir).UpdateState(version))
}

var _ = Describe("Update", func() {
	var dir string
	BeforeEach(func(ctx SpecContext) {
		dir = newDataDir()
		l := openLayer(ctx, dir, "0.59.0")
		Expect(l.KV.Set(ctx, []byte("key"), []byte("before"))).To(Succeed())
		Expect(l.Close()).To(Succeed())
	})

	failedAt := func(version string) string {
		return "Update to Core " + version + " failed. The data is back at Core " +
			"0.59.0. Run Core 0.59.0, or fix the cause and run `synnax update retry`."
	}

	It("Should start an update that a commit keeps", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.Update.From).To(Equal("0.59.0"))
		Expect(l.Update.To).To(Equal("0.60.0"))
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdatePending))
		Expect(l.Update.Commit()).To(Succeed())
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateCommitted))
		Expect(l.Close()).To(Succeed())
		l = MustSucceed(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		))
		Expect(l.Update).To(BeNil())
		Expect(l.Close()).To(Succeed())
	})

	It("Should put the data back when the Core fails", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
		Expect(l.Close()).To(Succeed())
		Expect(l.Update.Undo(ctx, errors.New("no space left on device"))).To(And(
			beRefused(),
			MatchError("Update to Core 0.60.0 failed: no space left on device. The "+
				"data is back at Core 0.59.0. Run Core 0.59.0, or fix the cause and "+
				"run `synnax update retry`."),
		))
		Expect(recordedVersion(dir)).To(Equal("0.59.0"))
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateUndone))
		l = openLayer(ctx, dir, "0.59.0")
		Expect(readValue(ctx, l)).To(Equal("before"))
		Expect(l.Close()).To(Succeed())
	})

	It("Should put the data back without a failure on a stop", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
		Expect(l.Close()).To(Succeed())
		stopped, cancel := context.WithCancel(ctx)
		cancel()
		Expect(l.Update.Undo(stopped, context.Canceled)).
			To(MatchError(context.Canceled))
		Expect(recordedVersion(dir)).To(Equal("0.59.0"))
		Expect(filepath.Join(dir, failedUpdateFile)).ToNot(BeAnExistingFile())
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateNotStarted))
		l = openLayer(ctx, dir, "0.59.0")
		Expect(readValue(ctx, l)).To(Equal("before"))
		Expect(l.Close()).To(Succeed())
		Expect(startUpdate(ctx, dir, "0.60.0").Close()).To(Succeed())
	})

	It("Should refuse the failed version without a backup", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.Close()).To(Succeed())
		Expect(l.Update.Undo(ctx, errors.New("crashed"))).
			To(MatchError(HavePrefix("Update to Core 0.60.0 failed: crashed.")))
		Expect(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		)).Error().
			To(And(beRefused(), MatchError(failedAt("0.60.0"))))
		Expect(recordedVersion(dir)).To(Equal("0.59.0"))
		Expect(entries(filepath.Join(dir, "backups"))).To(HaveLen(1))
	})

	It("Should update to another version after a failed one", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.Close()).To(Succeed())
		Expect(l.Update.Undo(ctx, errors.New("crashed"))).
			To(MatchError(HavePrefix("Update to Core 0.60.0 failed: crashed.")))
		l = startUpdate(ctx, dir, "0.60.1")
		Expect(l.Update.From).To(Equal("0.59.0"))
		Expect(l.Close()).To(Succeed())
	})

	It("Should update again after a retry", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.Close()).To(Succeed())
		Expect(l.Update.Undo(ctx, errors.New("crashed"))).
			To(MatchError(HavePrefix("Update to Core 0.60.0 failed: crashed.")))
		Expect(newBackups(dir).Retry()).To(Equal("0.60.0"))
		Expect(filepath.Join(dir, failedUpdateFile)).ToNot(BeAnExistingFile())
		l = startUpdate(ctx, dir, "0.60.0")
		Expect(l.Close()).To(Succeed())
	})

	It("Should not retry when no update failed", func() {
		Expect(newBackups(dir).Retry()).Error().
			To(MatchError("there is no failed update to retry"))
	})

	It(
		"Should undo an update that an earlier start did not commit",
		func(ctx SpecContext) {
			l := startUpdate(ctx, dir, "0.60.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
			Expect(l.Close()).To(Succeed())
			Expect(storage.OpenLayer(
				ctx,
				storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
			)).Error().To(MatchError(failedAt("0.60.0")))
			Expect(filepath.Join(dir, updateMarkerFile)).ToNot(BeAnExistingFile())
			Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateUndone))
			l = openLayer(ctx, dir, "0.59.0")
			Expect(readValue(ctx, l)).To(Equal("before"))
			Expect(l.Close()).To(Succeed())
		},
	)

	It(
		"Should open the old version on data that an update did not commit",
		func(ctx SpecContext) {
			l := startUpdate(ctx, dir, "0.60.0")
			Expect(l.KV.Set(ctx, []byte("key"), []byte("after"))).To(Succeed())
			Expect(l.Close()).To(Succeed())
			l = openLayer(ctx, dir, "0.59.0")
			Expect(readValue(ctx, l)).To(Equal("before"))
			Expect(l.Close()).To(Succeed())
			Expect(MustSucceed(os.ReadFile(filepath.Join(dir, failedUpdateFile)))).
				To(BeEquivalentTo("0.60.0"))
		},
	)

	It("Should undo the update when the open fails after it", func(ctx SpecContext) {
		// A directory in place of the Pebble lock file fails the open of the store,
		// after the backup and the mark.
		lock := filepath.Join(dir, "kv", "LOCK")
		Expect(os.Remove(lock)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(lock, "blocked"), 0o700)).To(Succeed())
		Expect(storage.OpenLayer(
			ctx,
			storage.LayerConfig{Dirname: dir, Version: "0.60.0"},
		)).Error().To(MatchError(SatisfyAll(
			HavePrefix("Update to Core 0.60.0 failed: "),
			HaveSuffix(" The data is back at Core 0.59.0. Run Core 0.59.0, or "+
				"fix the cause and run `synnax update retry`."),
		)))
		Expect(recordedVersion(dir)).To(Equal("0.59.0"))
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateUndone))
	})

	It("Should end an update that a restore undoes", func(ctx SpecContext) {
		l := startUpdate(ctx, dir, "0.60.0")
		Expect(l.Close()).To(Succeed())
		MustSucceed(newBackups(dir).Restore(onlyBackup(dir)))
		Expect(filepath.Join(dir, updateMarkerFile)).ToNot(BeAnExistingFile())
		Expect(filepath.Join(dir, failedUpdateFile)).ToNot(BeAnExistingFile())
		openAt(ctx, dir, "0.59.0")
	})

	It("Should skip the backup and the undo when disabled", func(ctx SpecContext) {
		l := MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname:              dir,
			Version:              "0.60.0",
			UpdateBackupDisabled: new(true),
		}))
		Expect(l.Update).To(BeNil())
		Expect(l.Close()).To(Succeed())
		Expect(recordedVersion(dir)).To(Equal("0.60.0"))
		Expect(entries(filepath.Join(dir, "backups"))).To(BeEmpty())
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateCommitted))
	})

	It("Should not count a version that the data is not at", func() {
		Expect(updateState(dir, "0.60.0")).To(Equal(storage.UpdateNotStarted))
		Expect(updateState(dir, "0.59.0")).To(Equal(storage.UpdateCommitted))
	})
})
