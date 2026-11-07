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
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apivideo "github.com/synnaxlabs/synnax/pkg/api/video"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	It("Should require the video and RBAC services", func() {
		Expect(apivideo.ServiceConfig{RBAC: &rbac.Service{}}.Validate()).
			To(MatchError("video: must be non-nil"))
		Expect(apivideo.ServiceConfig{Video: &video.Service{}}.Validate()).
			To(MatchError("rbac: must be non-nil"))
	})
})

var _ = Describe("Video", func() {
	var editor user.User
	BeforeEach(func(ctx SpecContext) {
		editor = newUser(
			ctx,
			access.ActionCreate,
			access.ActionRetrieve,
			access.ActionUpdate,
			access.ActionDelete,
		)
	})

	create := func(ctx SpecContext, name string) video.Video {
		GinkgoHelper()
		res := MustSucceed(
			apiSvc.Create(AuthedCtx(ctx, editor), nil, apivideo.CreateRequest{
				Project: proj.Key,
				Videos:  []video.Video{{Name: name, Channel: 4}},
			}),
		)
		Expect(res.Videos).To(HaveLen(1))
		return res.Videos[0]
	}

	It("Should create and retrieve a video view", func(ctx SpecContext) {
		v := create(ctx, "camera")
		Expect(v.Key).ToNot(Equal(uuid.Nil()))
		res := MustSucceed(
			apiSvc.Retrieve(AuthedCtx(ctx, editor), apivideo.RetrieveRequest{
				Where: &video.Where{Key: &filter.Set[video.Key]{Eq: &v.Key}},
			}),
		)
		Expect(res.Videos).To(Equal([]video.Video{v}))
	})

	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			rec := &Recorder[video.Key, video.Action]{}
			DeferCleanup(svc.Video.OnAction(rec.Record))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, editor), nil, apivideo.CreateRequest{
					Project:     proj.Key,
					Videos:      []video.Video{{Name: "broadcast"}},
					DispatchKey: "dk-create",
				}),
			)
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(res.Videos[0].Key))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)

	It("Should dispatch actions to a video view", func(ctx SpecContext) {
		v := create(ctx, "camera")
		MustSucceed(
			apiSvc.Dispatch(AuthedCtx(ctx, editor), nil, apivideo.DispatchRequest{
				Key:         v.Key,
				DispatchKey: "dk",
				Actions: []video.Action{
					video.NewSetChannelAction(video.SetChannelPayload{Channel: 9}),
				},
			}),
		)
		res := MustSucceed(
			apiSvc.Retrieve(AuthedCtx(ctx, editor), apivideo.RetrieveRequest{
				Where: &video.Where{Key: &filter.Set[video.Key]{Eq: &v.Key}},
			}),
		)
		Expect(res.Videos[0].Channel).To(Equal(channel.Key(9)))
	})

	It("Should delete a video view", func(ctx SpecContext) {
		v := create(ctx, "camera")
		MustSucceed(apiSvc.Delete(AuthedCtx(ctx, editor), nil, apivideo.DeleteRequest{
			Keys: []video.Key{v.Key},
		}))
		res := MustSucceed(
			apiSvc.Retrieve(AuthedCtx(ctx, editor), apivideo.RetrieveRequest{
				Where: &video.Where{Key: &filter.Set[video.Key]{Eq: &v.Key}},
			}),
		)
		Expect(res.Videos).To(BeEmpty())
	})

	Describe("Access", func() {
		It("Should deny a create to a user with no policy", func(ctx SpecContext) {
			stranger := newUser(ctx)
			Expect(apiSvc.Create(AuthedCtx(ctx, stranger), nil, apivideo.CreateRequest{
				Project: proj.Key,
				Videos:  []video.Video{{Name: "camera"}},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It(
			"Should deny each request a user lacks permission for",
			func(ctx SpecContext) {
				v := create(ctx, "camera")
				stranger := newUser(ctx)
				Expect(
					apiSvc.Retrieve(AuthedCtx(ctx, stranger), apivideo.RetrieveRequest{
						Where: &video.Where{Key: &filter.Set[video.Key]{Eq: &v.Key}},
					}),
				).Error().
					To(MatchError(access.ErrDenied))
				Expect(
					apiSvc.Dispatch(
						AuthedCtx(ctx, stranger),
						nil,
						apivideo.DispatchRequest{
							Key: v.Key,
							Actions: []video.Action{
								video.NewRenameAction(video.RenamePayload{Name: "x"}),
							},
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))
				Expect(
					apiSvc.Delete(AuthedCtx(ctx, stranger), nil, apivideo.DeleteRequest{
						Keys: []video.Key{v.Key},
					}),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)
	})
})
