// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"io"
	"time"
)

type scenario struct {
	resCount int
	name     string
	keys     channel.Keys
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
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func() { s = _sF() })
			AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func() {
				reader := MustSucceed(s.relay.NewStreamer(context.TODO(), relay.StreamerConfig{
					Keys: s.keys,
				}))
				sCtx, _ := signal.Isolated()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				// We need to give a few milliseconds for the reader to boot up.
				time.Sleep(10 * time.Millisecond)
				writer := MustSucceed(s.writer.New(context.TODO(), writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
				}))
				defer func() {
					defer GinkgoRecover()
					Expect(writer.Close()).To(Succeed())
				}()
				writeF := core.Frame{
					Keys: s.keys,
					Series: []telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				}
				Expect(writer.Write(writeF)).To(BeTrue())
				var f framer.Frame
				for i := 0; i < s.resCount; i++ {
					var res relay.Response
					Eventually(readerRes.Outlet()).Should(Receive(&res))
					f = core.MergeFrames([]core.Frame{f, res.Frame})
				}
				Expect(f.Keys).To(HaveLen(3))
				for i, k := range f.Keys {
					wi := lo.IndexOf(s.keys, k)
					s := f.Series[i]
					ws := writeF.Series[wi]
					Expect(s.Data).To(Equal(ws.Data))
					Expect(s.DataType).To(Equal(ws.DataType))
					Expect(s.Alignment).To(BeNumerically(">", telem.AlignmentPair(0)))
				}
				streamerReq.Close()
				confluence.Drain(readerRes)
			})
		}
	})
})

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
		{
			Name:     "test2",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
		{
			Name:     "test3",
			Rate:     1 * telem.Hz,
			DataType: telem.Int64T,
		},
	}
}

func gatewayOnlyScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(1)
	svc := services[1]
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		resCount: 1,
		name:     "gatewayOnly",
		keys:     keys,
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
		name:     "peerOnly",
		keys:     keys,
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
		keys:     keys,
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
