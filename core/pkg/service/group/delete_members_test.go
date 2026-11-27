// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"context"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("DeleteMembers", func() {
	var (
		db      *gorp.DB
		svc     *group.Service
		notes   *gorp.Table[string, note]
		calls   [][]group.Key
		hookErr error
		// kept makes the hook delete no member.
		kept bool
	)

	BeforeEach(func(ctx SpecContext) {
		calls, hookErr, kept = nil, nil, false
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: db,
			DeleteMembers: map[resource.Type]group.DeleteMembers{
				resource.TypeChannel: func(
					ctx context.Context,
					tx gorp.Tx,
					groups []group.Key,
				) error {
					calls = append(calls, groups)
					if hookErr != nil || kept {
						return hookErr
					}
					return notes.NewDelete().
						Where(gorp.Match(func(_ gorp.Context, n *note) (bool, error) {
							return slices.Contains(groups, n.Group), nil
						})).
						Exec(ctx, tx)
				},
			},
		}))
		notes = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, note]{
			DB:         db,
			Name:       "note",
			References: []gorp.Reference[string, note]{noteReference()},
		}))
	})

	create := func(ctx context.Context, name string, scope resource.Type) group.Group {
		GinkgoHelper()
		g := group.Group{Name: name, Scope: scope}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		Expect(notes.NewCreate().
			Entry(&note{Key: name, Group: g.Key}).
			Exec(ctx, db)).To(Succeed())
		return g
	}

	exists := func(ctx context.Context, key group.Key) bool {
		GinkgoHelper()
		return MustSucceed(svc.NewRetrieve().
			Where(group.MatchKeys(key)).
			Exists(ctx, db))
	}

	It("Should run the hook of a group's scope before the delete", func(
		ctx SpecContext,
	) {
		g := create(ctx, "channels", resource.TypeChannel)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, g.Key)
		})).To(Succeed())
		Expect(calls).To(Equal([][]group.Key{{g.Key}}))
		Expect(exists(ctx, g.Key)).To(BeFalse())
		Expect(notes.NewRetrieve().
			Where(gorp.MatchKeys[string, note]("channels")).
			Exists(ctx, db)).To(BeFalse())
	})

	It("Should refuse a group of a scope with no hook that holds a member", func(
		ctx SpecContext,
	) {
		channels := create(ctx, "channels", resource.TypeChannel)
		devices := create(ctx, "devices", resource.TypeDevice)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, channels.Key, devices.Key)
		})).To(MatchError(ContainSubstring(
			"cannot delete group devices because it is not empty",
		)))
		Expect(calls).To(Equal([][]group.Key{{channels.Key}}))
		Expect(exists(ctx, channels.Key)).To(BeTrue())
		Expect(exists(ctx, devices.Key)).To(BeTrue())
	})

	It("Should return the error of the hook and delete nothing", func(
		ctx SpecContext,
	) {
		g := create(ctx, "channels", resource.TypeChannel)
		hookErr = errors.New("hook failed")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, g.Key)
		})).To(MatchError(hookErr))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should skip the hook for a key with no group", func(ctx SpecContext) {
		key := uuid.New()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, key)
		})).To(Succeed())
		Expect(calls).To(BeEmpty())
		Expect(svc.NewRetrieve().
			Where(group.MatchKeys(key)).
			Entry(new(group.Group)).
			Exec(ctx, db)).To(MatchError(query.ErrNotFound))
	})

	It("Should refuse the delete when the hook leaves a member", func(
		ctx SpecContext,
	) {
		g := create(ctx, "channels", resource.TypeChannel)
		kept = true
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, g.Key)
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"cannot delete group channels because it is not empty",
			)),
		))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})
})
