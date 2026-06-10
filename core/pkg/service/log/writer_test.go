// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Log", func(ctx SpecContext) {
			l := log.Log{
				Name: "test",
				Channels: []log.ChannelEntry{
					{Channel: channel.Key(1), Color: color.MustFromHex("#ff0000")},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(l.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should establish a ParentOf relationship to the project", func(ctx SpecContext) {
			l := log.Log{Name: "with-ws"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(otg.NewWriter(tx).HasRelationship(
				ctx,
				project.OntologyID(ws.Key),
				ontology.RelationshipTypeParentOf,
				log.OntologyID(l.Key),
			)).To(BeTrue())
		})

		It("Should skip the project ParentOf relationship when ws is uuid.Nil", func(ctx SpecContext) {
			l := log.Log{Name: "no-ws"}
			Expect(svc.NewWriter(tx).Create(ctx, uuid.Nil, &l)).To(Succeed())
			Expect(otg.NewWriter(tx).HasRelationship(
				ctx,
				project.OntologyID(ws.Key),
				ontology.RelationshipTypeParentOf,
				log.OntologyID(l.Key),
			)).To(BeFalse())
		})

		It("Should still register the resource in the ontology when ws is uuid.Nil", func(ctx SpecContext) {
			l := log.Log{Name: "orphan"}
			Expect(svc.NewWriter(tx).Create(ctx, uuid.Nil, &l)).To(Succeed())
			var resource ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(log.OntologyID(l.Key)).
				Entry(&resource).
				Exec(ctx, tx)).To(Succeed())
			Expect(resource.ID).To(Equal(log.OntologyID(l.Key)))
		})

		It("Should update the existing log when called with an existing key", func(ctx SpecContext) {
			key := uuid.New()
			first := log.Log{Key: key, Name: "first"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &first)).To(Succeed())
			second := log.Log{Key: key, Name: "second"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &second)).To(Succeed())
			var res log.Log
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("second"))
		})
	})
	Describe("CreateMany", func() {
		It("Should create multiple logs", func(ctx SpecContext) {
			logs := []log.Log{
				{Name: "log-1"},
				{Name: "log-2"},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, ws.Key, &logs)).To(Succeed())

			var retrieved []log.Log
			Expect(svc.NewRetrieve().Where(log.MatchKeys(
				logs[0].Key,
				logs[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Delete", func() {
		It("Should delete a Log so it is no longer retrievable", func(ctx SpecContext) {
			l := log.Log{Name: "to-delete"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&log.Log{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should remove the log's ontology resource", func(ctx SpecContext) {
			l := log.Log{Name: "to-delete-otg"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, l.Key)).To(Succeed())
			Expect(otg.NewRetrieve().
				WhereIDs(log.OntologyID(l.Key)).
				Entry(&ontology.Resource{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple logs in one call", func(ctx SpecContext) {
			a := log.Log{Name: "a"}
			b := log.Log{Name: "b"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &b)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(a.Key, b.Key)).
				Entries(&[]log.Log{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})
	Describe("Dispatch", func() {
		retrieve := func(ctx context.Context, key log.Key) log.Log {
			var res log.Log
			Expect(svc.NewRetrieve().Where(log.MatchKeys(key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			return res
		}
		Describe("per-action semantics", func() {
			It("Should apply Rename", func(ctx SpecContext) {
				l := log.Log{Name: "original"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewRenameAction(log.RenamePayload{Name: "renamed"}),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).Name).To(Equal("renamed"))
			})

			It("Should append a default channel entry via AddChannel", func(ctx SpecContext) {
				l := log.Log{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 42}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(1))
				entry := res.Channels[0]
				Expect(entry.Channel).To(Equal(channel.Key(42)))
				Expect(entry.Notation).To(Equal(notation.NotationStandard))
				Expect(entry.Precision).To(Equal(int32(-1)))
				Expect(entry.Timestamp.Format).To(Equal(telem.TimestampFormatPreciseDate))
				Expect(entry.Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
			})

			It("Should treat a duplicate AddChannel as a no-op", func(ctx SpecContext) {
				l := log.Log{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 42}),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 42}),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).Channels).To(HaveLen(1))
			})

			It("Should drop an entry via RemoveChannel", func(ctx SpecContext) {
				l := log.Log{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 1}),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 2}),
					log.NewRemoveChannelAction(log.RemoveChannelPayload{Channel: 1}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.Channels[0].Channel).To(Equal(channel.Key(2)))
			})

			It("Should insert then replace an entry via SetChannelEntry", func(ctx SpecContext) {
				l := log.Log{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetChannelEntryAction(log.SetChannelEntryPayload{
						Entry: log.ChannelEntry{Channel: 5, Alias: "first"},
					}),
					log.NewSetChannelEntryAction(log.SetChannelEntryPayload{
						Entry: log.ChannelEntry{Channel: 5, Alias: "second"},
					}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.Channels[0].Alias).To(Equal("second"))
			})

			It("Should replace the whole list via SetChannels", func(ctx SpecContext) {
				l := log.Log{Name: "test", Channels: []log.ChannelEntry{{Channel: 1}}}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetChannelsAction(log.SetChannelsPayload{
						Channels: []log.ChannelEntry{{Channel: 2}, {Channel: 3}},
					}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(2))
				Expect(res.Channels[0].Channel).To(Equal(channel.Key(2)))
				Expect(res.Channels[1].Channel).To(Equal(channel.Key(3)))
			})

			It("Should repoint a channel in place via SwapChannel, keeping position and config", func(ctx SpecContext) {
				l := log.Log{
					Name: "test",
					Channels: []log.ChannelEntry{
						{Channel: 1},
						{Channel: 2, Alias: "kept", Precision: 3},
						{Channel: 4},
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSwapChannelAction(log.SwapChannelPayload{From: 2, To: 5}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(3))
				Expect(res.Channels[1].Channel).To(Equal(channel.Key(5)))
				Expect(res.Channels[1].Alias).To(Equal("kept"))
				Expect(res.Channels[1].Precision).To(Equal(int32(3)))
			})

			It("Should treat SwapChannel as a no-op when the from channel is absent", func(ctx SpecContext) {
				l := log.Log{Name: "test", Channels: []log.ChannelEntry{{Channel: 1}}}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSwapChannelAction(log.SwapChannelPayload{From: 99, To: 5}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.Channels[0].Channel).To(Equal(channel.Key(1)))
			})

			It("Should set the timestamp precision via SetTimestampPrecision", func(ctx SpecContext) {
				l := log.Log{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetTimestampPrecisionAction(log.SetTimestampPrecisionPayload{
						TimestampPrecision: 3,
					}),
				})).To(Succeed())
				Expect(retrieve(ctx, l.Key).TimestampPrecision).To(Equal(int32(3)))
			})

			DescribeTable("Should reject an out-of-range timestamp precision",
				func(ctx SpecContext, precision int32) {
					l := log.Log{Name: "test"}
					Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
					Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
						log.NewSetTimestampPrecisionAction(log.SetTimestampPrecisionPayload{
							TimestampPrecision: precision,
						}),
					})).Error().To(MatchError(validate.ErrValidation))
				},
				Entry("below minimum", int32(-1)),
				Entry("above maximum", int32(4)),
			)

			It("Should toggle the display flags", func(ctx SpecContext) {
				l := log.Log{Name: "test", ShowChannelNames: true, ShowReceiptTimestamp: true}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetShowChannelNamesAction(log.SetShowChannelNamesPayload{
						ShowChannelNames: false,
					}),
					log.NewSetShowReceiptTimestampAction(log.SetShowReceiptTimestampPayload{
						ShowReceiptTimestamp: false,
					}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.ShowChannelNames).To(BeFalse())
				Expect(res.ShowReceiptTimestamp).To(BeFalse())
			})
		})

		Describe("atomicity and broadcast", func() {
			It("Should apply a multi-action batch atomically", func(ctx SpecContext) {
				l := log.Log{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewRenameAction(log.RenamePayload{Name: "batched"}),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 10}),
					log.NewSetTimestampPrecisionAction(log.SetTimestampPrecisionPayload{
						TimestampPrecision: 2,
					}),
				})).To(Succeed())
				res := retrieve(ctx, l.Key)
				Expect(res.Name).To(Equal("batched"))
				Expect(res.Channels).To(HaveLen(1))
				Expect(res.TimestampPrecision).To(Equal(int32(2)))
			})

			It("Should not apply any action when one in the batch is rejected", func(ctx SpecContext) {
				l := log.Log{Name: "before"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewRenameAction(log.RenamePayload{Name: "after"}),
					log.NewSetTimestampPrecisionAction(log.SetTimestampPrecisionPayload{
						TimestampPrecision: 99,
					}),
				})).Error().To(MatchError(validate.ErrValidation))
				Expect(retrieve(ctx, l.Key).Name).To(Equal("before"))
			})

			It("Should notify subscribers with the dispatched ScopedAction on success", func(ctx SpecContext) {
				l := log.Log{Name: "observed"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				rec := &Recorder[log.Key, log.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				actions := []log.Action{
					log.NewRenameAction(log.RenamePayload{Name: "broadcast"}),
					log.NewAddChannelAction(log.AddChannelPayload{Channel: 5}),
				}
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "client-xyz", actions)).
					To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(l.Key))
				Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
				Expect(seen[0].Actions).To(HaveLen(2))
				Expect(seen[0].Actions[0].Type).To(Equal(log.ActionTypeRename))
				Expect(seen[0].Actions[1].Type).To(Equal(log.ActionTypeAddChannel))
			})

			It("Should assign strictly monotonic Seq across successive dispatches", func(ctx SpecContext) {
				l := log.Log{Name: "seq"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				rec := &Recorder[log.Key, log.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				for _, name := range []string{"a", "b", "c"} {
					Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d", []log.Action{
						log.NewRenameAction(log.RenamePayload{Name: name}),
					})).To(Succeed())
				}
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(3))
				Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
				Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
			})

			It("Should not notify subscribers when Reduce rejects the action", func(ctx SpecContext) {
				l := log.Log{Name: "rejected"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
				rec := &Recorder[log.Key, log.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(svc.NewWriter(tx).Dispatch(ctx, l.Key, "d1", []log.Action{
					log.NewSetTimestampPrecisionAction(log.SetTimestampPrecisionPayload{
						TimestampPrecision: 99,
					}),
				})).Error().To(MatchError(validate.ErrValidation))
				Expect(rec.Snapshot()).To(BeEmpty())
			})
		})
	})
})
