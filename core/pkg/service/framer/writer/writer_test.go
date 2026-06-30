// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	createIndexed := func(ctx SpecContext) (channel.Channel, channel.Channel) {
		idxCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}
		Expect(channelSvc.Create(ctx, &idxCh)).To(Succeed())
		dataCh := channel.Channel{
			Name:       channel.NewRandomName(),
			DataType:   telem.Float32T,
			LocalIndex: idxCh.LocalKey,
		}
		Expect(channelSvc.Create(ctx, &dataCh)).To(Succeed())
		return idxCh, dataCh
	}

	Describe("ServiceConfig", func() {
		Describe("Override", func() {
			It("Should retain the base value when the override is nil", func() {
				base := writer.ServiceConfig{Framer: node.Framer, Channel: channelSvc}
				res := base.Override(writer.ServiceConfig{})
				Expect(res.Framer).To(Equal(node.Framer))
				Expect(res.Channel).To(Equal(channelSvc))
			})
			It("Should replace the base value when the override is non-nil", func() {
				res := writer.ServiceConfig{}.Override(writer.ServiceConfig{
					Framer:  node.Framer,
					Channel: channelSvc,
				})
				Expect(res.Framer).To(Equal(node.Framer))
				Expect(res.Channel).To(Equal(channelSvc))
			})
			It("Should override zero-value instrumentation", func() {
				ins := alamos.New("writer-test")
				res := writer.ServiceConfig{}.Override(writer.ServiceConfig{
					Instrumentation: ins,
				})
				Expect(res.Instrumentation.IsZero()).To(BeFalse())
			})
			It("Should retain base instrumentation when the override is zero", func() {
				base := writer.ServiceConfig{Instrumentation: alamos.New("writer-test")}
				res := base.Override(writer.ServiceConfig{})
				Expect(res.Instrumentation.IsZero()).To(BeFalse())
			})
		})

		Describe("Validate", func() {
			It("Should return nil for a fully specified configuration", func() {
				cfg := writer.ServiceConfig{Framer: node.Framer, Channel: channelSvc}
				Expect(cfg.Validate()).To(Succeed())
			})
			It("Should return an error when Framer is nil", func() {
				cfg := writer.ServiceConfig{Channel: channelSvc}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("framer")))
			})
			It("Should return an error when Channel is nil", func() {
				cfg := writer.ServiceConfig{Framer: node.Framer}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("channel")))
			})
		})
	})

	Describe("NewService", func() {
		It("Should open a service from a valid configuration", func() {
			Expect(writer.NewService(writer.ServiceConfig{
				Framer:  node.Framer,
				Channel: channelSvc,
			})).Error().ToNot(HaveOccurred())
		})
		It("Should return an error for an invalid configuration", func() {
			Expect(writer.NewService(writer.ServiceConfig{Channel: channelSvc})).
				Error().To(MatchError(ContainSubstring("framer: must be non-nil")))
		})
	})

	Describe("Open", func() {
		It("Should resolve channels by key and write a frame", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			w := MustOpen(writerSvc.Open(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{idxCh.Key(), dataCh.Key()},
			}))
			fr := frame.NewMulti(
				[]channel.Key{idxCh.Key(), dataCh.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(1, 2, 3),
					telem.NewSeriesV[float32](1, 2, 3),
				},
			)
			MustSucceed(w.Write(fr))
			MustSucceed(w.Commit())
		})

		It("Should reject a write whose series data type does not match the resolved channel", func(ctx SpecContext) {
			vCh := channel.Channel{
				Name: channel.NewRandomName(), DataType: telem.Float32T, Virtual: true,
			}
			Expect(channelSvc.Create(ctx, &vCh)).To(Succeed())
			w := MustSucceed(writerSvc.Open(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{vCh.Key()},
				Sync:  new(true),
			}))
			fr := frame.NewUnary(vCh.Key(), telem.NewSeriesV[float64](1, 2, 3))
			Expect(w.Write(fr)).
				Error().To(MatchError(ContainSubstring("expected data type")))
		})

		It("Should return an error when a key has no corresponding channel", func(ctx SpecContext) {
			Expect(writerSvc.Open(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{channel.NewKey(node.Cluster.HostKey(), 9999)},
			})).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should return a validation error when no keys are provided", func(ctx SpecContext) {
			Expect(writerSvc.Open(ctx, writer.Config{Start: telem.SecondTS})).
				Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
	})

	Describe("NewStream", func() {
		It("Should open a stream writer for resolved channels", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			s := MustSucceed(writerSvc.NewStream(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{idxCh.Key(), dataCh.Key()},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			inlet, outlet := confluence.Attach(s)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			inlet.Inlet() <- writer.Request{
				Command: writer.CommandWrite,
				Frame: frame.NewMulti(
					[]channel.Key{idxCh.Key(), dataCh.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3),
						telem.NewSeriesV[float32](1, 2, 3),
					},
				),
			}
			inlet.Close()
			Eventually(outlet.Outlet()).Should(BeClosed())
		})

		It("Should return an error when a key has no corresponding channel", func(ctx SpecContext) {
			Expect(writerSvc.NewStream(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{channel.NewKey(node.Cluster.HostKey(), 9999)},
			})).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should return a validation error when no keys are provided", func(ctx SpecContext) {
			Expect(writerSvc.NewStream(ctx, writer.Config{Start: telem.SecondTS})).
				Error().To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
	})
})
