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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/alignment"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func transientChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:       key,
		Name:      name,
		DataType:  telem.Int64T,
		Virtual:   true,
		Transient: true,
	}
}

var _ = Describe("Transient Channels", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			ShouldNotLeakGoroutinesPerSpec()
			var (
				db  *cesium.DB
				xFS fs.FS
			)
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				xFS = openFS()
				db = openDBOnFS(ctx, xFS)
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
			})

			Describe("Create", func() {
				It("Should create and retrieve a transient virtual channel", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(key, "ephemeral"))).To(Succeed())
					ch := MustSucceed(db.RetrieveChannel(ctx, key))
					Expect(ch.Name).To(Equal("ephemeral"))
					Expect(ch.Transient).To(BeTrue())
				})

				It("Should reject a transient channel that is not virtual", func(ctx SpecContext) {
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:       GenerateChannelKey(),
						Name:      "not_virtual",
						DataType:  telem.TimeStampT,
						IsIndex:   true,
						Transient: true,
					})).To(MatchError(ContainSubstring("only virtual channels can be transient")))
				})

				It("Should not create a directory for the channel on the file system", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(key, "no_dir"))).To(Succeed())
					Expect(xFS.Exists(channelKeyToPath(key))).To(BeFalse())
				})
			})

			Describe("Write and Stream", func() {
				It("Should deliver written frames to streamers", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(key, "streamed"))).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    10 * telem.SecondTS,
					}))
					r := MustSucceed(db.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{key},
					}))
					i, o := confluence.Attach(r, 1)
					sCtx, cancel := signal.WithCancel(ctx)
					defer cancel()
					r.Flow(sCtx, confluence.CloseOutputInletsOnExit())

					d := telem.NewSeriesV[int64](1, 2, 3)
					MustSucceed(w.Write(telem.UnaryFrame(key, d)))

					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					Expect(res.Frame.Count()).To(Equal(1))
					d.Alignment = alignment.Leading(1, 0)
					Expect(res.Frame.SeriesAt(0)).To(Equal(d))
					i.Close()
					Expect(sCtx.Wait()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})

				It("Should validate series data types on write", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(key, "validated"))).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    10 * telem.SecondTS,
						Sync:     new(true),
					}))
					Expect(w.Write(telem.UnaryFrame(key, telem.NewSeriesV[float32](1, 2)))).
						Error().To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring("invalid data type")),
					))
					Expect(w.Close()).To(MatchError(ContainSubstring("invalid data type")))
				})
			})

			Describe("Delete", func() {
				It("Should delete a transient channel without touching the file system", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(key, "deleted"))).To(Succeed())
					Expect(db.DeleteChannel(key)).To(Succeed())
					Expect(db.RetrieveChannel(ctx, key)).Error().
						To(MatchError(cesium.ErrChannelNotFound))
				})
			})

			Describe("Rename", func() {
				It("Should rename a transient channel in memory", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(key, "old_name"))).To(Succeed())
					Expect(db.RenameChannel(ctx, key, "new_name")).To(Succeed())
					ch := MustSucceed(db.RetrieveChannel(ctx, key))
					Expect(ch.Name).To(Equal("new_name"))
					Expect(xFS.Exists(channelKeyToPath(key))).To(BeFalse())
				})
			})

			Describe("Rekey", func() {
				It("Should rekey a transient channel in memory", func(ctx SpecContext) {
					oldKey := GenerateChannelKey()
					newKey := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, transientChannel(oldKey, "rekeyed"))).To(Succeed())
					Expect(db.RekeyChannel(ctx, oldKey, newKey)).To(Succeed())
					ch := MustSucceed(db.RetrieveChannel(ctx, newKey))
					Expect(ch.Name).To(Equal("rekeyed"))
					Expect(ch.Transient).To(BeTrue())
					Expect(db.RetrieveChannel(ctx, oldKey)).Error().
						To(MatchError(cesium.ErrChannelNotFound))
					Expect(xFS.Exists(channelKeyToPath(newKey))).To(BeFalse())
				})
			})

			Describe("Restart", func() {
				It("Should not survive a database reopen, while persistent virtual channels do", func(ctx SpecContext) {
					subFS := MustSucceed(xFS.Sub("restart"))
					restartDB := openDBOnFS(ctx, subFS)
					transientKey := GenerateChannelKey()
					persistentKey := GenerateChannelKey()
					Expect(restartDB.CreateChannel(ctx,
						transientChannel(transientKey, "gone_on_restart"),
						cesium.Channel{
							Key:      persistentKey,
							Name:     "kept_on_restart",
							DataType: telem.Int64T,
							Virtual:  true,
						},
					)).To(Succeed())
					Expect(restartDB.Close()).To(Succeed())

					restartDB = openDBOnFS(ctx, subFS)
					Expect(restartDB.RetrieveChannel(ctx, transientKey)).Error().
						To(MatchError(cesium.ErrChannelNotFound))
					ch := MustSucceed(restartDB.RetrieveChannel(ctx, persistentKey))
					Expect(ch.Name).To(Equal("kept_on_restart"))
					Expect(restartDB.Close()).To(Succeed())
				})

				It("Should persist renames of persistent virtual channels across reopens", func(ctx SpecContext) {
					subFS := MustSucceed(xFS.Sub("rename-restart"))
					restartDB := openDBOnFS(ctx, subFS)
					key := GenerateChannelKey()
					Expect(restartDB.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "before",
						DataType: telem.Int64T,
						Virtual:  true,
					})).To(Succeed())
					Expect(restartDB.RenameChannel(ctx, key, "after")).To(Succeed())
					Expect(restartDB.Close()).To(Succeed())

					restartDB = openDBOnFS(ctx, subFS)
					ch := MustSucceed(restartDB.RetrieveChannel(ctx, key))
					Expect(ch.Name).To(Equal("after"))
					Expect(restartDB.Close()).To(Succeed())
				})
			})
		})
	}

	Describe("FS Interaction", func() {
		It("Should perform no file system writes for a transient channel's full lifecycle", func(ctx SpecContext) {
			// Transient channels are defined as never being persisted; this test pins
			// that contract directly by recording every operation against the
			// underlying FS across create, write, rename, and delete.
			rec := fs.NewRecorder(fs.NewMem())
			db := openDBOnFS(ctx, rec)
			rec.Reset()

			key := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, transientChannel(key, "fs_check"))).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{key},
				Start:    10 * telem.SecondTS,
			}))
			MustSucceed(w.Write(telem.UnaryFrame(key, telem.NewSeriesV[int64](1, 2, 3))))
			Expect(w.Close()).To(Succeed())
			Expect(db.RenameChannel(ctx, key, "fs_check_renamed")).To(Succeed())
			Expect(db.DeleteChannel(key)).To(Succeed())
			Expect(db.Close()).To(Succeed())

			Expect(rec.Count(fs.MatchOp(fs.OpWrite, fs.OpWriteAt))).To(BeZero())
		})
	})
})
