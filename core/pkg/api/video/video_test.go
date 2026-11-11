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
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apivideo "github.com/synnaxlabs/synnax/pkg/api/video"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	It("Should require the video service", func() {
		Expect(apivideo.ServiceConfig{}.Validate()).
			To(MatchError("video: must be non-nil"))
	})
})

var _ = Describe("Video", func() {
	var editor user.User
	BeforeEach(func(ctx SpecContext) { editor = newEditor(ctx) })

	createVideo := func(ctx SpecContext, name string) video.Video {
		GinkgoHelper()
		res := MustSucceed(create(AuthedCtx(ctx, editor), apivideo.CreateRequest{
			Videos: []video.Video{{Name: name, Channel: 4, Project: &proj.Key}},
		}))
		Expect(res.Videos).To(HaveLen(1))
		return res.Videos[0]
	}
	retrieveVideos := func(ctx SpecContext, key video.Key) []video.Video {
		GinkgoHelper()
		return MustSucceed(retrieve(AuthedCtx(ctx, editor), apivideo.RetrieveRequest{
			Where: &video.Where{Key: &filter.Set[video.Key]{Eq: &key}},
		})).Videos
	}

	It("Should create and retrieve a video view", func(ctx SpecContext) {
		v := createVideo(ctx, "camera")
		Expect(v.Key).ToNot(Equal(uuid.Nil()))
		Expect(retrieveVideos(ctx, v.Key)).To(Equal([]video.Video{v}))
	})

	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			rec := &Recorder[video.Key, video.Action]{}
			DeferCleanup(svc.Video.OnAction(rec.Record))
			res := MustSucceed(create(AuthedCtx(ctx, editor), apivideo.CreateRequest{
				Videos:      []video.Video{{Name: "broadcast", Project: &proj.Key}},
				DispatchKey: "dk-create",
			}))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(res.Videos[0].Key))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)

	It("Should dispatch actions to a video view", func(ctx SpecContext) {
		v := createVideo(ctx, "camera")
		MustSucceed(dispatch(AuthedCtx(ctx, editor), apivideo.DispatchRequest{
			Key:         v.Key,
			DispatchKey: "dk",
			Actions: []video.Action{
				video.NewSetChannelAction(video.SetChannelPayload{Channel: 9}),
			},
		}))
		Expect(retrieveVideos(ctx, v.Key)[0].Channel).To(Equal(channel.Key(9)))
	})

	It("Should delete a video view", func(ctx SpecContext) {
		v := createVideo(ctx, "camera")
		MustSucceed(del(AuthedCtx(ctx, editor), apivideo.DeleteRequest{
			Keys: []video.Key{v.Key},
		}))
		Expect(retrieveVideos(ctx, v.Key)).To(BeEmpty())
	})

	It("Should move a video view into a group", func(ctx SpecContext) {
		v := createVideo(ctx, "camera")
		g := group.Group{Name: "cameras", Parent: new(project.ResourceID(proj.Key))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		grant(ctx, access.ActionUpdate, editor.ResourceID(), g.ResourceID())
		Expect(dispatch(AuthedCtx(ctx, editor), apivideo.DispatchRequest{
			Key:         v.Key,
			DispatchKey: "dk-move",
			Actions: []video.Action{video.NewSetAction(video.SetPayload{
				Path:  []string{"group"},
				Value: g.Key,
			})},
		})).To(HaveField("Actions", BeEmpty()))
		Expect(retrieveVideos(ctx, v.Key)[0].Group).To(Equal(&g.Key))
	})

	Describe("Access", func() {
		It("Should deny a create to a user with no policy", func(ctx SpecContext) {
			stranger := newUser(ctx)
			Expect(create(AuthedCtx(ctx, stranger), apivideo.CreateRequest{
				Videos: []video.Video{{Name: "camera", Project: &proj.Key}},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It(
			"Should deny each request a user lacks permission for",
			func(ctx SpecContext) {
				v := createVideo(ctx, "camera")
				stranger := newUser(ctx)
				Expect(retrieve(AuthedCtx(ctx, stranger), apivideo.RetrieveRequest{
					Where: &video.Where{Key: &filter.Set[video.Key]{Eq: &v.Key}},
				})).Error().To(MatchError(access.ErrDenied))
				Expect(dispatch(AuthedCtx(ctx, stranger), apivideo.DispatchRequest{
					Key: v.Key,
					Actions: []video.Action{
						video.NewSetChannelAction(video.SetChannelPayload{Channel: 2}),
					},
				})).Error().To(MatchError(access.ErrDenied))
				Expect(del(AuthedCtx(ctx, stranger), apivideo.DeleteRequest{
					Keys: []video.Key{v.Key},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
})
