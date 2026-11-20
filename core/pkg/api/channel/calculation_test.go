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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation records", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "records-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
		)
	})
	create := func(ctx SpecContext, req apichannel.CreateRequest) apichannel.Channel {
		GinkgoHelper()
		return MustSucceed(createChannels(AuthedCtx(ctx, u), req)).Channels[0]
	}
	recordsOf := func(ctx SpecContext, key channel.Key) []calculation.Calculation {
		GinkgoHelper()
		var out []calculation.Calculation
		Expect(layer.Calculation.NewRetrieve().
			Where(calculation.MatchOutputs(key)).
			Entries(&out).
			Exec(ctx, db)).To(Succeed())
		return out
	}
	calculated := func(input channel.Channel, ops ...channel.Operation) apichannel.Channel {
		return apichannel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s * 2", input.Name),
			Operations: ops,
		}
	}

	It("Should write the record of a calculated channel", func(ctx SpecContext) {
		input := createChannel(ctx, telem.Float32T)
		reset := createChannel(ctx, telem.Uint8T)
		ch := create(ctx, apichannel.CreateRequest{Channels: []apichannel.Channel{
			calculated(
				input,
				channel.Operation{
					Type:     channel.OperationTypeAvg,
					Duration: telem.Second,
				},
				channel.Operation{Type: channel.OperationTypeNone},
				channel.Operation{
					Type:         channel.OperationTypeMax,
					ResetChannel: reset.Key(),
				},
			),
		}})
		records := recordsOf(ctx, ch.Key)
		Expect(records).To(HaveLen(1))
		Expect(records[0].Expression).To(Equal(ch.Expression))
		Expect(records[0].Operations).To(Equal([]calculation.Operation{
			{Variant: calculation.AvgOperation{
				Window: calculation.Window{Duration: telem.Second},
			}},
			{Variant: calculation.MaxOperation{
				Window: calculation.Window{Reset: new(reset.Key())},
			}},
		}))
	})

	It("Should write no record for a channel with no expression", func(
		ctx SpecContext,
	) {
		ch := create(ctx, apichannel.CreateRequest{Channels: []apichannel.Channel{{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}}})
		Expect(recordsOf(ctx, ch.Key)).To(BeEmpty())
	})

	It("Should keep one record when a create finds the channel by name", func(
		ctx SpecContext,
	) {
		input := createChannel(ctx, telem.Float32T)
		req := apichannel.CreateRequest{
			Channels:             []apichannel.Channel{calculated(input)},
			RetrieveIfNameExists: true,
		}
		first := create(ctx, req)
		Expect(create(ctx, req).Key).To(Equal(first.Key))
		Expect(recordsOf(ctx, first.Key)).To(HaveLen(1))
	})

	It("Should update the record when a dispatch sets the expression", func(
		ctx SpecContext,
	) {
		input := createChannel(ctx, telem.Float32T)
		ch := create(ctx, apichannel.CreateRequest{
			Channels: []apichannel.Channel{calculated(input)},
		})
		expression := fmt.Sprintf("return %s * 3", input.Name)
		Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
			Key: ch.Key,
			Actions: []channel.Action{channel.NewSetAction(channel.SetPayload{
				Path:  []string{"expression"},
				Value: expression,
			})},
		})).Error().To(Succeed())
		records := recordsOf(ctx, ch.Key)
		Expect(records).To(HaveLen(1))
		Expect(records[0].Expression).To(Equal(expression))
	})
})
