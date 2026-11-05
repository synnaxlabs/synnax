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
	"net"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pion/webrtc/v4"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/whip"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/video"
)

// freeUDPPort returns a local UDP port that no socket holds.
func freeUDPPort() int {
	GinkgoHelper()
	conn := MustSucceed(net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}))
	defer func() { Expect(conn.Close()).To(Succeed()) }()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

var _ = Describe("ReadTask", func() {
	var (
		factory *whip.Factory
		cfg     whip.FactoryConfig
	)
	BeforeEach(func() {
		cfg = whip.FactoryConfig{
			DB:            db,
			Channel:       channelSvc,
			Framer:        framerSvc,
			Status:        statusSvc,
			KeyframeLimit: 500 * time.Millisecond,
		}
		factory = MustOpen(whip.OpenFactory(cfg))
	})

	It(
		"Should write the frames that an encoder publishes to a video channel",
		func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			t := newTask(readConfig(ch))
			start(ctx, factory, t)
			e := openEncoder(webrtc.MimeTypeH264)
			e.publish(ctx, factory, t)
			Expect(v.connect(e, sps, pps, idr)).To(Equal(keyframe(sps, pps, idr)))
			e.write(pSlice)
			v.expect(deltaFrame(pSlice))
		},
	)

	It(
		"Should write timed frames to a persisted video channel and its index",
		func(ctx SpecContext) {
			idx, ch := createPersisted(ctx)
			v := watch(ctx, ch)
			cfg := readConfig(ch)
			cfg.Offset = -telem.Hour
			t := newTask(cfg)
			before := telem.Now()
			running := start(ctx, factory, t)
			e := openEncoder(webrtc.MimeTypeH264)
			e.publish(ctx, factory, t)
			v.connect(e, sps, pps, idr)
			Expect(running.Stop(false)).To(Succeed())
			stamps, frames := readPersisted(ctx, idx, ch)
			Expect(frames[0]).To(Equal(keyframe(sps, pps, idr)))
			Expect(stamps).To(HaveLen(len(frames)))
			Expect(stamps[0]).To(BeNumerically(">=", before.Add(-telem.Hour)))
			Expect(stamps[len(stamps)-1]).
				To(BeNumerically("<=", telem.Now().Add(-telem.Hour)))
		},
	)

	It(
		"Should time frames from the sender reports of the encoder",
		func(ctx SpecContext) {
			idx, ch := createPersisted(ctx)
			v := watch(ctx, ch)
			cfg := readConfig(ch)
			cfg.TimeSource = whip.TimeSourceCapture
			t := newTask(cfg)
			running := start(ctx, factory, t)
			e := openEncoder(webrtc.MimeTypeH264)
			e.publish(ctx, factory, t)
			captured := 1000 * telem.SecondTS
			Eventually(func(g Gomega) {
				g.Expect(e.report(captured)).To(Succeed())
				e.write(sps, pps, idr)
				_, ok := v.take()
				g.Expect(ok).To(BeTrue())
			}).WithTimeout(10 * time.Second).Should(Succeed())
			Expect(running.Stop(false)).To(Succeed())
			stamps, _ := readPersisted(ctx, idx, ch)
			// Each report maps RTP timestamp 0, the first frame, to captured. A frame
			// that reaches the task before the first report has no time.
			Expect(stamps).ToNot(BeEmpty())
			Expect(stamps[0]).To(BeNumerically(">=", captured))
			Expect(stamps[len(stamps)-1]).
				To(BeNumerically("<", captured.Add(telem.Minute)))
		},
	)

	It(
		"Should receive the media of every session on one UDP port",
		func(ctx SpecContext) {
			cfg.UDPPort = freeUDPPort()
			factory = MustOpen(whip.OpenFactory(cfg))
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			t := newTask(readConfig(ch))
			start(ctx, factory, t)
			e := openEncoder(webrtc.MimeTypeH264)
			e.publish(ctx, factory, t)
			Expect(v.connect(e, sps, pps, idr)).To(Equal(keyframe(sps, pps, idr)))
		},
	)

	It(
		"Should replace the session of a task with a new publish",
		func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			t := newTask(readConfig(ch))
			start(ctx, factory, t)
			first := openEncoder(webrtc.MimeTypeH264).publish(ctx, factory, t)
			e := openEncoder(webrtc.MimeTypeH264)
			second := e.publish(ctx, factory, t)
			Expect(v.connect(e, sps, pps, idr)).To(Equal(keyframe(sps, pps, idr)))
			Expect(factory.Unpublish(t.Key, first)).To(MatchError(query.ErrNotFound))
			Expect(factory.Unpublish(t.Key, second)).To(Succeed())
		},
	)

	It("Should end a session", func(ctx SpecContext) {
		ch := createVirtual(ctx, telem.VideoT)
		t := newTask(readConfig(ch))
		start(ctx, factory, t)
		key := openEncoder(webrtc.MimeTypeH264).publish(ctx, factory, t)
		Expect(factory.Unpublish(t.Key, key)).To(Succeed())
		Expect(factory.Unpublish(t.Key, key)).To(SatisfyAll(
			MatchError(query.ErrNotFound),
			MatchError(ContainSubstring("session "+key.String()+" does not exist")),
		))
	})

	It("Should reject a publish to a task that has not started", func(ctx SpecContext) {
		ch := createVirtual(ctx, telem.VideoT)
		cfg := readConfig(ch)
		cfg.AutoStart = false
		t := newTask(cfg)
		MustSucceed(factory.ConfigureTask(ctx, t, driver.NoCommand))
		offer := openEncoder(webrtc.MimeTypeH264).offer()
		Expect(factory.Publish(ctx, t.Key, offer)).Error().
			To(MatchError(whip.ErrNotRunning))
		Expect(factory.Channel(t.Key)).Error().To(MatchError(whip.ErrNotRunning))
	})

	It("Should reject an offer with no H.264 video", func(ctx SpecContext) {
		ch := createVirtual(ctx, telem.VideoT)
		t := newTask(readConfig(ch))
		start(ctx, factory, t)
		offer := openEncoder(webrtc.MimeTypeVP8).offer()
		Expect(factory.Publish(ctx, t.Key, offer)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"offer: the encoder offers no H.264 video. Set its video encoder to H.264",
			)),
		))
	})

	It("Should return the video channel of a running task", func(ctx SpecContext) {
		ch := createVirtual(ctx, telem.VideoT)
		t := newTask(readConfig(ch))
		start(ctx, factory, t)
		Expect(factory.Channel(t.Key)).To(Equal(ch.Key()))
	})

	It("Should fail the task for a stream with B-frames", func(ctx SpecContext) {
		ch := createVirtual(ctx, telem.VideoT)
		v := watch(ctx, ch)
		t := newTask(readConfig(ch))
		start(ctx, factory, t)
		e := openEncoder(webrtc.MimeTypeH264)
		e.publish(ctx, factory, t)
		v.connect(e, sps, pps, idr)
		e.write(bSlice)
		expectStatus(
			ctx, t, status.VariantError, "Failed to run task", video.ErrBFrames.Error(),
		)
	})

	It(
		"Should warn while the stream has no keyframe for longer than the limit",
		func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			t := newTask(readConfig(ch))
			start(ctx, factory, t)
			e := openEncoder(webrtc.MimeTypeH264)
			e.publish(ctx, factory, t)
			v.connect(e, sps, pps, idr)
			Eventually(func(g Gomega) {
				e.write(pSlice)
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Message).To(Equal("Failed to read some data"))
				g.Expect(stat.Description).To(Equal(video.ErrNoKeyframe.Error()))
			}).WithTimeout(5 * time.Second).Should(Succeed())
			v.clear()
			Eventually(func(g Gomega) {
				e.write(sps, pps, idr)
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
				g.Expect(stat.Message).To(Equal("Started task"))
			}).WithTimeout(5 * time.Second).Should(Succeed())
		},
	)

	Describe("Configuration", func() {
		expectInvalid := func(ctx SpecContext, cfg whip.ReadConfig, message string) {
			GinkgoHelper()
			Expect(factory.ConfigureTask(ctx, newTask(cfg), "cmd")).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
		}

		It("Should reject a channel that does not exist", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.VideoT)
			Expect(channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false)).
				To(Succeed())
			expectInvalid(
				ctx, readConfig(ch), "channel: "+ch.Key().String()+" does not exist",
			)
		})

		It(
			"Should reject a channel that is not a video channel",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float32T)
				expectInvalid(
					ctx,
					readConfig(ch),
					"channel: "+ch.String()+" has data type float32, not video",
				)
			},
		)

		It("Should not handle other task types", func(ctx SpecContext) {
			t := task.Task{Key: uuid.New(), Type: "rtsp_read"}
			Expect(factory.ConfigureTask(ctx, t, "cmd")).Error().
				To(MatchError(driver.ErrTaskNotHandled))
		})
	})
})
