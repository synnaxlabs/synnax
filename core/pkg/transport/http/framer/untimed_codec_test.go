// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Untimed codec state", func() {
	It("Should keep the codec state when a request changes only its untimed keys", func(
		ctx SpecContext,
	) {
		channels := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.Int64T, Virtual: true},
			{Name: UniqueChannelName(), DataType: telem.Int64T, Virtual: true},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &channels)
		})).To(Succeed())
		keys := channel.KeysFromChannels(channels)
		server := framer.NewCodec(ctx, codec.NewDynamic(apiChannelSvc))
		request := func(req framer.StreamerRequest) {
			GinkgoHelper()
			encoded := MustSucceed(server.Encode(http.WSMessage[framer.StreamerRequest]{
				Type:    http.WSMessageTypeData,
				Payload: req,
			}))
			var msg http.WSMessage[framer.StreamerRequest]
			Expect(server.Decode(encoded, &msg)).To(Succeed())
		}
		respond := func(applied int) []byte {
			GinkgoHelper()
			return MustSucceed(server.Encode(http.WSMessage[framer.StreamerResponse]{
				Type: http.WSMessageTypeData,
				Payload: framer.StreamerResponse{
					Frame: frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesV[int64](1),
						telem.NewSeriesV[int64](2),
					}),
					Applied: applied,
				},
			}))
		}
		request(framer.StreamerRequest{Keys: keys})
		respond(0)
		steady := respond(0)
		request(framer.StreamerRequest{Keys: keys, UntimedKeys: keys[:1]})
		Expect(respond(1)).To(Equal(steady))
	})
})
