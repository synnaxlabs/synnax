// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	setChannelName    = "sy_panel_set"
	deleteChannelName = "sy_panel_delete"
)

var _ = Describe("Signals", func() {
	openStreamer := func(ctx context.Context, name string) confluence.Outlet[framer.StreamerResponse] {
		var ch channel.Channel
		Expect(channelSvc.NewRetrieve().Where(channel.MatchNames(name)).Entry(&ch).
			Exec(ctx, nil)).To(Succeed())
		streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
			Keys: channel.Keys{ch.Key()},
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		shutdown := signal.NewHardShutdown(sCtx, cancel)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		DeferCleanup(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(shutdown.Close()).To(Succeed())
		})
		time.Sleep(10 * time.Millisecond)
		return responses
	}

	It("Should create the set and delete signal channels", func(ctx SpecContext) {
		for _, name := range []string{setChannelName, deleteChannelName} {
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().Where(channel.MatchNames(name)).Entry(&ch).
				Exec(ctx, nil)).To(Succeed())
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Internal).To(BeTrue())
		}
	})

	It(
		"Should broadcast a dispatched action vector on the set channel",
		func(ctx SpecContext) {
			responses := openStreamer(ctx, setChannelName)
			p := panel.Panel{Name: "sig", Parent: &parentID}
			Expect(writer.Create(ctx, &p)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, p.Key)).To(Succeed())
			})
			Expect(writer.Dispatch(ctx, p.Key, "dk-1", []panel.Action{
				panel.NewRenameAction(panel.RenamePayload{Name: "renamed"}),
			})).To(Succeed())
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), time.Second*5).Should(Receive(&res))
			var decoded []actions.Scoped[panel.Key, panel.Action]
			for sample := range res.Frame.SeriesAt(0).Samples() {
				var sa actions.Scoped[panel.Key, panel.Action]
				Expect(json.Unmarshal(sample, &sa)).To(Succeed())
				decoded = append(decoded, sa)
			}
			Expect(decoded).ToNot(BeEmpty())
			last := decoded[len(decoded)-1]
			Expect(last.Key).To(Equal(p.Key))
			Expect(last.DispatchKey).To(Equal("dk-1"))
			Expect(last.Actions).To(HaveLen(1))
			Expect(last.Actions[0].Type).To(Equal(panel.ActionTypeRename))
		},
	)

	It(
		"Should emit the deleted panel key on the delete channel",
		func(ctx SpecContext) {
			responses := openStreamer(ctx, deleteChannelName)
			p := panel.Panel{Name: "sig-del", Parent: &parentID}
			Expect(writer.Create(ctx, &p)).To(Succeed())
			Expect(writer.Delete(ctx, p.Key)).To(Succeed())
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), time.Second*5).Should(Receive(&res))
			var keys []uuid.UUID
			for sample := range res.Frame.SeriesAt(0).Samples() {
				keys = append(keys, MustSucceed(uuid.FromBytes(sample)))
			}
			Expect(keys).To(ContainElement(p.Key))
		},
	)
})
