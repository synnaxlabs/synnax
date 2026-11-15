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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// change makes a change to a readable and a denied record in one commit.
type change func(ctx SpecContext)

var _ = Describe("Change channels", func() {
	var (
		ch     channel.Channel
		client freighter.ClientStream[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		]
	)
	// find sets ch to the change channel with the given name.
	find := func(ctx SpecContext, name string) {
		GinkgoHelper()
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
	}
	// newUser creates a user that can retrieve ch and the granted records.
	newUser := func(ctx SpecContext, granted ...resource.ID) user.User {
		GinkgoHelper()
		return createUserGranted(
			ctx,
			access.ActionRetrieve,
			append(svcframer.ResourceIDs(channel.Keys{ch.Key()}), granted...)...,
		)
	}
	// open opens a stream of ch as u.
	open := func(ctx SpecContext, u user.User) {
		GinkgoHelper()
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
	}
	// stream opens a stream of the change channel with the given name as a new user
	// that can retrieve the channel and the granted records.
	stream := func(ctx SpecContext, name string, granted ...resource.ID) user.User {
		GinkgoHelper()
		find(ctx, name)
		u := newUser(ctx, granted...)
		open(ctx, u)
		return u
	}
	// received returns the record of each sample in the next frame.
	received := func() []resource.ID {
		GinkgoHelper()
		res := MustSucceed(client.Receive())
		recordOf := svc.Signals.Records()[ch.Key()].Of
		var records []resource.ID
		for _, series := range res.Frame.Get(ch.Key()).Series {
			for sample := range series.Samples() {
				records = append(records, MustSucceed(recordOf(sample)))
			}
		}
		return records
	}
	createLabels := func(ctx SpecContext, labels ...label.Label) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Label.NewWriter(tx).CreateMany(ctx, &labels)
		})).To(Succeed())
	}
	newLabel := func() label.Label {
		return label.Label{Key: uuid.New(), Name: "change-channel"}
	}
	newRange := func(ctx SpecContext) ranger.Range {
		GinkgoHelper()
		r := ranger.Range{
			Key:  uuid.New(),
			Name: "change-channel",
			TimeRange: telem.TimeRange{
				Start: telem.SecondTS,
				End:   telem.SecondTS * 2,
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		return r
	}
	pairs := func(ranges ...ranger.Range) []kv.Pair {
		out := make([]kv.Pair, len(ranges))
		for i, r := range ranges {
			out[i] = kv.Pair{Range: r.Key, Key: "change-channel", Value: "value"}
		}
		return out
	}

	DescribeTable(
		"Should send a change only to a subject that can retrieve its record",
		func(
			ctx SpecContext,
			name string,
			prepare func(SpecContext) (resource.ID, change),
		) {
			readable, makeChange := prepare(ctx)
			stream(ctx, name, readable)
			makeChange(ctx)
			Expect(received()).To(Equal([]resource.ID{readable}))
		},
		Entry(
			"a label set",
			"sy_label_set",
			func(SpecContext) (resource.ID, change) {
				GinkgoHelper()
				readable, denied := newLabel(), newLabel()
				return label.ResourceID(readable.Key), func(ctx SpecContext) {
					GinkgoHelper()
					createLabels(ctx, readable, denied)
				}
			},
		),
		Entry(
			"a label action",
			"sy_label_action_set",
			func(SpecContext) (resource.ID, change) {
				GinkgoHelper()
				readable, denied := newLabel(), newLabel()
				return label.ResourceID(readable.Key), func(ctx SpecContext) {
					GinkgoHelper()
					createLabels(ctx, readable, denied)
				}
			},
		),
		Entry(
			"a label delete",
			"sy_label_delete",
			func(ctx SpecContext) (resource.ID, change) {
				GinkgoHelper()
				readable, denied := newLabel(), newLabel()
				createLabels(ctx, readable, denied)
				return label.ResourceID(readable.Key), func(ctx SpecContext) {
					GinkgoHelper()
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.Label.NewWriter(tx).Delete(
							ctx,
							readable.Key,
							denied.Key,
						)
					})).To(Succeed())
				}
			},
		),
		Entry(
			"a range KV set",
			"sy_range_kv_set",
			func(ctx SpecContext) (resource.ID, change) {
				GinkgoHelper()
				readable, denied := newRange(ctx), newRange(ctx)
				return ranger.ResourceID(readable.Key), func(ctx SpecContext) {
					GinkgoHelper()
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.KV.NewWriter(tx).SetMany(
							ctx,
							pairs(readable, denied),
						)
					})).To(Succeed())
				}
			},
		),
		Entry(
			"a range KV delete",
			"sy_range_kv_delete",
			func(ctx SpecContext) (resource.ID, change) {
				GinkgoHelper()
				readable, denied := newRange(ctx), newRange(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.KV.NewWriter(tx).SetMany(
						ctx,
						pairs(readable, denied),
					)
				})).To(Succeed())
				return ranger.ResourceID(readable.Key), func(ctx SpecContext) {
					GinkgoHelper()
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						w := svc.KV.NewWriter(tx)
						for _, p := range pairs(readable, denied) {
							if err := w.Delete(ctx, p.Range, p.Key); err != nil {
								return err
							}
						}
						return nil
					})).To(Succeed())
				}
			},
		),
	)

	It("Should send a subject the changes to the policies of its role", func(
		ctx SpecContext,
	) {
		other := role.Role{Name: "change-channel"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Role.NewWriter(tx, false).Create(ctx, &other)
		})).To(Succeed())
		u := stream(ctx, "sy_policy_set")
		own := policy.Policy{
			Role:    u.Role,
			Name:    "change-channel",
			Objects: []resource.ID{{Type: resource.TypeLabel}},
			Actions: []access.Action{access.ActionUpdate},
		}
		denied := own
		denied.Role = other.Key
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := rbacSvc.Policy.NewWriter(tx, false)
			if err := w.Create(ctx, &own); err != nil {
				return err
			}
			return w.Create(ctx, &denied)
		})).To(Succeed())
		// The policy that stream grants the user can reach the stream after it opens.
		var records []resource.ID
		Eventually(func() []resource.ID {
			records = received()
			return records
		}).WithTimeout(5 * time.Second).Should(ContainElement(own.ResourceID()))
		Expect(records).ToNot(ContainElement(denied.ResourceID()))
	}, SpecTimeout(10*time.Second))

	It("Should send a subject the deletes of the policies of its role", func(
		ctx SpecContext,
	) {
		find(ctx, "sy_policy_delete")
		u := newUser(ctx)
		other := role.Role{Name: "change-channel"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Role.NewWriter(tx, false).Create(ctx, &other)
		})).To(Succeed())
		own := policy.Policy{
			Role:    u.Role,
			Name:    "change-channel",
			Objects: []resource.ID{{Type: resource.TypeLabel}},
			Actions: []access.Action{access.ActionUpdate},
		}
		denied := own
		denied.Role = other.Key
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := rbacSvc.Policy.NewWriter(tx, false)
			if err := w.Create(ctx, &own); err != nil {
				return err
			}
			return w.Create(ctx, &denied)
		})).To(Succeed())
		open(ctx, u)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Policy.NewWriter(tx, false).Delete(ctx, own.Key, denied.Key)
		})).To(Succeed())
		Expect(received()).To(Equal([]resource.ID{own.ResourceID()}))
	}, SpecTimeout(10*time.Second))

	It("Should send the delete of a policy that a role held at an earlier check", func(
		ctx SpecContext,
	) {
		find(ctx, "sy_policy_delete")
		u := newUser(ctx)
		other := role.Role{Name: "change-channel"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Role.NewWriter(tx, false).Create(ctx, &other)
		})).To(Succeed())
		held := policy.Policy{
			Role:    u.Role,
			Name:    "change-channel",
			Objects: []resource.ID{{Type: resource.TypeLabel}},
			Actions: []access.Action{access.ActionUpdate},
		}
		unrelated := held
		unrelated.Role = other.Key
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := rbacSvc.Policy.NewWriter(tx, false)
			if err := w.Create(ctx, &held); err != nil {
				return err
			}
			return w.Create(ctx, &unrelated)
		})).To(Succeed())
		open(ctx, u)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return userSvc.NewWriter(tx).SetRole(ctx, other.Key, u.Key)
		})).To(Succeed())
		// The check of this delete no longer finds held among the policies of u.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Policy.NewWriter(tx, false).Delete(ctx, unrelated.Key)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Policy.NewWriter(tx, false).Delete(ctx, held.Key)
		})).To(Succeed())
		Expect(received()).To(Equal([]resource.ID{held.ResourceID()}))
	}, SpecTimeout(10*time.Second))

	It("Should stop sending a record once a policy no longer allows it", func(
		ctx SpecContext,
	) {
		u := stream(ctx, "sy_label_set")
		p := MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			userSvc,
			u.Key,
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeLabel},
		))
		first := newLabel()
		createLabels(ctx, first)
		Expect(received()).To(Equal([]resource.ID{label.ResourceID(first.Key)}))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rbacSvc.Policy.NewWriter(tx, false).Delete(ctx, p.Key)
		})).To(Succeed())
		removed, granted := newLabel(), newLabel()
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			userSvc,
			u.Key,
			[]access.Action{access.ActionRetrieve},
			label.ResourceID(granted.Key),
		))
		createLabels(ctx, removed)
		createLabels(ctx, granted)
		Expect(received()).To(Equal([]resource.ID{label.ResourceID(granted.Key)}))
	})
})
