// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package dispatch_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/dispatch"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func run[K comparable, A any](
	ctx context.Context,
	handle dispatch.Handler[K, A],
	req actions.DispatchRequest[K, A],
) (res actions.DispatchResponse[A], err error) {
	err = db.WithTx(ctx, func(tx gorp.Tx) error {
		res, err = handle(ctx, tx, req)
		return err
	})
	return res, err
}

var _ = Describe("New", func() {
	Describe("Rack", func() {
		var (
			handle dispatch.Handler[rack.Key, rack.Action]
			rec    *Recorder[rack.Key, rack.Action]
			r      rack.Rack
		)
		BeforeEach(func(ctx SpecContext) {
			handle = dispatch.New(svc.Rack.NewWriter, rack.Writer.Apply)
			r = rack.Rack{Name: "Rack"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Rack.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			rec = &Recorder[rack.Key, rack.Action]{}
			DeferCleanup(svc.Rack.Actions().OnChange(rec.Record))
		})

		It("Should reply with the actions it broadcasts", func(ctx SpecContext) {
			rename := rack.NewSetAction(
				rack.SetPayload{Path: []string{"name"}, Value: "Renamed"},
			)
			res := MustSucceed(
				run(ctx, handle, actions.DispatchRequest[rack.Key, rack.Action]{
					Key:         r.Key,
					DispatchKey: "dk",
					Actions:     []rack.Action{rename},
				}),
			)
			Expect(res.Actions).To(Equal([]rack.Action{rename}))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(r.Key))
			Expect(frames[0].DispatchKey).To(Equal("dk"))
			Expect(frames[0].Actions).To(Equal(res.Actions))
		})

		It("Should reply with nothing when the apply fails", func(ctx SpecContext) {
			res, err := run(ctx, handle, actions.DispatchRequest[rack.Key, rack.Action]{
				Key:         r.Key + 1000,
				DispatchKey: "dk",
				Actions: []rack.Action{rack.NewSetAction(
					rack.SetPayload{Path: []string{"name"}, Value: "Renamed"},
				)},
			})
			Expect(err).To(MatchError(query.ErrNotFound))
			Expect(res).To(BeZero())
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})

	It("Should reply with the actions that only the Core computes", func(
		ctx SpecContext,
	) {
		handle := dispatch.New(svc.Channel.NewWriter, channel.Writer.Apply)
		ch := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return 1 + 1",
			Virtual:    true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		rec := &Recorder[channel.Key, channel.Action]{}
		DeferCleanup(svc.Channel.Actions().OnChange(rec.Record))
		edit := channel.NewSetAction(channel.SetPayload{
			Path:  []string{"expression"},
			Value: "return 1.5 * 2.0",
		})
		res := MustSucceed(run(
			ctx,
			handle,
			actions.DispatchRequest[channel.Key, channel.Action]{
				Key:         ch.Key(),
				DispatchKey: "dk",
				Actions:     []channel.Action{edit},
			},
		))
		Expect(res.Actions).To(Equal([]channel.Action{
			edit,
			channel.NewSetAction(channel.SetPayload{
				Path:  []string{"data_type"},
				Value: telem.Float64T,
			}),
		}))
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].DispatchKey).To(Equal("dk"))
		Expect(frames[0].Actions).To(Equal(res.Actions))
	})
})
