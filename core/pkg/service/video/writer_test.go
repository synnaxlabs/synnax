// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a video view with a new key", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := video.Video{
				Name:          "camera",
				Channel:       5,
				PlaybackRange: &ranger.Ref{Variant: ranger.KeyRef{Key: uuid.New()}},
				Project:       &proj.Key,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(v.Key).ToNot(Equal(uuid.Nil()))
			var res video.Video
			Expect(svc.NewRetrieve().Where(video.MatchKeys(v.Key)).Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(v))
		})

		It("Should reject a video view with no name", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).Create(ctx, &video.Video{Project: &proj.Key})).
				To(MatchError(ContainSubstring("name: required")))
		})

		It("Should reject a video view with no owner", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).Create(ctx, &video.Video{Name: "camera"})).
				To(MatchError(
					"project: must belong to exactly one of a project and a range: " +
						"validation error",
				))
		})

		It("Should refuse a view whose key exists", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "first"})
			again := video.Video{Key: v.Key, Name: "second", Project: &proj.Key}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &again)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("video %s already exists", v.Key)),
			))
			Expect(retrieve(ctx, v.Key)).To(Equal(v))
		})
	})

	Describe("Delete", func() {
		It("Should delete a view", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, v.Key)
			})).To(Succeed())
			Expect(svc.NewRetrieve().Where(video.MatchKeys(v.Key)).
				Entry(&video.Video{}).Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Apply", func() {
		It("Should rename the view", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "before"})
			Expect(dispatch(ctx, v.Key, "d1", []video.Action{
				video.NewSetAction(video.SetPayload{
					Path:  []string{"name"},
					Value: "after",
				}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).Name).To(Equal("after"))
		})

		It("Should set the channel of the view", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			Expect(dispatch(ctx, v.Key, "d1", []video.Action{
				video.NewSetChannelAction(video.SetChannelPayload{Channel: 12}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).Channel).To(Equal(channel.Key(12)))
		})

		It("Should set and clear the playback range", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			ref := ranger.Ref{Variant: ranger.KeyRef{Key: uuid.New()}}
			Expect(dispatch(ctx, v.Key, "d1", []video.Action{
				video.NewSetPlaybackRangeAction(
					video.SetPlaybackRangePayload{PlaybackRange: &ref},
				),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).PlaybackRange).To(Equal(&ref))
			Expect(dispatch(ctx, v.Key, "d2", []video.Action{
				video.NewSetPlaybackRangeAction(video.SetPlaybackRangePayload{}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).PlaybackRange).To(BeNil())
		})

		It("Should reject a create without changing any view", func(ctx SpecContext) {
			target := create(ctx, video.Video{Name: "target"})
			Expect(dispatch(ctx, target.Key, "d1", []video.Action{
				video.NewCreateAction(video.CreatePayload{
					Video: video.Video{Key: target.Key, Name: "overwritten"},
				}),
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, target.Key).Name).To(Equal("target"))
		})

		It("Should reject an action with no payload", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			Expect(dispatch(ctx, v.Key, "d1", []video.Action{
				{Type: video.ActionTypeSetChannel},
			})).To(MatchError(union.ErrMissingPayload))
		})

		It(
			"Should apply no action of a batch that leaves the view invalid",
			func(ctx SpecContext) {
				v := create(ctx, video.Video{Name: "camera"})
				Expect(dispatch(ctx, v.Key, "d1", []video.Action{
					video.NewSetChannelAction(video.SetChannelPayload{Channel: 3}),
					video.NewSetAction(video.SetPayload{
						Path:  []string{"name"},
						Value: "",
					}),
				})).To(MatchError(ContainSubstring("name: required")))
				res := retrieve(ctx, v.Key)
				Expect(res.Name).To(Equal("camera"))
				Expect(res.Channel).To(BeZero())
			},
		)

		It("Should notify subscribers of the applied actions", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			rec := &Recorder[video.Key, video.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(dispatch(ctx, v.Key, "client-xyz", []video.Action{
				video.NewSetChannelAction(video.SetChannelPayload{Channel: 7}),
			})).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(v.Key))
			Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
			Expect(seen[0].Actions).To(Equal([]video.Action{
				video.NewSetChannelAction(video.SetChannelPayload{Channel: 7}),
			}))
		})

		It("Should broadcast a create to subscribers", func(ctx SpecContext) {
			rec := &Recorder[video.Key, video.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			v := create(ctx, video.Video{Name: "camera"})
			Eventually(rec.Snapshot).Should(ContainElement(SatisfyAll(
				HaveField("Key", v.Key),
				HaveField("Actions", ConsistOf(
					video.NewCreateAction(video.CreatePayload{Video: v}),
				)),
			)))
		})
	})
})
