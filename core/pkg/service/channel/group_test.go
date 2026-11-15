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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Group", Ordered, func() {
	var (
		db       *gorp.DB
		groupSvc *group.Service
		svc      *channel.Service
		g        group.Group
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, gs := serviceConfig(ctx, node, node.DB)
		groupSvc = gs
		svc = MustOpen(channel.OpenService(ctx, cfg))
		g = MustSucceed(groupSvc.CreateOrRetrieve(ctx, "Sensors", resource.TypeChannel))
	})
	create := func(ctx context.Context, grp *group.Key) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
			Group:    grp,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	retrieve := func(ctx context.Context, key channel.Key) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}

	It("Should create a channel in a group", func(ctx SpecContext) {
		ch := create(ctx, &g.Key)
		Expect(retrieve(ctx, ch.Key()).Group).To(Equal(&g.Key))
	})

	createCalculated := func(
		ctx context.Context,
		name string,
		grp *group.Key,
	) channel.Channel {
		GinkgoHelper()
		idx := create(ctx, nil)
		data := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			LocalIndex: idx.LocalKey,
		}
		calc := channel.Channel{
			Name:       name,
			Expression: "return " + data.Name,
			Group:      grp,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Create(ctx, &data); err != nil {
				return err
			}
			return w.Create(ctx, &calc)
		})).To(Succeed())
		return retrieve(ctx, calc.Key())
	}

	It(
		"Should create the index of a calculated channel in its group",
		func(ctx SpecContext) {
			calc := createCalculated(ctx, UniqueChannelName(), &g.Key)
			calcIdx := retrieve(ctx, calc.Index())
			Expect(calcIdx.IsIndex).To(BeTrue())
			Expect(calcIdx.Group).To(Equal(&g.Key))
		},
	)

	It(
		"Should move an adopted index into the group of its calculated channel",
		func(ctx SpecContext) {
			name := UniqueChannelName()
			orphan := channel.Channel{
				Name:        name + "_time",
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Virtual:     true,
				Leaseholder: node.KeyFree,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &orphan)
			})).To(Succeed())
			calc := createCalculated(ctx, name, &g.Key)
			Expect(calc.Index()).To(Equal(orphan.Key()))
			Expect(retrieve(ctx, calc.Index()).Group).To(Equal(&g.Key))
		},
	)

	It(
		"Should delete a group once every calculated channel in it is deleted",
		func(ctx SpecContext) {
			grp := group.Group{Name: "Calculations", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &grp)
			})).To(Succeed())
			a := createCalculated(ctx, UniqueChannelName(), &grp.Key)
			b := createCalculated(ctx, UniqueChannelName(), &grp.Key)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					DeleteMany(ctx, channel.Keys{a.Key(), b.Key()}, false)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(a.Index(), b.Index())).
				Exists(ctx, db)).To(BeFalse())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Delete(ctx, grp.Key)
			})).To(Succeed())
		},
	)

	It(
		"Should delete the index of a calculated channel deleted by name",
		func(ctx SpecContext) {
			calc := createCalculated(ctx, UniqueChannelName(), &g.Key)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					DeleteManyByNames(ctx, []string{calc.Name}, false)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(calc.Key(), calc.Index())).
				Exists(ctx, db)).To(BeFalse())
		},
	)

	shareIndex := func(ctx context.Context, calc channel.Channel) channel.Channel {
		GinkgoHelper()
		data := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float32T,
			Virtual:     true,
			Leaseholder: node.KeyFree,
			LocalIndex:  calc.LocalIndex,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &data)
		})).To(Succeed())
		Expect(data.Index()).To(Equal(calc.Index()))
		return data
	}

	DescribeTable(
		"Should keep the index of a deleted calculated channel that another uses",
		func(
			ctx SpecContext,
			del func(context.Context, channel.Writer, channel.Channel) error,
		) {
			calc := createCalculated(ctx, UniqueChannelName(), nil)
			shareIndex(ctx, calc)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return del(ctx, svc.NewWriter(tx), calc)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(calc.Index())).
				Exists(ctx, db)).To(BeTrue())
		},
		Entry(
			"by key",
			func(ctx context.Context, w channel.Writer, c channel.Channel) error {
				return w.Delete(ctx, c.Key(), false)
			},
		),
		Entry(
			"by name",
			func(ctx context.Context, w channel.Writer, c channel.Channel) error {
				return w.DeleteManyByNames(ctx, []string{c.Name}, false)
			},
		),
	)

	It(
		"Should leave a shared index in place when its calculated channel is created again",
		func(ctx SpecContext) {
			name := UniqueChannelName()
			old := createCalculated(ctx, name, nil)
			shareIndex(ctx, old)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, old.Key(), false)
			})).To(Succeed())
			calc := createCalculated(ctx, name, &g.Key)
			Expect(calc.Index()).To(Equal(old.Index()))
			Expect(retrieve(ctx, calc.Index()).Group).To(BeNil())
		},
	)

	It(
		"Should leave a shared index in place when its calculated channel moves",
		func(ctx SpecContext) {
			calc := createCalculated(ctx, UniqueChannelName(), nil)
			shareIndex(ctx, calc)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, calc.Key())
			})).To(Succeed())
			Expect(retrieve(ctx, calc.Key()).Group).To(Equal(&g.Key))
			Expect(retrieve(ctx, calc.Index()).Group).To(BeNil())
		},
	)

	It("Should reject a group that does not exist", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
			Group:    new(uuid.New()),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(MatchError(ContainSubstring("does not exist")))
	})

	Describe("SetGroup", func() {
		It("Should move channels into a group and back out", func(ctx SpecContext) {
			a, b := create(ctx, nil), create(ctx, nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, a.Key(), b.Key())
			})).To(Succeed())
			Expect(retrieve(ctx, a.Key()).Group).To(Equal(&g.Key))
			Expect(retrieve(ctx, b.Key()).Group).To(Equal(&g.Key))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, nil, a.Key())
			})).To(Succeed())
			Expect(retrieve(ctx, a.Key()).Group).To(BeNil())
		})

		It(
			"Should move the index of a calculated channel with it",
			func(ctx SpecContext) {
				calc := createCalculated(ctx, UniqueChannelName(), nil)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).SetGroup(ctx, &g.Key, calc.Key())
				})).To(Succeed())
				Expect(retrieve(ctx, calc.Index()).Group).To(Equal(&g.Key))
			},
		)

		It(
			"Should move a calculated channel that has no index",
			func(ctx SpecContext) {
				calc := createCalculated(ctx, UniqueChannelName(), nil)
				calc.LocalIndex = 0
				Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
					Set(ctx, calc)).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).SetGroup(ctx, &g.Key, calc.Key())
				})).To(Succeed())
				Expect(retrieve(ctx, calc.Key()).Group).To(Equal(&g.Key))
			},
		)
	})

	It(
		"Should keep the stored group when a channel is created again",
		func(ctx SpecContext) {
			ch := create(ctx, &g.Key)
			again := channel.Channel{
				Name:     ch.Name,
				DataType: ch.DataType,
				IsIndex:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					Create(ctx, &again, channel.RetrieveIfNameExists())
			})).To(Succeed())
			Expect(retrieve(ctx, ch.Key()).Group).To(Equal(&g.Key))
		},
	)

	It("Should filter channels by group", func(ctx SpecContext) {
		in, out := create(ctx, &g.Key), create(ctx, nil)
		var grouped, ungrouped []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(in.Key(), out.Key())).
			Where(channel.MatchGroups(g.Key)).
			Entries(&grouped).
			Exec(ctx, db)).To(Succeed())
		Expect(channel.KeysFromChannels(grouped)).To(ConsistOf(in.Key()))
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(in.Key(), out.Key())).
			Where(channel.MatchHasGroup(false)).
			Entries(&ungrouped).
			Exec(ctx, db)).To(Succeed())
		Expect(channel.KeysFromChannels(ungrouped)).To(ConsistOf(out.Key()))
	})

	It("Should refuse to delete a group that holds a channel", func(ctx SpecContext) {
		full := MustSucceed(
			groupSvc.CreateOrRetrieve(ctx, "Full", resource.TypeChannel),
		)
		create(ctx, &full.Key)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Delete(ctx, full.Key)
		})).To(MatchError(
			ContainSubstring("cannot delete group Full because it is not empty"),
		))
	})
})
