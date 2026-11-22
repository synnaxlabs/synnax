// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Checker built-in neighbors", func() {
	It("Should rename the free and leased keys around a built-in, and refuse the "+
		"built-ins", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		cfg, _ := serviceConfig(ctx, n, n.DB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		write := func(f func(w channel.Writer) error) error {
			return n.DB.WithTx(
				ctx,
				func(tx gorp.Tx) error { return f(svc.NewWriter(tx)) },
			)
		}
		free := func() channel.Channel {
			return channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.JSONT,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
		}
		index := func() channel.Channel {
			return channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
		}
		create := func(chs ...*channel.Channel) {
			GinkgoHelper()
			for _, ch := range chs {
				Expect(write(func(w channel.Writer) error {
					return w.Create(ctx, ch)
				})).To(Succeed())
			}
		}
		before, beforeIndex := free(), index()
		create(&before, &beforeIndex)
		builtInFree, builtInLeased := free(), index()
		for _, ch := range []*channel.Channel{&builtInFree, &builtInLeased} {
			Expect(write(func(w channel.Writer) error {
				return w.Declare(ctx, ch)
			})).To(Succeed())
		}
		after, afterIndex := free(), index()
		create(&after, &afterIndex)
		Expect(builtInFree.Key().LocalKey()).To(Equal(before.Key().LocalKey() + 1))
		Expect(after.Key().LocalKey()).To(Equal(builtInFree.Key().LocalKey() + 1))
		rename := func(ch channel.Channel) error {
			return write(func(w channel.Writer) error {
				return w.Rename(ctx, ch.Key(), UniqueChannelName(), false)
			})
		}
		for _, ch := range []channel.Channel{before, beforeIndex, after, afterIndex} {
			Expect(rename(ch)).To(Succeed(), "channel %d is not built-in", ch.Key())
		}
		for _, ch := range []channel.Channel{builtInFree, builtInLeased} {
			Expect(rename(ch)).To(MatchError(validate.ErrValidation))
		}
	})
})
