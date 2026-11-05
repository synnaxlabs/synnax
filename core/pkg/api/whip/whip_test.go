// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pion/webrtc/v4"
	apiwhip "github.com/synnaxlabs/synnax/pkg/api/whip"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/whip"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// grant lets subject run action on object.
func grant(
	ctx context.Context,
	subject ontology.ID,
	action access.Action,
	object ontology.ID,
) {
	GinkgoHelper()
	r := &role.Role{Name: "whip-" + uuid.New().String()}
	Expect(rbacSvc.Role.NewWriter(nil, true).Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    "whip-policy-" + uuid.New().String(),
		Objects: []ontology.ID{object},
		Actions: []access.Action{action},
	}
	policies := rbacSvc.Policy.NewWriter(nil, true)
	Expect(policies.Create(ctx, p)).To(Succeed())
	Expect(policies.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(rbacSvc.Role.NewWriter(nil, true).AssignRole(ctx, subject, r.Key)).
		To(Succeed())
}

// newKey creates a user with an API key and returns the key. The user may write to ch
// when writable is true.
func newKey(ctx context.Context, ch channel.Channel, writable bool) string {
	GinkgoHelper()
	u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Name: "whip-" + uuid.New().String(),
	}))
	subject := user.OntologyID(u.Key)
	if writable {
		grant(ctx, subject, access.ActionCreate, channel.OntologyID(ch.Key()))
	}
	_, token := MustSucceed2(apiKeySvc.NewWriter(nil).Create(ctx, subject, "encoder"))
	return token
}

// offer returns an H.264 offer from an encoder that closes at the end of the spec.
func offer() (*webrtc.PeerConnection, string) {
	GinkgoHelper()
	codec := webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeH264,
		ClockRate: 90000,
		SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;" +
			"profile-level-id=42e01f",
	}
	media := &webrtc.MediaEngine{}
	Expect(media.RegisterCodec(
		webrtc.RTPCodecParameters{RTPCodecCapability: codec, PayloadType: 96},
		webrtc.RTPCodecTypeVideo,
	)).To(Succeed())
	pc := MustSucceed(
		webrtc.NewAPI(webrtc.WithMediaEngine(media)).
			NewPeerConnection(webrtc.Configuration{}),
	)
	DeferCleanup(func() { Expect(pc.Close()).To(Succeed()) })
	track := MustSucceed(webrtc.NewTrackLocalStaticRTP(codec, "video", "synnax"))
	MustSucceed(pc.AddTransceiverFromTrack(track, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionSendonly,
	}))
	o := MustSucceed(pc.CreateOffer(nil))
	gathered := webrtc.GatheringCompletePromise(pc)
	Expect(pc.SetLocalDescription(o)).To(Succeed())
	Eventually(gathered).Should(BeClosed())
	return pc, pc.LocalDescription().SDP
}

var _ = Describe("Service", func() {
	var (
		svc             *apiwhip.Service
		t               task.Task
		allowed, denied string
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustSucceed(apiwhip.NewService(apiwhip.ServiceConfig{
			Factory: factory,
			APIKey:  apiKeySvc,
			RBAC:    rbacSvc,
		}))
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.VideoT,
			Virtual:  true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(errors.Skip(
				channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false),
				query.ErrNotFound,
			)).To(Succeed())
		})
		var cfg msgpack.EncodedJSON
		Expect(json.Unmarshal(MustSucceed(json.Marshal(whip.ReadConfig{
			AutoStart: true,
			Channel:   ch.Key(),
		})), &cfg)).To(Succeed())
		t = task.Task{
			Key:    uuid.New(),
			Name:   "obs",
			Type:   whip.ReadTaskType,
			Config: cfg,
		}
		configured := MustSucceed(factory.ConfigureTask(ctx, t, driver.NoCommand))
		DeferCleanup(func() { Expect(configured.Stop(false)).To(Succeed()) })
		allowed = newKey(ctx, ch, true)
		denied = newKey(ctx, ch, false)
	})

	Describe("Publish", func() {
		It("Should answer an offer from a key that may write to the channel", func(
			ctx SpecContext,
		) {
			pc, sdp := offer()
			res := MustSucceed(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  t.Key,
				Key:   allowed,
				Offer: sdp,
			}))
			Expect(res.Session).ToNot(Equal(uuid.Nil()))
			Expect(pc.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  res.Answer,
			})).To(Succeed())
		})

		It("Should reject an unknown API key", func(ctx SpecContext) {
			_, sdp := offer()
			Expect(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  t.Key,
				Key:   "not-a-key",
				Offer: sdp,
			})).Error().To(MatchError(auth.ErrInvalidCredentials))
		})

		It("Should reject a key that may not write to the channel", func(
			ctx SpecContext,
		) {
			_, sdp := offer()
			Expect(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  t.Key,
				Key:   denied,
				Offer: sdp,
			})).Error().To(MatchError(auth.ErrAccessDenied))
		})

		It("Should reject a task that is not running", func(ctx SpecContext) {
			_, sdp := offer()
			Expect(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  uuid.New(),
				Key:   allowed,
				Offer: sdp,
			})).Error().To(MatchError(whip.ErrNotRunning))
		})

		It("Should reject an offer that is not valid SDP", func(ctx SpecContext) {
			Expect(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  t.Key,
				Key:   allowed,
				Offer: "nope",
			})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(HavePrefix("offer: ")),
			))
		})
	})

	Describe("Unpublish", func() {
		It("Should end a session", func(ctx SpecContext) {
			_, sdp := offer()
			res := MustSucceed(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  t.Key,
				Key:   allowed,
				Offer: sdp,
			}))
			req := apiwhip.UnpublishRequest{
				Task:    t.Key,
				Session: res.Session,
				Key:     allowed,
			}
			MustSucceed(svc.Unpublish(ctx, req))
			Expect(svc.Unpublish(ctx, req)).Error().To(And(
				MatchError(query.ErrNotFound),
				MatchError(HavePrefix(
					"session "+res.Session.String()+" does not exist",
				)),
			))
		})

		It("Should reject a key that may not write to the channel", func(
			ctx SpecContext,
		) {
			_, sdp := offer()
			res := MustSucceed(svc.Publish(ctx, apiwhip.PublishRequest{
				Task:  t.Key,
				Key:   allowed,
				Offer: sdp,
			}))
			Expect(svc.Unpublish(ctx, apiwhip.UnpublishRequest{
				Task:    t.Key,
				Session: res.Session,
				Key:     denied,
			})).Error().To(MatchError(auth.ErrAccessDenied))
		})
	})
})
