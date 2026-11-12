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
	"github.com/synnaxlabs/x/validate"
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
	return svc.Alias.NewReader(db).Retrieve(ctx, r.Key, ch.Key())
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
	It("Should refuse a request that gives one alias to two channels", func(
		ctx SpecContext,
	) {
		other := createChannel(ctx)
		grant(
			ctx,
			u,
			access.ActionCreate,
			aliasID(),
			alias.ResourceID(r.Key, other.Key()),
		)
		grant(ctx, u, access.ActionUpdate, rangeID())
		Expect(set(AuthedCtx(ctx, u), apialias.SetRequest{
			Range:   r.Key,
			Aliases: map[channel.Key]string{ch.Key(): "x", other.Key(): "x"},
		})).Error().To(MatchError(validate.ErrValidation))
		Expect(retrieveAlias(ctx, r, ch)).Error().To(MatchError(query.ErrNotFound))
		Expect(retrieveAlias(ctx, r, other)).Error().To(MatchError(query.ErrNotFound))
	})
	Context("Alias held by another channel", func() {
		var other channel.Channel
		BeforeEach(func(ctx SpecContext) {
			other = createChannel(ctx)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Alias.NewWriter(tx).Set(ctx, r.Key, other.Key(), "x")
			})).To(Succeed())
			grant(ctx, u, access.ActionCreate, aliasID())
			grant(ctx, u, access.ActionUpdate, rangeID())
		})
		otherID := func() resource.ID { return alias.ResourceID(r.Key, other.Key()) }
		It("Should refuse the move without delete on the old alias", func(
			ctx SpecContext,
		) {
			Expect(setAs(ctx, "x")).Error().
				To(denied(u, access.ActionDelete, otherID()))
			Expect(retrieveAlias(ctx, r, other)).To(Equal("x"))
			Expect(retrieveAlias(ctx, r, ch)).Error().To(MatchError(query.ErrNotFound))
		})
		It("Should move the alias with delete on the old alias", func(ctx SpecContext) {
			grant(ctx, u, access.ActionDelete, otherID())
			MustSucceed(setAs(ctx, "x"))
			Expect(retrieveAlias(ctx, r, ch)).To(Equal("x"))
			Expect(retrieveAlias(ctx, r, other)).Error().
				To(MatchError(query.ErrNotFound))
		})
	})
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

var _ = Describe("Dispatch", func() {
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
			return svc.Alias.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "stored")
		})).To(Succeed())
	})
	dispatchAs := func(ctx SpecContext, name string) (struct{}, error) {
		return dispatch(AuthedCtx(ctx, u), apialias.DispatchRequest{
			Key: alias.Alias{Range: r.Key, Channel: ch.Key()}.GorpKey(),
			Actions: []alias.Action{alias.NewSetAction(
				alias.SetPayload{Path: []string{"alias"}, Value: name},
			)},
		})
	}
	aliasID := func() resource.ID { return alias.ResourceID(r.Key, ch.Key()) }
	It("Should change the alias with update on it", func(ctx SpecContext) {
		grant(ctx, u, access.ActionUpdate, aliasID())
		MustSucceed(dispatchAs(ctx, "changed"))
		Expect(retrieveAlias(ctx, r, ch)).To(Equal("changed"))
	})
	It("Should refuse a change without update on the alias", func(ctx SpecContext) {
		grant(ctx, u, access.ActionUpdate, r.ResourceID())
		Expect(dispatchAs(ctx, "changed")).Error().
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

var _ = Describe("Read", func() {
	var (
		u                user.User
		r                ranger.Range
		allowed, blocked channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		r = createRange(ctx)
		allowed, blocked = createChannel(ctx), createChannel(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.Alias.NewWriter(tx)
			if err := w.Set(ctx, r.Key, allowed.Key(), "allowed"); err != nil {
				return err
			}
			return w.Set(ctx, r.Key, blocked.Key(), "blocked")
		})).To(Succeed())
		grant(ctx, u, access.ActionRetrieve, alias.ResourceID(r.Key, allowed.Key()))
	})
	It(
		"Should resolve only the aliases the subject can retrieve",
		func(ctx SpecContext) {
			Expect(resolve(AuthedCtx(ctx, u), apialias.ResolveRequest{
				Range:   r.Key,
				Aliases: []string{"allowed", "blocked", "absent"},
			})).To(Equal(apialias.ResolveResponse{
				Aliases: map[string]channel.Key{"allowed": allowed.Key()},
			}))
		},
	)
	Context("Alias that two channels hold", func() {
		BeforeEach(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Alias.NewWriter(tx).Apply(
					ctx,
					alias.Alias{Range: r.Key, Channel: blocked.Key()}.GorpKey(),
					alias.NewSetAction(
						alias.SetPayload{Path: []string{"alias"}, Value: "allowed"},
					),
				)
			})).To(Succeed())
		})
		resolveAllowed := func(ctx SpecContext) (apialias.ResolveResponse, error) {
			return resolve(AuthedCtx(ctx, u), apialias.ResolveRequest{
				Range:   r.Key,
				Aliases: []string{"allowed"},
			})
		}
		It("Should refuse to resolve it when the subject can retrieve both", func(
			ctx SpecContext,
		) {
			grant(
				ctx,
				u,
				access.ActionRetrieve,
				alias.ResourceID(r.Key, blocked.Key()),
			)
			Expect(resolveAllowed(ctx)).Error().To(MatchError(alias.ErrAmbiguous))
		})
		It("Should leave it out when the subject can retrieve only one", func(
			ctx SpecContext,
		) {
			Expect(resolveAllowed(ctx)).To(Equal(apialias.ResolveResponse{
				Aliases: map[string]channel.Key{},
			}))
		})
	})
	It("Should list only the aliases the subject can retrieve", func(ctx SpecContext) {
		Expect(list(AuthedCtx(ctx, u), apialias.ListRequest{Range: r.Key})).
			To(Equal(apialias.ListResponse{
				Aliases: map[channel.Key]string{allowed.Key(): "allowed"},
			}))
	})
	It("Should retrieve the named aliases the subject can retrieve", func(
		ctx SpecContext,
	) {
		Expect(retrieve(AuthedCtx(ctx, u), apialias.RetrieveRequest{
			Range:    r.Key,
			Channels: []channel.Key{allowed.Key()},
		})).To(Equal(apialias.RetrieveResponse{
			Aliases: map[channel.Key]string{allowed.Key(): "allowed"},
		}))
	})
	It("Should deny a named alias the subject cannot retrieve", func(ctx SpecContext) {
		Expect(retrieve(AuthedCtx(ctx, u), apialias.RetrieveRequest{
			Range:    r.Key,
			Channels: []channel.Key{allowed.Key(), blocked.Key()},
		})).Error().To(denied(
			u,
			access.ActionRetrieve,
			alias.ResourceID(r.Key, blocked.Key()),
		))
	})
})
