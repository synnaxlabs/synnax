// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ReadChannels", Ordered, func() {
	var (
		core     testutil.Node
		channels v0.Channels
		idx      channel.Channel
		data     channel.Channel
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		core = testutil.NewNode(ctx)
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: core.DB}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: core.DB,
		}))
		channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: core.Channel,
			DB:      core.DB,
			HostKey: core.Identity.HostKey,
			Status:  statusSvc,
		}))
		channels = icd.ReadChannels(channelSvc)
		idx = channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: core.Identity.HostKey,
		}
		Expect(core.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &idx)
		})).To(Succeed())
		data = channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float32T,
			Leaseholder: core.Identity.HostKey,
			LocalIndex:  idx.LocalKey,
		}
		Expect(core.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &data)
		})).To(Succeed())
	})

	It(
		"Should return the name, data type, and index of each channel",
		func(ctx SpecContext) {
			found := MustSucceed(channels(
				ctx,
				core.DB,
				[]channel.Key{idx.Key(), data.Key()},
			))
			Expect(found).To(Equal(map[channel.Key]v0.Channel{
				idx.Key():  {Name: idx.Name, DataType: telem.TimestampT, Index: true},
				data.Key(): {Name: data.Name, DataType: telem.Float32T},
			}))
		},
	)

	It("Should leave out a key with no channel", func(ctx SpecContext) {
		missing := distchannel.NewKey(core.Identity.HostKey, 60000)
		Expect(channels(ctx, core.DB, []channel.Key{data.Key(), missing})).
			To(Equal(map[channel.Key]v0.Channel{
				data.Key(): {Name: data.Name, DataType: telem.Float32T},
			}))
	})
})
