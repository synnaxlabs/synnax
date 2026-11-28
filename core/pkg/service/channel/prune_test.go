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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Untyped channel deletion", func() {
	It("Should delete each unreferenced channel with no data type and keep the rest",
		func(ctx SpecContext) {
			dist := testutil.NewNode(ctx)
			free := func(name string, local channel.LocalKey) channel.Channel {
				return channel.Channel{
					Name:        name,
					Leaseholder: node.KeyFree,
					LocalKey:    local,
					Virtual:     true,
				}
			}
			unreferenced := free("unreferenced", 7101)
			typed := free("typed", 7102)
			typed.DataType = telem.Float64T
			calculated := free("calculated", 7103)
			calculated.Expression = "return 1"
			restricted := free("restricted", 7104)
			restricted.IsIndex = true
			holder := free("holder", 7105)
			holder.DataType = telem.Float64T
			holder.LocalIndex = restricted.LocalKey
			createStored(
				ctx, dist.DB, unreferenced, typed, calculated, restricted, holder,
			)
			core, logs := observer.New(zapcore.WarnLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			svc := openService(ctx, dist)
			ins := alamos.New("test", alamos.WithLogger(logger))
			Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.UntypedDeleteMigration().Run(ctx, tx, ins)
			})).To(Succeed())
			var stored []channel.Channel
			Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
				Entries(&stored).
				Exec(ctx, dist.DB)).To(Succeed())
			keys := channel.KeysFromChannels(stored)
			Expect(keys).To(ContainElements(
				typed.Key(), calculated.Key(), restricted.Key(), holder.Key(),
			))
			Expect(keys).NotTo(ContainElement(unreferenced.Key()))
			deleted := logs.FilterMessage("deleted a channel with no data type").All()
			Expect(deleted).To(HaveLen(1))
			Expect(deleted[0].ContextMap()).To(HaveKeyWithValue("name", "unreferenced"))
			kept := logs.FilterMessage(
				"kept a channel with no data type that another entry references",
			).All()
			Expect(kept).To(HaveLen(1))
			Expect(kept[0].ContextMap()).To(HaveKeyWithValue("name", "restricted"))
			Expect(kept[0].ContextMap()).To(HaveKeyWithValue(
				"holder", "channel "+holder.Key().String(),
			))
		},
	)
})
