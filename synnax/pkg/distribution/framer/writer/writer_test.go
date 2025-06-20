// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
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
			AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func() {
				writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  s.keys,
					Start: 10 * telem.SecondTS,
					Sync:  config.True(),
				}))
				MustSucceed(writer.Write(core.MultiFrame(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				)))
				MustSucceed(writer.Commit())
				MustSucceed(writer.Write(core.MultiFrame(
					s.keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				)))
				MustSucceed(writer.Commit())
				Expect(writer.Close()).To(Succeed())
			})
		}
	})
	Describe("Open Errors", Ordered, func() {
		var s scenario
		BeforeAll(func() { s = gatewayOnlyScenario() })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		It("Should return an error if no keys are provided", func() {
			_, err := s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  []channel.Key{},
				Start: 10 * telem.SecondTS,
				Sync:  config.True(),
			})
			Expect(err).To(MatchError(ContainSubstring("keys: must be non-empty")))
		})
		It("Should return an error if the channel can't be found", func() {
			_, err := s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys: []channel.Key{
					channel.NewKey(0, 22),
					s.keys[0],
				},
				Start: 10 * telem.SecondTS,
				Sync:  config.True(),
			})
			Expect(err).To(HaveOccurredAs(query.NotFound))
			Expect(err.Error()).To(ContainSubstring("Channel"))
			Expect(err.Error()).To(ContainSubstring("22"))
			Expect(err.Error()).ToNot(ContainSubstring("1"))
		})
	})
	Describe("Frame Errors", Ordered, func() {
		var s scenario
		BeforeAll(func() { s = peerOnlyScenario() })
		AfterAll(func() { Expect(s.closer.Close()).To(Succeed()) })
		It("Should return an error if a key is provided that is not in the list of keys provided to the writer", func() {
			writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  s.keys,
				Start: 10 * telem.SecondTS,
				Sync:  config.True(),
			}))
			_, err := writer.Write(core.MultiFrame(
				append(s.keys, channel.NewKey(12, 22)),
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](3, 4, 5),
					telem.NewSeriesV[int64](5, 6, 7),
					telem.NewSeriesV[int64](5, 6, 7),
				},
			))
			Expect(err).To(HaveOccurredAs(validate.Error))
			Expect(writer.Close()).To(HaveOccurredAs(validate.Error))
		})
	})
})

type scenario struct {
	name   string
	keys   channel.Keys
	dist   mock.Node
	closer io.Closer
}

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
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Gateway Only", keys: keys, dist: dist, closer: builder}
}

func peerOnlyScenario() scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKey(i + 2)
		channels[i] = ch
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Peer Only", keys: keys, dist: dist, closer: builder}
}

func mixedScenario() scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 3)
	svc := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKey(i + 1)
		channels[i] = ch
	}
	Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Mixed Gateway and Peer", keys: keys, dist: svc, closer: builder}
}

func freeWriterScenario() scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 3)
	svc := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.Free
		ch.Virtual = true
		channels[i] = ch
	}
	Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := svc.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Free Writes", keys: keys, dist: svc, closer: builder}
}
