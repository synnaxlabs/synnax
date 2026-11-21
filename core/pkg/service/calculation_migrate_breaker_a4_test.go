// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"context"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// storeV062Calculations writes a v0.62 channel v1 record for a boolean reset channel
// and one calculated channel with each list of ops, under keys that the counter of n
// issued. It returns the reset channel, then the calculated channels.
func storeV062Calculations(
	ctx context.Context,
	n testutil.Node,
	ops func(reset channel.Key) [][]channelv1.Operation,
) (channelv1.Channel, []channelv1.Channel) {
	GinkgoHelper()
	count := len(ops(0)) + 1
	reqs := make([]distchannel.Channel, count)
	for i := range reqs {
		reqs[i] = distchannel.Channel{
			Name:        "a4_" + strings.ReplaceAll(uuid.New().String()[:8], "-", ""),
			Leaseholder: node.KeyFree,
			Virtual:     true,
		}
	}
	keys := MustSucceed(n.Channel.Allocate(ctx, reqs))
	stored := make([]channelv1.Channel, count)
	for i := range stored {
		stored[i] = channelv1.Channel{
			Name:        keys[i].Name,
			Leaseholder: node.KeyFree,
			LocalKey:    keys[i].LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
	}
	stored[0].DataType = telem.BooleanT
	for i, o := range ops(stored[0].Key()) {
		stored[i+1].Expression = "return 3.0"
		stored[i+1].Operations = o
	}
	Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
		Set(ctx, stored...)).To(Succeed())
	return stored[0], stored[1:]
}

// oldAPIRead returns the channels with keys as the old channel API shows them to a
// subject that can retrieve every channel and calculation.
func oldAPIRead(
	ctx context.Context,
	n testutil.Node,
	l *service.Layer,
	keys []channel.Key,
) []apichannel.Channel {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(ctx, n.DB, l.RBAC, l.User, user.User{
		Name: "a4-reader-" + uuid.New().String(),
	}))
	MustSucceed(rbactestutil.Grant(
		ctx,
		n.DB,
		l.RBAC,
		l.User,
		u.Key,
		[]access.Action{access.ActionRetrieve},
		resource.ID{Type: resource.TypeChannel},
		resource.ID{Type: resource.TypeCalculation},
	))
	apiChannels := MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
		DB:          n.DB,
		RBAC:        l.RBAC,
		Channel:     l.Channel,
		Status:      l.Status,
		Calculation: l.Calculation,
	}))
	binder := MustSucceed(bind.NewService(bind.ServiceConfig{
		DB:   n.DB,
		RBAC: l.RBAC,
		Held: apiauth.HeldTables(),
	}))
	res := MustSucceed(bind.Read(binder, apiChannels.Retrieve)(
		AuthedCtx(ctx, u),
		apichannel.RetrieveRequest{
			Where: &channel.Where{Key: &filter.Set[channel.Key]{In: keys}},
		},
	))
	byKey := lo.KeyBy(res.Channels, func(ch apichannel.Channel) channel.Key {
		return ch.Key
	})
	return lo.Map(keys, func(k channel.Key, _ int) apichannel.Channel {
		return byKey[k]
	})
}

// apiOperations returns the old API form of v0.62 stored operations.
func apiOperations(ops []channelv1.Operation) []apichannel.Operation {
	return lo.Map(ops, func(op channelv1.Operation, _ int) apichannel.Operation {
		return apichannel.Operation{
			Type:         apichannel.OperationType(op.Type),
			ResetChannel: op.ResetChannel,
			Duration:     op.Duration,
		}
	})
}

var _ = Describe("Calculation records upgrade breaker a4", func() {
	unissued := channel.Key(distchannel.NewKey(node.KeyFree, 900_000))

	It("Should convert every operation type of a v0.62 store", func(ctx SpecContext) {
		var (
			reset channelv1.Channel
			calcs []channelv1.Channel
		)
		n := openUpgraded(ctx, func(n testutil.Node) {
			reset, calcs = storeV062Calculations(
				ctx,
				n,
				func(reset channel.Key) [][]channelv1.Operation {
					return [][]channelv1.Operation{
						{
							{
								Type:     channelv1.OperationTypeMin,
								Duration: 5 * telem.Second,
							},
						},
						{{Type: channelv1.OperationTypeMax, ResetChannel: reset}},
						{{
							Type:         channelv1.OperationTypeAvg,
							ResetChannel: reset,
							Duration:     telem.Second,
						}},
						{{Type: channelv1.OperationTypeDerivative}},
						{{Type: channelv1.OperationTypeNone}},
						{{Type: channelv1.OperationTypeMax, ResetChannel: unissued}},
						{{Duration: telem.Second}},
					}
				},
			)
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		records := lo.Map(
			calcs,
			func(ch channelv1.Channel, _ int) []calculation.Operation {
				return calculationOf(ctx, n, l.Calculation, ch.Key()).Operations
			},
		)
		Expect(records).To(Equal([][]calculation.Operation{
			{{Variant: calculation.MinOperation{
				Window: calculation.Window{Duration: 5 * telem.Second},
			}}},
			{{Variant: calculation.MaxOperation{
				Window: calculation.Window{Reset: new(reset.Key())},
			}}},
			{{Variant: calculation.AvgOperation{
				Window: calculation.Window{
					Reset:    new(reset.Key()),
					Duration: telem.Second,
				},
			}}},
			{{Variant: calculation.DerivativeOperation{}}},
			{},
			{{Variant: calculation.MaxOperation{}}},
			{},
		}))

		read := oldAPIRead(ctx, n, l, lo.Map(
			calcs,
			func(ch channelv1.Channel, _ int) channel.Key { return ch.Key() },
		))
		shown := lo.Map(
			read,
			func(ch apichannel.Channel, _ int) []apichannel.Operation {
				return ch.Operations
			},
		)
		none := []apichannel.Operation{{Type: apichannel.OperationTypeNone}}
		Expect(shown).To(Equal([][]apichannel.Operation{
			apiOperations(calcs[0].Operations),
			apiOperations(calcs[1].Operations),
			apiOperations(calcs[2].Operations),
			apiOperations(calcs[3].Operations),
			none,
			{{Type: apichannel.OperationTypeMax}},
			none,
		}))
		Expect(lo.Map(read, func(ch apichannel.Channel, _ int) string {
			return ch.Expression
		})).To(HaveEach("return 3.0"))

		var statuses []status.Status[any]
		Expect(l.Status.NewRetrieve[any]().Entries(&statuses).Exec(ctx, n.DB)).
			To(Succeed())
		warnings := lo.Filter(statuses, func(s status.Status[any], _ int) bool {
			return s.Message == "Dropped operations with no type"
		})
		Expect(warnings).To(HaveLen(1))
		Expect(warnings[0].Name).To(Equal(calcs[6].Name))
	})

	It(
		"Should show a v0.62 derivative and a none beside another operation as v0.62 did",
		func(ctx SpecContext) {
			var calcs []channelv1.Channel
			n := openUpgraded(ctx, func(n testutil.Node) {
				_, calcs = storeV062Calculations(
					ctx,
					n,
					func(reset channel.Key) [][]channelv1.Operation {
						return [][]channelv1.Operation{
							{{
								Type:         channelv1.OperationTypeDerivative,
								Duration:     5 * telem.Second,
								ResetChannel: reset,
							}},
							{
								{
									Type:     channelv1.OperationTypeAvg,
									Duration: telem.Second,
								},
								{Type: channelv1.OperationTypeNone},
							},
						}
					},
				)
			})
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			read := oldAPIRead(ctx, n, l, []channel.Key{calcs[0].Key(), calcs[1].Key()})
			Expect(read[0].Operations).To(Equal(apiOperations(calcs[0].Operations)))
			Expect(read[1].Operations).To(Equal(apiOperations(calcs[1].Operations)))
		},
	)
})
