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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", Ordered, func() {
	var (
		dist       mock.Node
		channelSvc *channel.Service
		wr         *writer.Service
	)
	BeforeAll(func(ctx SpecContext) {
		dist = mock.NewNode(ctx)
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      dist.Channel,
			DB:           dist.DB,
			HostResolver: dist.Cluster,
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			Search:       dist.Search,
		}))
		wr = MustSucceed(writer.NewService(writer.ServiceConfig{
			Framer:  dist.Framer,
			Channel: channelSvc,
		}))
	})

	createIndexed := func(ctx SpecContext) (channel.Channel, channel.Channel) {
		idxCh := channel.Channel{Name: channel.NewRandomName(), DataType: telem.TimeStampT, IsIndex: true}
		Expect(channelSvc.Create(ctx, &idxCh)).To(Succeed())
		dataCh := channel.Channel{Name: channel.NewRandomName(), DataType: telem.Float32T, LocalIndex: idxCh.LocalKey}
		Expect(channelSvc.Create(ctx, &dataCh)).To(Succeed())
		return idxCh, dataCh
	}

	Describe("Open", func() {
		It("Should resolve channels by key and write a frame", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			w := MustOpen(wr.Open(ctx, writer.Config{
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
			vCh := channel.Channel{Name: channel.NewRandomName(), DataType: telem.Float32T, Virtual: true}
			Expect(channelSvc.Create(ctx, &vCh)).To(Succeed())
			w := MustSucceed(wr.Open(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{vCh.Key()},
				Sync:  lo.ToPtr(true),
			}))
			fr := frame.NewUnary(vCh.Key(), telem.NewSeriesV[float64](1, 2, 3))
			Expect(w.Write(fr)).Error().To(MatchError(ContainSubstring("expected data type")))
		})

		It("Should return an error when a key has no corresponding channel", func(ctx SpecContext) {
			Expect(wr.Open(ctx, writer.Config{
				Start: telem.SecondTS,
				Keys:  []channel.Key{channel.NewKey(dist.Cluster.HostKey(), 9999)},
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("NewStream", func() {
		It("Should open a stream writer for resolved channels", func(ctx SpecContext) {
			idxCh, dataCh := createIndexed(ctx)
			s := MustSucceed(wr.NewStream(ctx, writer.Config{
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
	})
})
