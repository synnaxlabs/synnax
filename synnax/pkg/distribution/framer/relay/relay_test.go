// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	"context"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type scenario struct {
	resCount int
	name     string
	channels []channel.Channel
	relay    *relay.Relay
	writer   *writer.Service
	close    io.Closer
}

var _ = Describe("Relay", func() {
	Describe("Happy Path", Ordered, func() {

		scenarios := []func() scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
			freeScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func() { s = _sF() })
			AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func() {
				keys := channel.KeysFromChannels(s.channels)
				reader := MustSucceed(s.relay.NewStreamer(context.TODO(), relay.StreamerConfig{
					Keys: keys,
				}))
				sCtx, _ := signal.Isolated()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				// We need to give a few milliseconds for the reader to boot up.
				time.Sleep(10 * time.Millisecond)
				w := MustSucceed(s.writer.Open(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				defer func() {
					defer GinkgoRecover()
					Expect(w.Close()).To(Succeed())
				}()
				writeF := core.MultiFrame(
					keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				)
				Expect(w.Write(writeF)).To(BeTrue())
				var f framer.Frame
				for range s.resCount {
					var res relay.Response
					Eventually(readerRes.Outlet()).Should(Receive(&res))
					f = core.MergeFrames([]core.Frame{f, res.Frame})
				}
				Expect(f.Count()).To(Equal(3))
				for i, k := range f.KeysSlice() {
					wi := lo.IndexOf(keys, k)
					ch := s.channels[wi]
					s := f.SeriesAt(i)
					ws := writeF.SeriesAt(wi)
					Expect(s.Data).To(Equal(ws.Data))
					Expect(s.DataType).To(Equal(ws.DataType))
					if !ch.Free() {
						Expect(s.Alignment).To(BeNumerically(">", telem.Alignment(0)))
					}
				}
				streamerReq.Close()
				confluence.Drain(readerRes)
			})
		}
	})
	Describe("Errors", func() {
		It("Should raise an error if a channel is not found", func() {
			builder, services := provision(1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := services[1]
			_, err := svc.relay.NewStreamer(context.TODO(), relay.StreamerConfig{
				Keys: []channel.Key{12345},
			})
			Expect(err).To(HaveOccurredAs(query.NotFound))
		})
	})
})

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test2",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test3",
			Virtual:  true,
			DataType: telem.Int64T,
		},
	}
}

func gatewayOnlyScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(1)
	svc := services[1]
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	return scenario{
		resCount: 1,
		name:     "gateway-only",
		channels: channels,
		relay:    svc.relay,
		writer:   svc.writer,
		close: xio.CloserFunc(func() error {
			e := errors.NewCatcher(errors.WithAggregation())
			e.Exec(builder.Close)
			for _, svc := range services {
				e.Exec(svc.relay.Close)
			}
			return e.Error()
		}),
	}
}

func peerOnlyScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(4)
	svc := services[1]
	for i, ch := range channels {
		ch.Leaseholder = dcore.NodeKey(i + 2)
		channels[i] = ch
	}
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(svc.channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	return scenario{
		resCount: 3,
		name:     "peer-only",
		channels: channels,
		relay:    svc.relay,
		writer:   svc.writer,
		close: xio.CloserFunc(func() error {
			e := errors.NewCatcher(errors.WithAggregation())
			e.Exec(builder.Close)
			for _, svc := range services {
				e.Exec(svc.relay.Close)
			}
			return e.Error()
		}),
	}
}
func mixedScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(3)
	svc := services[1]
	for i, ch := range channels {
		ch.Leaseholder = dcore.NodeKey(i + 1)
		channels[i] = ch
	}
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(svc.channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	return scenario{
		resCount: 3,
		name:     "mixed",
		channels: channels,
		relay:    svc.relay,
		writer:   svc.writer,
		close: xio.CloserFunc(func() error {
			e := errors.NewCatcher(errors.WithAggregation())
			e.Exec(builder.Close)
			for _, svc := range services {
				e.Exec(svc.relay.Close)
			}
			return e.Error()
		}),
	}
}

func freeScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(1)
	svc := services[1]
	for i, ch := range channels {
		ch.Leaseholder = dcore.Free
		ch.Virtual = true
		channels[i] = ch
	}
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(svc.channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).
			Exec(ctx, nil)).To(Succeed())
	})
	return scenario{
		resCount: 1,
		name:     "free",
		channels: channels,
		relay:    svc.relay,
		writer:   svc.writer,
		close: xio.CloserFunc(func() error {
			e := errors.NewCatcher(errors.WithAggregation())
			e.Exec(builder.Close)
			for _, svc := range services {
				e.Exec(svc.relay.Close)
			}
			return e.Error()
		}),
	}
}
