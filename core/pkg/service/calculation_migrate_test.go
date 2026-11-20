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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation grants upgrade", func() {
	It("Should grant calculations to a custom policy that grants channels", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
			{Name: "doubled", Leaseholder: node.KeyFree, Virtual: true},
		}))
		doubled := channelv1.Channel{
			Name:        keys[0].Name,
			Leaseholder: node.KeyFree,
			LocalKey:    keys[0].LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
			Expression:  "return 2",
		}
		channels := gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB)
		Expect(channels.Set(ctx, doubled)).To(Succeed())
		// v0.62 stores role and policy records, and their chains have run.
		roles := MustSucceed(role.OpenService(ctx, role.ServiceConfig{DB: n.DB}))
		policies := MustSucceed(policy.OpenService(ctx, policy.ServiceConfig{
			DB:   n.DB,
			Role: roles,
		}))
		r := role.Role{Name: "Channels only"}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return roles.NewWriter(tx, false).Create(ctx, &r)
		})).To(Succeed())
		Expect(policies.Close()).To(Succeed())
		Expect(roles.Close()).To(Succeed())
		byType := policy.Policy{
			Key:     uuid.New(),
			Name:    "every channel",
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{access.ActionRetrieve},
			Role:    r.Key,
		}
		byKey := policy.Policy{
			Key:     uuid.New(),
			Name:    "doubled",
			Objects: []resource.ID{channel.ResourceID(doubled.Key())},
			Actions: []access.Action{access.ActionUpdate},
			Role:    r.Key,
		}
		stored := gorp.WrapWriter[policy.Key, policy.Policy](n.DB)
		Expect(stored.Set(ctx, byType, byKey)).To(Succeed())

		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		objectsOf := func(key policy.Key) []resource.ID {
			GinkgoHelper()
			var p policy.Policy
			Expect(l.RBAC.Policy.NewRetrieve().
				Where(policy.MatchKeys(key)).
				Entry(&p).
				Exec(ctx, n.DB)).To(Succeed())
			return p.Objects
		}
		Expect(objectsOf(byType.Key)).To(ConsistOf(
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
		))
		Expect(objectsOf(byKey.Key)).To(ConsistOf(
			channel.ResourceID(doubled.Key()),
			calculation.ResourceID(
				calculationOf(ctx, n, l.Calculation, doubled.Key()).Key,
			),
		))
	})
})

var _ = Describe("Calculated channels of a channel-only subject after upgrade", func() {
	It("Should let the subject create, retrieve, and edit through both APIs", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
			{Name: "doubled", Leaseholder: node.KeyFree, Virtual: true},
		}))
		doubled := channelv1.Channel{
			Name:        keys[0].Name,
			Leaseholder: node.KeyFree,
			LocalKey:    keys[0].LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
			Expression:  "return 2",
		}
		channels := gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB)
		Expect(channels.Set(ctx, doubled)).To(Succeed())
		roles := MustSucceed(role.OpenService(ctx, role.ServiceConfig{DB: n.DB}))
		policies := MustSucceed(policy.OpenService(ctx, policy.ServiceConfig{
			DB:   n.DB,
			Role: roles,
		}))
		r := role.Role{Name: "Channel editors"}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return roles.NewWriter(tx, false).Create(ctx, &r)
		})).To(Succeed())
		Expect(policies.Close()).To(Succeed())
		Expect(roles.Close()).To(Succeed())
		stored := gorp.WrapWriter[policy.Key, policy.Policy](n.DB)
		Expect(stored.Set(ctx, policy.Policy{
			Key:     uuid.New(),
			Name:    "every channel",
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{
				access.ActionCreate,
				access.ActionRetrieve,
				access.ActionUpdate,
			},
			Role: r.Key,
		})).To(Succeed())

		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		var u user.User
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			u, err = l.User.NewWriter(tx).Create(ctx, user.User{
				Name: "editor-" + uuid.New().String(),
				Role: r.Key,
			})
			return err
		})).To(Succeed())
		apiChannels := MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
			DB:          n.DB,
			RBAC:        l.RBAC,
			Channel:     l.Channel,
			Status:      l.Status,
			Calculation: l.Calculation,
		}))
		apiCalculations := MustSucceed(apicalculation.NewService(
			apicalculation.ServiceConfig{Calculation: l.Calculation, Status: l.Status},
		))
		binder := MustSucceed(bind.NewService(bind.ServiceConfig{
			DB:   n.DB,
			RBAC: l.RBAC,
			Held: apiauth.HeldTables(),
		}))
		authed := AuthedCtx(ctx, u)

		created := MustSucceed(bind.Write(binder, apiChannels.Create)(
			authed,
			apichannel.CreateRequest{Channels: []apichannel.Channel{{
				Name:       "tripled",
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return 3",
			}}},
		)).Channels[0]
		retrieved := MustSucceed(bind.Read(binder, apiCalculations.Retrieve)(
			authed,
			apicalculation.RetrieveRequest{},
		)).Calculations
		outputs := make([]channel.Key, len(retrieved))
		for i, c := range retrieved {
			outputs[i] = c.Output
		}
		Expect(outputs).To(ContainElements(doubled.Key(), created.Key))

		Expect(bind.Write(binder, apiChannels.Dispatch)(
			authed,
			apichannel.DispatchRequest{
				Key: doubled.Key(),
				Actions: []channel.Action{channel.NewSetAction(channel.SetPayload{
					Path:  []string{"expression"},
					Value: "return 4",
				})},
			},
		)).Error().To(Succeed())
		set := calculation.NewSetAction(
			calculation.SetPayload{Path: []string{"expression"}, Value: "return 5"},
		)
		Expect(bind.Write(binder, apiCalculations.Dispatch)(
			authed,
			apicalculation.DispatchRequest{
				Key:     calculationOf(ctx, n, l.Calculation, doubled.Key()).Key,
				Actions: []calculation.Action{set},
			},
		)).Error().To(Succeed())
	})
})

// calculationOf returns the calculation that writes the channel with key output.
func calculationOf(
	ctx context.Context,
	n testutil.Node,
	calculations *calculation.Service,
	output channel.Key,
) calculation.Calculation {
	GinkgoHelper()
	var c calculation.Calculation
	Expect(calculations.NewRetrieve().
		Where(calculation.MatchOutputs(output)).
		Entry(&c).
		Exec(ctx, n.DB)).To(Succeed())
	return c
}

// openUpgraded runs create on a node, then returns a node over the same storage, so
// that each key create allocates counts as issued when the Core opens, as on a store
// that an older Core wrote.
func openUpgraded(ctx context.Context, create func(n testutil.Node)) testutil.Node {
	GinkgoHelper()
	store := DeferClose(storagetestutil.OpenLayer(GinkgoT().Context()))
	first := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store.Layer})
	create(first)
	Expect(first.Close()).To(Succeed())
	return testutil.NewNode(ctx, distribution.LayerConfig{Storage: store.Layer})
}

var _ = Describe("Calculation records upgrade", func() {
	It("Should build a record for each calculated channel of a v0.62 store", func(
		ctx SpecContext,
	) {
		var plain, doubled, windowed channelv1.Channel
		n := openUpgraded(ctx, func(n testutil.Node) {
			createV062Store(ctx, n, &plain, &doubled, &windowed)
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		var records []calculation.Calculation
		Expect(l.Calculation.NewRetrieve().Entries(&records).Exec(ctx, n.DB)).
			To(Succeed())
		byOutput := make(map[channel.Key]calculation.Calculation, len(records))
		for _, c := range records {
			byOutput[c.Output] = c
		}
		Expect(byOutput).To(HaveKey(doubled.Key()))
		Expect(byOutput).To(HaveKey(windowed.Key()))
		Expect(byOutput).ToNot(HaveKey(plain.Key()))
		Expect(byOutput[doubled.Key()].Expression).To(Equal("return 2"))
		Expect(byOutput[doubled.Key()].Operations).To(BeEmpty())
		Expect(byOutput[windowed.Key()].Operations).To(Equal([]calculation.Operation{
			{Variant: calculation.AvgOperation{
				Window: calculation.Window{Duration: telem.Second},
			}},
			{Variant: calculation.MaxOperation{
				Window: calculation.Window{Reset: new(plain.Key())},
			}},
		}))
		Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
			Where(gorp.MatchKeys[status.Key, status.Status[any]](v062StatusKey)).
			Exists(ctx, n.DB)).To(BeFalse())
	})

	It("Should give a calculated channel the same calculation key on each store", func(
		ctx SpecContext,
	) {
		keyOn := func() (channel.Key, calculation.Key) {
			var plain, doubled, windowed channelv1.Channel
			n := openUpgraded(ctx, func(n testutil.Node) {
				createV062Store(ctx, n, &plain, &doubled, &windowed)
			})
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			c := calculationOf(ctx, n, l.Calculation, doubled.Key())
			return doubled.Key(), c.Key
		}
		firstOutput, first := keyOn()
		secondOutput, second := keyOn()
		Expect(secondOutput).To(Equal(firstOutput))
		Expect(second).To(Equal(first))
	})

	It("Should keep a warning for an operation it drops for no type", func(
		ctx SpecContext,
	) {
		var untyped channelv1.Channel
		n := openUpgraded(ctx, func(n testutil.Node) {
			keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{Name: "untyped", Leaseholder: node.KeyFree, Virtual: true},
			}))
			untyped = channelv1.Channel{
				Name:        keys[0].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[0].LocalKey,
				DataType:    telem.Float32T,
				Virtual:     true,
				Expression:  "return 3",
				Operations: []channelv1.Operation{
					{Duration: telem.Second},
					{Type: channelv1.OperationTypeMax},
				},
			}
			Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
				Set(ctx, untyped)).To(Succeed())
		})

		l := MustOpen(svctestutil.OpenLayer(ctx, n))

		c := calculationOf(ctx, n, l.Calculation, untyped.Key())
		Expect(c.Operations).To(Equal([]calculation.Operation{
			{Variant: calculation.MaxOperation{}},
		}))
		var statuses []status.Status[any]
		Expect(l.Status.NewRetrieve[any]().Entries(&statuses).Exec(ctx, n.DB)).
			To(Succeed())
		warnings := lo.Filter(statuses, func(s status.Status[any], _ int) bool {
			return s.Message == "Dropped operations with no type"
		})
		Expect(warnings).To(HaveLen(1))
		warning := warnings[0]
		Expect(warning.Owner).To(BeNil())
		Expect(warning.Variant).To(Equal(status.VariantWarning))
		Expect(warning.Name).To(Equal("untyped"))
		Expect(warning.Description).To(Equal(
			"The upgrade dropped the stored operations of untyped that had no type. " +
				"Add them again to keep them.",
		))
	})
})

// v062StatusKey is the key of the status that createV062Store gives the doubled
// channel.
const v062StatusKey = "calculation-status"

// createV062Store writes the channel and status records that v0.62 stores for a plain
// channel, a calculated channel with a status, and a calculated channel with
// operations.
func createV062Store(
	ctx context.Context,
	n testutil.Node,
	plain, doubled, windowed *channelv1.Channel,
) {
	GinkgoHelper()
	// v0.62 stores channel v1 records, under keys that its counter issued.
	keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
		{Name: "plain", Leaseholder: node.KeyFree, Virtual: true},
		{Name: "doubled", Leaseholder: node.KeyFree, Virtual: true},
		{Name: "windowed", Leaseholder: node.KeyFree, Virtual: true},
	}))
	virtual := func(i int) channelv1.Channel {
		return channelv1.Channel{
			Name:        keys[i].Name,
			Leaseholder: node.KeyFree,
			LocalKey:    keys[i].LocalKey,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
	}
	*plain = virtual(0)
	*doubled = virtual(1)
	doubled.Expression = "return 2"
	*windowed = virtual(2)
	windowed.Expression = "return 3"
	windowed.Operations = []channelv1.Operation{
		{Type: channelv1.OperationTypeAvg, Duration: telem.Second},
		{Type: channelv1.OperationTypeMax, ResetChannel: plain.Key()},
	}
	Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
		Set(ctx, *plain, *doubled, *windowed)).To(Succeed())
	// v0.62 stores status v2 records, and its status chain has run, so opening the
	// status service once applies the chain before the record is written.
	Expect(MustSucceed(status.OpenService(ctx, status.ServiceConfig{DB: n.DB})).
		Close()).To(Succeed())
	owned := status.Status[any]{
		Key:     v062StatusKey,
		Name:    doubled.Name,
		Variant: status.VariantError,
		Message: "Failed to compile calculation",
		Owner:   new(channel.ResourceID(doubled.Key())),
	}
	Expect(gorp.WrapWriter[status.Key, status.Status[any]](n.DB).Set(ctx, owned)).
		To(Succeed())
}
