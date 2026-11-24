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
	"encoding/json/v2"
	"fmt"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Old API edits that let a subject read a calculation", func() {
	type batch = actions.Scoped[channel.Key, channel.Action]
	var (
		ch     channel.Channel
		client freighter.ClientStream[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		]
		oldAPI   *apichannel.Service
		retrieve freighter.UnaryHandler[
			apichannel.RetrieveRequest,
			apichannel.RetrieveResponse,
		]
	)
	BeforeEach(func(ctx SpecContext) {
		oldAPI = MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
			DB:          db,
			RBAC:        svc.RBAC,
			Channel:     svc.Channel,
			Status:      svc.Status,
			Calculation: svc.Calculation,
			Holders:     svc.TaskHolders,
			Impact:      svc.Impact,
		}))
		binder := MustSucceed(bind.NewService(bind.ServiceConfig{
			DB:      db,
			RBAC:    svc.RBAC,
			Held:    apiauth.HeldTables(),
			Holders: svc.TaskHolders,
		}))
		retrieve = bind.Read(binder, oldAPI.Retrieve)
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchNames("sy_channel_action_set")).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
	})
	// stream opens a stream of the channel action channel as a new user that may
	// retrieve it and granted.
	stream := func(ctx SpecContext, granted ...resource.ID) user.User {
		GinkgoHelper()
		u := createUserGranted(
			ctx,
			append(svcframer.ResourceIDs(channel.Keys{ch.Key()}), granted...)...,
		)
		var server apiframer.StreamerStream
		client, server = freightertestutil.NewStreams[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		](ctx)
		done := make(chan error, 1)
		go func() { done <- apiSvc.Stream(subjectCtx(ctx, u.ResourceID()), server) }()
		DeferCleanup(func() {
			Expect(client.CloseSend()).To(Succeed())
			Eventually(done).WithTimeout(5 * time.Second).Should(Receive())
		})
		Expect(client.Send(apiframer.StreamerRequest{Keys: channel.Keys{ch.Key()}})).
			To(Succeed())
		Expect(MustSucceed(client.Receive()).Frame.Empty()).To(BeTrue())
		return u
	}
	// until returns the batches that the stream receives up to and with the first batch
	// for witness.
	until := func(witness channel.Key) []batch {
		GinkgoHelper()
		var out []batch
		for {
			res := MustSucceed(client.Receive())
			seen := false
			for _, series := range res.Frame.Get(ch.Key()).Series {
				for sample := range series.Samples() {
					var b batch
					Expect(json.Unmarshal(sample, &b)).To(Succeed())
					out = append(out, b)
					seen = seen || b.Key == witness
				}
			}
			if seen {
				return out
			}
		}
	}
	onKey := func(batches []batch, key channel.Key) []batch {
		return slices.DeleteFunc(slices.Clone(batches), func(b batch) bool {
			return b.Key != key
		})
	}
	setsPath := func(path string) func(b batch) bool {
		return func(b batch) bool {
			return slices.ContainsFunc(b.Actions, func(a channel.Action) bool {
				return a.Set != nil && slices.Equal(a.Set.Path, []string{path})
			})
		}
	}
	virtual := func(ctx SpecContext, dt telem.DataType) channel.Channel {
		GinkgoHelper()
		c := channel.Channel{Name: UniqueChannelName(), DataType: dt, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		return c
	}
	double := func(c channel.Channel) string {
		return fmt.Sprintf("return %s * 2", c.Name)
	}
	// createOutput creates c with a new output and returns the output.
	createOutput := func(ctx SpecContext, c calculation.Calculation) channel.Channel {
		GinkgoHelper()
		var created []channel.Channel
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			created, err = svc.Calculation.NewWriter(tx).
				CreateOutput(ctx, &c, UniqueChannelName(), nil)
			return err
		})).To(Succeed())
		return created[0]
	}
	rename := func(ctx SpecContext, witness channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).
				Rename(ctx, witness.Key(), UniqueChannelName(), false)
		})).To(Succeed())
	}
	set := func(path string, value any) channel.Action {
		return channel.NewSetAction(
			channel.SetPayload{Path: []string{path}, Value: value},
		)
	}
	// dispatchOld applies acts to the channel with key through the old channel API, as
	// root.
	dispatchOld := func(ctx SpecContext, key channel.Key, acts ...channel.Action) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := oldAPI.Dispatch(rootCtx(ctx), tx, apichannel.DispatchRequest{
				Key:     key,
				Actions: acts,
			})
			return err
		})).To(Succeed())
	}
	// retrieveAs returns the old API form of the channel with key, as u retrieves it.
	retrieveAs := func(
		ctx SpecContext,
		u user.User,
		key channel.Key,
	) apichannel.Channel {
		GinkgoHelper()
		res := MustSucceed(retrieve(
			subjectCtx(ctx, u.ResourceID()),
			apichannel.RetrieveRequest{
				Where: &channel.Where{Key: &filter.Set[channel.Key]{Eq: &key}},
			},
		))
		Expect(res.Channels).To(HaveLen(1))
		return res.Channels[0]
	}
	calcType := resource.ID{Type: resource.TypeCalculation}

	It(
		"Should send the expression with the operations to a subject that an old API "+
			"operations edit lets read the calculation",
		func(ctx SpecContext) {
			in := virtual(ctx, telem.Float32T)
			secret := virtual(ctx, telem.BooleanT)
			witness := virtual(ctx, telem.Float32T)
			output := createOutput(ctx, calculation.Calculation{
				Expression: double(in),
				Operations: []calculation.Operation{{
					Variant: calculation.MaxOperation{
						Window: calculation.Window{Reset: new(secret.Key())},
					},
				}},
			})
			reader := stream(
				ctx,
				calcType,
				channel.ResourceID(in.Key()),
				channel.ResourceID(output.Key()),
				channel.ResourceID(witness.Key()),
			)
			before := retrieveAs(ctx, reader, output.Key())
			Expect(before.Calculated).To(BeTrue())
			Expect(before.Expression).To(BeEmpty())
			dispatchOld(ctx, output.Key(), set("operations", []channel.Operation{}))
			Expect(retrieveAs(ctx, reader, output.Key()).Expression).
				To(Equal(double(in)))
			rename(ctx, witness)
			announced := onKey(until(witness.Key()), output.Key())
			Expect(announced).ToNot(BeEmpty())
			Expect(announced).To(ContainElement(Satisfy(setsPath("expression"))))
		},
		SpecTimeout(30*time.Second),
	)

	It(
		"Should send the operations with the expression to a subject that an old API "+
			"expression edit lets read the calculation",
		func(ctx SpecContext) {
			in, secret, witness := virtual(ctx, telem.Float32T),
				virtual(ctx, telem.Float32T),
				virtual(ctx, telem.Float32T)
			output := createOutput(ctx, calculation.Calculation{
				Expression: double(secret),
				Operations: []calculation.Operation{{
					Variant: calculation.AvgOperation{
						Window: calculation.Window{Duration: telem.Second},
					},
				}},
			})
			reader := stream(
				ctx,
				calcType,
				channel.ResourceID(in.Key()),
				channel.ResourceID(output.Key()),
				channel.ResourceID(witness.Key()),
			)
			Expect(retrieveAs(ctx, reader, output.Key()).Operations).To(BeEmpty())
			dispatchOld(ctx, output.Key(), set("expression", double(in)))
			Expect(retrieveAs(ctx, reader, output.Key()).Operations).To(HaveLen(1))
			rename(ctx, witness)
			announced := onKey(until(witness.Key()), output.Key())
			Expect(announced).ToNot(BeEmpty())
			Expect(announced).To(ContainElement(Satisfy(setsPath("operations"))))
		},
		SpecTimeout(30*time.Second),
	)
})
