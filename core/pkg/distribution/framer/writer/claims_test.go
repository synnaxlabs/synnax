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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete Claims", func() {
	var (
		host      testutil.Node
		idx, data channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		host = testutil.NewNode(ctx)
		idx = MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
			Name:     "claim_time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		}}))[0]
		data = MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
			Name:       "claim",
			DataType:   telem.Int64T,
			LocalIndex: idx.LocalKey,
		}}))[0]
	})
	open := func(ctx SpecContext, cfg writer.Config) (*writer.Writer, error) {
		cfg.Start = 10 * telem.SecondTS
		return host.Framer.OpenWriter(ctx, cfg)
	}

	It(
		"Should fail a claim while a writer holds one of the keys",
		func(ctx SpecContext) {
			w := MustSucceed(open(ctx, writer.Config{Keys: channel.Keys{data.Key()}}))
			Expect(host.Framer.ClaimDelete(channel.Keys{idx.Key(), data.Key()})).
				Error().To(SatisfyAll(
				MatchError(writer.ErrChannelInUse),
				MatchError(ContainSubstring("1 unclosed writers")),
			))
			Expect(w.Close()).To(Succeed())
			release := MustSucceed(host.Framer.ClaimDelete(channel.Keys{data.Key()}))
			release()
		},
	)

	It(
		"Should fail a writer open while a claim holds one of the keys",
		func(ctx SpecContext) {
			release := MustSucceed(host.Framer.ClaimDelete(channel.Keys{data.Key()}))
			Expect(open(ctx, writer.Config{Keys: channel.Keys{data.Key()}})).
				Error().To(MatchError(writer.ErrChannelDeleting))
			release()
			Expect(MustSucceed(open(ctx, writer.Config{
				Keys: channel.Keys{data.Key()},
			})).Close()).To(Succeed())
		},
	)

	It(
		"Should release a claim once however often its release runs",
		func(ctx SpecContext) {
			first := MustSucceed(host.Framer.ClaimDelete(channel.Keys{data.Key()}))
			second := MustSucceed(host.Framer.ClaimDelete(channel.Keys{data.Key()}))
			first()
			first()
			Expect(open(ctx, writer.Config{Keys: channel.Keys{data.Key()}})).
				Error().To(MatchError(writer.ErrChannelDeleting))
			second()
			Expect(MustSucceed(open(ctx, writer.Config{
				Keys: channel.Keys{data.Key()},
			})).Close()).To(Succeed())
		},
	)

	It(
		"Should count the index that AutoIndex adds as held by the writer",
		func(ctx SpecContext) {
			w := MustSucceed(open(ctx, writer.Config{
				Keys:      channel.Keys{data.Key()},
				AutoIndex: new(true),
			}))
			Expect(host.Framer.ClaimDelete(channel.Keys{idx.Key()})).
				Error().To(MatchError(writer.ErrChannelInUse))
			Expect(w.Close()).To(Succeed())
		},
	)

	It("Should not count a writer of free channels", func(ctx SpecContext) {
		s := freeWriterScenario(ctx)
		defer func() { Expect(s.Close()).To(Succeed()) }()
		w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
			Keys:  s.keys,
			Start: 10 * telem.SecondTS,
		}))
		release := MustSucceed(s.dist.Framer.ClaimDelete(s.keys))
		release()
		Expect(w.Close()).To(Succeed())
	})
})
