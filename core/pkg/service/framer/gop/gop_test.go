// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gop_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/gop"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func videoFrame(keyframe bool, b byte) telem.VideoFrame {
	return telem.VideoFrame{
		Keyframe: keyframe,
		Codec:    telem.VideoCodecH264,
		Data:     []byte{b},
	}
}

func decode(series []telem.Series) []telem.VideoFrame {
	var frames []telem.VideoFrame
	for _, s := range series {
		frames = append(frames, MustSucceed(s.DecodeVideo())...)
	}
	return frames
}

var _ = Describe("Cache", Ordered, func() {
	var (
		node       testutil.Node
		channelSvc *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node = testutil.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Group:    groupSvc,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Group:    groupSvc,
			Ontology: otg,
			Label:    labelSvc,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostProvider: node.Cluster,
			Ontology:     otg,
			Group:        groupSvc,
			Status:       statusSvc,
		}))
	})

	var (
		key   channel.Key
		w     *framer.Writer
		out   confluence.Outlet[framer.StreamerResponse]
		cache *gop.Cache
	)

	open := func(ctx SpecContext, cfgs ...gop.Config) {
		cache = MustOpen(gop.Open(ctx, append([]gop.Config{{
			Channel: channelSvc,
			Framer:  node.Framer,
		}}, cfgs...)...))
		ch := &channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.VideoT,
			Virtual:  true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, ch)).To(Succeed())
		key = ch.Key()
		w = MustOpen(node.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{key},
		}))
		streamer := MustSucceed(node.Framer.NewStreamer(framer.StreamerConfig{
			Keys:        []channel.Key{key},
			SendOpenAck: new(true),
		}))
		var in confluence.Inlet[framer.StreamerRequest]
		in, out = confluence.Attach(streamer, 10)
		sCtx, cancel := signal.Isolated()
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		DeferCleanup(func() {
			in.Close()
			Eventually(out.Outlet()).Should(BeClosed())
			cancel()
		})
		Eventually(out.Outlet()).Should(Receive())
	}

	// write writes the frames as one series, and returns the alignments of its samples.
	write := func(frames ...telem.VideoFrame) telem.AlignmentBounds {
		Expect(w.Write(frame.NewUnary(key, telem.NewVideoSeries(frames)))).To(BeTrue())
		var res framer.StreamerResponse
		Eventually(out.Outlet()).Should(Receive(&res))
		return res.Frame.SeriesAt(0).AlignmentBounds()
	}

	// follow writes a GOP until the cache follows the channel, and returns the
	// alignments of the GOP's samples.
	follow := func(frames ...telem.VideoFrame) telem.AlignmentBounds {
		var bounds telem.AlignmentBounds
		Eventually(func(g Gomega) {
			bounds = write(frames...)
			_, ok := cache.Bridge(key, bounds.Upper)
			g.Expect(ok).To(BeTrue())
		}).Should(Succeed())
		return bounds
	}

	frames := []telem.VideoFrame{
		videoFrame(false, 0),
		videoFrame(true, 1),
		videoFrame(false, 2),
		videoFrame(false, 3),
	}

	It("Should bridge a viewer to the last keyframe", func(ctx SpecContext) {
		open(ctx)
		bounds := follow(frames...)
		Expect(decode(MustBeOk(cache.Bridge(key, bounds.Upper)))).To(Equal(frames[1:]))
	})

	It("Should bridge a viewer that starts inside the GOP", func(ctx SpecContext) {
		open(ctx)
		bounds := follow(frames...)
		Expect(decode(MustBeOk(cache.Bridge(key, bounds.Lower.AddSamples(3))))).
			To(Equal(frames[1:3]))
	})

	It(
		"Should bridge a viewer that starts at the keyframe with no frames",
		func(ctx SpecContext) {
			open(ctx)
			bounds := follow(frames...)
			Expect(
				MustBeOk(cache.Bridge(key, bounds.Lower.AddSamples(1))),
			).To(BeEmpty())
		},
	)

	It(
		"Should not bridge a viewer that starts before the keyframe",
		func(ctx SpecContext) {
			open(ctx)
			bounds := follow(frames...)
			_, ok := cache.Bridge(key, bounds.Lower)
			Expect(ok).To(BeFalse())
		},
	)

	It(
		"Should not bridge a viewer that starts after the cached frames",
		func(ctx SpecContext) {
			open(ctx)
			bounds := follow(frames...)
			_, ok := cache.Bridge(key, bounds.Upper.AddSamples(1))
			Expect(ok).To(BeFalse())
		},
	)

	It("Should add the frames that follow the keyframe", func(ctx SpecContext) {
		open(ctx)
		follow(frames...)
		bounds := write(frames[2])
		Expect(decode(MustBeOk(cache.Bridge(key, bounds.Upper)))).
			To(Equal(append(frames[1:], frames[2])))
	})

	It("Should restart at each new keyframe", func(ctx SpecContext) {
		open(ctx)
		follow(frames...)
		bounds := write(frames[1], frames[2])
		Expect(decode(MustBeOk(cache.Bridge(key, bounds.Upper)))).To(Equal(frames[1:3]))
	})

	It(
		"Should empty a channel whose GOP passes the frame bound",
		func(ctx SpecContext) {
			open(ctx, gop.Config{MaxFrames: 3})
			follow(frames...)
			bounds := write(frames[2])
			_, ok := cache.Bridge(key, bounds.Upper)
			Expect(ok).To(BeFalse())
		},
	)

	It("Should empty a channel whose GOP passes the size bound", func(ctx SpecContext) {
		open(ctx, gop.Config{MaxSize: 25})
		follow(frames...)
		bounds := write(frames[2])
		_, ok := cache.Bridge(key, bounds.Upper)
		Expect(ok).To(BeFalse())
	})
})
