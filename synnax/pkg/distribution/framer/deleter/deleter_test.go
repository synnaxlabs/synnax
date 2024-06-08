// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter_test

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"io"
)

var _ = Describe("Deleter", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func() scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
			freeWriterScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func() { s = _sF() })
			AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func() {
				writer := MustSucceed(s.writer.New(context.TODO(), writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
				}))
				Expect(writer.Write(core.Frame{
					Keys: s.keys,
					Series: []telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					}},
				)).To(BeTrue())
				Expect(writer.Commit()).To(BeTrue())
				Expect(writer.Error()).ToNot(HaveOccurred())
				Expect(writer.Close()).To(Succeed())

				deleter := s.deleter.NewDeleter(nil)
				Expect(deleter.DeleteTimeRangeMany(ctx, s.keys, (10 * telem.SecondTS).Range(12*telem.SecondTS))).To(Succeed())
			})
		}
	})
})

type scenario struct {
	name    string
	keys    channel.Keys
	writer  *writer.Service
	deleter deleter.Service
	channel channel.Service
	close   io.Closer
}

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
		name:    "gatewayOnly",
		keys:    keys,
		writer:  svc.writer,
		deleter: svc.deleter,
		close:   builder,
		channel: svc.channel,
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
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:    "peerOnly",
		keys:    keys,
		writer:  svc.writer,
		deleter: svc.deleter,
		close:   builder,
		channel: svc.channel,
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
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:    "mixed",
		keys:    keys,
		writer:  svc.writer,
		deleter: svc.deleter,
		close:   builder,
		channel: svc.channel,
	}
}

func freeWriterScenario() scenario {
	channels := newChannelSet()
	builder, services := provision(3)
	svc := services[1]
	for i, ch := range channels {
		ch.Leaseholder = dcore.Free
		ch.Virtual = true
		channels[i] = ch
	}
	Expect(svc.channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:    "freeWriter",
		keys:    keys,
		writer:  svc.writer,
		deleter: svc.deleter,
		close:   builder,
		channel: svc.channel,
	}
}
