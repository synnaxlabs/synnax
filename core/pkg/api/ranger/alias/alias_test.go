// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias_test

import (
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apialias "github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createRange(ctx SpecContext) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Ranger.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	return r
}

func createChannel(ctx SpecContext) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     "aliased_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
		DataType: telem.Float64T,
		Virtual:  true,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Channel.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	return ch
}

// retrieveAlias returns the alias of ch on r, or a query.ErrNotFound error.
func retrieveAlias(
	ctx SpecContext,
	r ranger.Range,
	ch channel.Channel,
) (string, error) {
	return svc.Alias.NewReader(nil).Retrieve(ctx, r.Key, ch.Key())
}

var _ = Describe("Set", func() {
	var (
		u  user.User
		r  ranger.Range
		ch channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
		ch = createChannel(ctx)
	})
	setAs := func(ctx SpecContext, name string) (struct{}, error) {
		return set(AuthedCtx(ctx, u), apialias.SetRequest{
			Range:   r.Key,
			Aliases: map[channel.Key]string{ch.Key(): name},
		})
	}
	aliasID := func() resource.ID { return alias.ResourceID(r.Key, ch.Key()) }
	rangeID := func() resource.ID { return r.ResourceID() }
	DescribeTable(
		"Should refuse the set and write no alias",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grant(ctx, u, granted, grantedOn())
			Expect(setAs(ctx, "alias")).Error().To(denied(u, refused, refusedOn()))
			Expect(retrieveAlias(ctx, r, ch)).Error().To(MatchError(query.ErrNotFound))
		},
		Entry(
			"without create on the alias",
			access.ActionUpdate,
			rangeID,
			access.ActionCreate,
			aliasID,
		),
		Entry(
			"without update on the range",
			access.ActionCreate,
			aliasID,
			access.ActionUpdate,
			rangeID,
		),
	)
	It(
		"Should set the alias with create on it and update on the range",
		func(ctx SpecContext) {
			grant(ctx, u, access.ActionCreate, aliasID())
			grant(ctx, u, access.ActionUpdate, rangeID())
			MustSucceed(setAs(ctx, "alias"))
			Expect(retrieveAlias(ctx, r, ch)).To(Equal("alias"))
		},
	)
	It("Should refuse to change an alias without update on it", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Alias.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "stored")
		})).
			To(Succeed())
		grant(ctx, u, access.ActionCreate, aliasID())
		Expect(setAs(ctx, "changed")).Error().
			To(denied(u, access.ActionUpdate, aliasID()))
		Expect(retrieveAlias(ctx, r, ch)).To(Equal("stored"))
	})
})

var _ = Describe("Delete", func() {
	var (
		u  user.User
		r  ranger.Range
		ch channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
		ch = createChannel(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Alias.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "alias")
		})).
			To(Succeed())
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apialias.DeleteRequest{
			Range:    r.Key,
			Channels: []channel.Key{ch.Key()},
		})
	}
	aliasID := func() resource.ID { return alias.ResourceID(r.Key, ch.Key()) }
	rangeID := func() resource.ID { return r.ResourceID() }
	DescribeTable(
		"Should refuse the delete and keep the alias",
		func(
			ctx SpecContext,
			granted access.Action,
			grantedOn func() resource.ID,
			refused access.Action,
			refusedOn func() resource.ID,
		) {
			grant(ctx, u, granted, grantedOn())
			Expect(deleteAs(ctx)).Error().To(denied(u, refused, refusedOn()))
			Expect(retrieveAlias(ctx, r, ch)).To(Equal("alias"))
		},
		Entry(
			"without delete on the alias",
			access.ActionUpdate,
			rangeID,
			access.ActionDelete,
			aliasID,
		),
		Entry(
			"without update on the range",
			access.ActionDelete,
			aliasID,
			access.ActionUpdate,
			rangeID,
		),
	)
	It(
		"Should delete the alias with delete on it and update on the range",
		func(ctx SpecContext) {
			grant(ctx, u, access.ActionDelete, aliasID())
			grant(ctx, u, access.ActionUpdate, rangeID())
			MustSucceed(deleteAs(ctx))
			Expect(retrieveAlias(ctx, r, ch)).Error().To(MatchError(query.ErrNotFound))
		},
	)
})
