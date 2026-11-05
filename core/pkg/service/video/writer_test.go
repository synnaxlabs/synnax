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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

func retrieve(ctx context.Context, key video.Key) video.Video {
	GinkgoHelper()
	var res video.Video
	Expect(svc.NewRetrieve().Where(video.MatchKeys(key)).Entry(&res).Exec(ctx, nil)).
		To(Succeed())
	return res
}

func create(ctx context.Context, v video.Video) video.Video {
	GinkgoHelper()
	Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &v)).To(Succeed())
	return v
}

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a video view with a new key", func(ctx SpecContext) {
			rangeKey := uuid.New()
			v := video.Video{
				Name:    "camera",
				Channel: 5,
				Range:   &ranger.Ref{Variant: ranger.KeyRef{Key: rangeKey}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &v)).To(Succeed())
			Expect(v.Key).ToNot(Equal(uuid.Nil()))
			var res video.Video
			Expect(svc.NewRetrieve().Where(video.MatchKeys(v.Key)).Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(v))
		})

		It("Should reject a video view with no name", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &video.Video{})).
				To(MatchError(ContainSubstring("name: required")))
		})

		It("Should make the view a child of its project", func(ctx SpecContext) {
			v := video.Video{Name: "camera"}
			Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &v)).To(Succeed())
			var children []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(project.OntologyID(proj.Key)).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&children).
				Exec(ctx, tx)).To(Succeed())
			Expect(children).To(ContainElement(HaveField("ID", v.OntologyID())))
		})

		It("Should create a view under no project", func(ctx SpecContext) {
			v := video.Video{Name: "camera"}
			Expect(svc.NewWriter(tx).Create(ctx, uuid.Nil(), &v)).To(Succeed())
			Expect(MustSucceed(svc.RetrieveResource(ctx, v.Key.String(), tx)).Name).
				To(Equal("camera"))
		})
	})

	Describe("Delete", func() {
		It("Should delete a view and its ontology resource", func(ctx SpecContext) {
			v := video.Video{Name: "camera"}
			Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, v.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(video.MatchKeys(v.Key)).
				Entry(&video.Video{}).Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			Expect(svc.RetrieveResource(ctx, v.Key.String(), tx)).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Dispatch", func() {
		It("Should rename the view", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "before"})
			Expect(svc.Dispatch(ctx, v.Key, "d1", []video.Action{
				video.NewRenameAction(video.RenamePayload{Name: "after"}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).Name).To(Equal("after"))
		})

		It("Should set the channel of the view", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			Expect(svc.Dispatch(ctx, v.Key, "d1", []video.Action{
				video.NewSetChannelAction(video.SetChannelPayload{Channel: 12}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).Channel).To(Equal(channel.Key(12)))
		})

		It("Should set and clear the range of the view", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			ref := ranger.Ref{Variant: ranger.KeyRef{Key: uuid.New()}}
			Expect(svc.Dispatch(ctx, v.Key, "d1", []video.Action{
				video.NewSetRangeAction(video.SetRangePayload{Range: &ref}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).Range).To(Equal(&ref))
			Expect(svc.Dispatch(ctx, v.Key, "d2", []video.Action{
				video.NewSetRangeAction(video.SetRangePayload{}),
			})).To(Succeed())
			Expect(retrieve(ctx, v.Key).Range).To(BeNil())
		})

		It("Should reject a create without changing any view", func(ctx SpecContext) {
			target := create(ctx, video.Video{Name: "target"})
			Expect(svc.Dispatch(ctx, target.Key, "d1", []video.Action{
				video.NewCreateAction(video.CreatePayload{
					Video: video.Video{Key: target.Key, Name: "overwritten"},
				}),
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("only the Core dispatches a video create")),
			))
			Expect(retrieve(ctx, target.Key).Name).To(Equal("target"))
		})

		It("Should reject an action with no payload", func(ctx SpecContext) {
			v := create(ctx, video.Video{Name: "camera"})
			Expect(svc.Dispatch(ctx, v.Key, "d1", []video.Action{
				{Type: video.ActionTypeRename},
			})).To(MatchError(union.ErrMissingPayload))
		})

		It(
			"Should apply no action of a batch that leaves the view invalid",
			func(ctx SpecContext) {
				v := create(ctx, video.Video{Name: "camera"})
				Expect(svc.Dispatch(ctx, v.Key, "d1", []video.Action{
					video.NewSetChannelAction(video.SetChannelPayload{Channel: 3}),
					video.NewRenameAction(video.RenamePayload{Name: ""}),
				})).To(MatchError(ContainSubstring("name: required")))
				res := retrieve(ctx, v.Key)
				Expect(res.Name).To(Equal("camera"))
				Expect(res.Channel).To(BeZero())
			},
		)

		It(
			"Should notify subscribers of the dispatched actions",
			func(ctx SpecContext) {
				v := create(ctx, video.Video{Name: "camera"})
				rec := &Recorder[video.Key, video.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(svc.Dispatch(ctx, v.Key, "client-xyz", []video.Action{
					video.NewSetChannelAction(video.SetChannelPayload{Channel: 7}),
				})).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Key))
				Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
				Expect(seen[0].Actions).To(Equal([]video.Action{
					video.NewSetChannelAction(video.SetChannelPayload{Channel: 7}),
				}))
			},
		)

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
