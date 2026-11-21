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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Description", func() {
	It("Should keep the description of a channel through create and retrieve", func(
		ctx SpecContext,
	) {
		created := MustSucceed(createChannels(
			AuthedCtx(ctx, author),
			apichannel.CreateRequest{Channels: []apichannel.Channel{{
				Name:        UniqueChannelName(),
				DataType:    telem.Float32T,
				Virtual:     true,
				Description: "Pressure at the inlet of the main tank",
			}}},
		)).Channels
		Expect(
			created[0].Description,
		).To(Equal("Pressure at the inlet of the main tank"))
		retrieved := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, author),
			apichannel.RetrieveRequest{Where: &channel.Where{
				Key: &filter.Set[channel.Key]{Eq: &created[0].Key},
			}},
		)).Channels
		Expect(
			retrieved[0].Description,
		).To(Equal("Pressure at the inlet of the main tank"))
	})
})
