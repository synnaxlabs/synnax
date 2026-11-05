// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rtsp_test

import (
	"time"
	"uuid"

	"github.com/bluenviron/gortsplib/v5/pkg/format"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/rtsp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/video"
)

var _ = Describe("ReadTask", func() {
	var factory *rtsp.Factory
	BeforeEach(func() {
		factory = MustSucceed(rtsp.NewFactory(rtsp.FactoryConfig{
			DB:            db,
			Channel:       channelSvc,
			Framer:        framerSvc,
			Status:        statusSvc,
			KeyframeLimit: 500 * time.Millisecond,
		}))
	})

	It(
		"Should write the frames of an H.264 stream to a video channel",
		func(ctx SpecContext) {
			cam := openH264Camera(freeAddr())
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			start(ctx, factory, newTask(readConfig(cam.url(), ch)))
			Expect(v.connect(cam, sps, pps, idr)).To(Equal(keyframe(sps, pps, idr)))
			cam.write(pSlice)
			v.expect(deltaFrame(pSlice))
		},
	)

	It(
		"Should write timed frames to a persisted video channel and its index",
		func(ctx SpecContext) {
			cam := openH264Camera(freeAddr())
			idx, ch := createPersisted(ctx)
			v := watch(ctx, ch)
			cfg := readConfig(cam.url(), ch)
			cfg.Offset = -telem.Hour
			before := telem.Now()
			configured := MustSucceed(factory.ConfigureTask(ctx, newTask(cfg), "cmd"))
			v.connect(cam, sps, pps, idr)
			cam.write(pSlice)
			v.expect(deltaFrame(pSlice))
			Expect(configured.Stop(false)).To(Succeed())
			stamps, frames := readPersisted(ctx, idx, ch)
			Expect(frames[0]).To(Equal(keyframe(sps, pps, idr)))
			Expect(frames[len(frames)-1]).To(Equal(deltaFrame(pSlice)))
			Expect(stamps).To(HaveLen(len(frames)))
			Expect(stamps[0]).To(BeNumerically(">=", before.Add(-telem.Hour)))
			Expect(stamps[len(stamps)-1]).
				To(BeNumerically("<=", telem.Now().Add(-telem.Hour)))
			for i := 1; i < len(stamps); i++ {
				Expect(stamps[i]).To(BeNumerically(">", stamps[i-1]))
			}
		},
	)

	It(
		"Should not write the frames before the first keyframe of a session",
		func(ctx SpecContext) {
			cam := openH264Camera(freeAddr())
			idx, ch := createPersisted(ctx)
			t := newTask(readConfig(cam.url(), ch))
			configured := MustSucceed(factory.ConfigureTask(ctx, t, "cmd"))
			// The warning shows that the task reads the stream.
			Eventually(func(g Gomega) {
				cam.write(pSlice)
				g.Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantWarning))
			}).WithTimeout(5 * time.Second).Should(Succeed())
			Eventually(func(g Gomega) {
				cam.write(sps, pps, idr)
				g.Expect(taskStatus(ctx, t).Variant).To(Equal(status.VariantSuccess))
			}).WithTimeout(5 * time.Second).Should(Succeed())
			Expect(configured.Stop(false)).To(Succeed())
			_, frames := readPersisted(ctx, idx, ch)
			Expect(frames[0]).To(Equal(keyframe(sps, pps, idr)))
		},
	)

	It(
		"Should put the parameter sets of the session description in front of an IDR",
		func(ctx SpecContext) {
			cam := openH264Camera(freeAddr(), sps, pps)
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			start(ctx, factory, newTask(readConfig(cam.url(), ch)))
			Expect(v.connect(cam, idr)).To(Equal(keyframe(sps, pps, idr)))
		},
	)

	It("Should fail the task for a stream with B-frames", func(ctx SpecContext) {
		cam := openH264Camera(freeAddr())
		ch := createVirtual(ctx, telem.VideoT)
		v := watch(ctx, ch)
		t := newTask(readConfig(cam.url(), ch))
		start(ctx, factory, t)
		v.connect(cam, sps, pps, idr)
		cam.write(bSlice)
		expectStatus(
			ctx, t, status.VariantError, "Failed to run task", video.ErrBFrames.Error(),
		)
	})

	It("Should fail the task for a stream with no H.264 track", func(ctx SpecContext) {
		cam := openCamera(freeAddr(), &format.MJPEG{})
		ch := createVirtual(ctx, telem.VideoT)
		t := newTask(readConfig(cam.url(), ch))
		start(ctx, factory, t)
		expectStatus(
			ctx, t, status.VariantError, "Failed to run task", rtsp.ErrNoH264.Error(),
		)
	})

	It(
		"Should warn while the stream has no keyframe for longer than the limit",
		func(ctx SpecContext) {
			cam := openH264Camera(freeAddr())
			ch := createVirtual(ctx, telem.VideoT)
			v := watch(ctx, ch)
			t := newTask(readConfig(cam.url(), ch))
			start(ctx, factory, t)
			v.connect(cam, sps, pps, idr)
			Eventually(func(g Gomega) {
				cam.write(pSlice)
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Message).To(Equal("Failed to read some data"))
				g.Expect(stat.Description).To(Equal(video.ErrNoKeyframe.Error()))
			}).WithTimeout(5 * time.Second).Should(Succeed())
			Eventually(func(g Gomega) {
				cam.write(sps, pps, idr)
				stat := taskStatus(ctx, t)
				g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
				g.Expect(stat.Message).To(Equal("Started task"))
			}).WithTimeout(5 * time.Second).Should(Succeed())
		},
	)

	It("Should warn, then connect again, after the stream ends", func(ctx SpecContext) {
		addr := freeAddr()
		cam := openH264Camera(addr)
		ch := createVirtual(ctx, telem.VideoT)
		v := watch(ctx, ch)
		t := newTask(readConfig(cam.url(), ch))
		start(ctx, factory, t)
		v.connect(cam, sps, pps, idr)
		cam.close()
		Eventually(func(g Gomega) {
			stat := taskStatus(ctx, t)
			g.Expect(stat.Variant).To(Equal(status.VariantWarning))
			g.Expect(stat.Message).To(Equal("Failed to read data"))
		}).WithTimeout(5 * time.Second).Should(Succeed())
		v.clear()
		cam = openH264Camera(addr)
		Expect(v.connect(cam, sps, pps, idr)).To(Equal(keyframe(sps, pps, idr)))
		expectStatus(ctx, t, status.VariantSuccess, "Started task", "")
	})

	Describe("Configuration", func() {
		var cam *camera
		BeforeEach(func() { cam = openH264Camera(freeAddr()) })

		expectInvalid := func(ctx SpecContext, cfg rtsp.ReadConfig, message string) {
			GinkgoHelper()
			Expect(factory.ConfigureTask(ctx, newTask(cfg), "cmd")).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
		}

		It(
			"Should reject a URL with a scheme other than rtsp or rtsps",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.VideoT)
				expectInvalid(
					ctx,
					readConfig("http://"+cam.addr, ch),
					"url: unsupported scheme 'http'",
				)
			},
		)

		It("Should reject a channel that does not exist", func(ctx SpecContext) {
			ch := createVirtual(ctx, telem.VideoT)
			Expect(channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false)).
				To(Succeed())
			expectInvalid(
				ctx,
				readConfig(cam.url(), ch),
				"channel: "+ch.Key().String()+" does not exist",
			)
		})

		It(
			"Should reject a channel that is not a video channel",
			func(ctx SpecContext) {
				ch := createVirtual(ctx, telem.Float32T)
				expectInvalid(
					ctx,
					readConfig(cam.url(), ch),
					"channel: "+ch.String()+" has data type float32, not video",
				)
			},
		)

		It("Should not handle other task types", func(ctx SpecContext) {
			t := task.Task{Key: uuid.New(), Type: "mqtt_read"}
			Expect(factory.ConfigureTask(ctx, t, "cmd")).Error().
				To(MatchError(driver.ErrTaskNotHandled))
		})
	})
})
