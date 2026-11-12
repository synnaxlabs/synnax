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
	"context"
	"fmt"
	"math/rand/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Alias", Ordered, func() {
	var (
		db         *gorp.DB
		rangerSvc  *ranger.Service
		aliasSvc   *alias.Service
		channelSvc *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB: node.DB,
		}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: node.DB,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: node.DB,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:    node.DB,
			Group: groupSvc,
		}))
		aliasSvc = MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
			DB:              node.DB,
			Channel:         channelSvc,
			ParentRetriever: rangerSvc,
		}))
	})
	channelCount := 0
	createChannel := func(ctx context.Context) channel.Channel {
		GinkgoHelper()
		channelCount++
		ch := channel.Channel{
			DataType: telem.Float32T,
			Name:     fmt.Sprintf("test_%d", channelCount),
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createRange := func(ctx context.Context, tx gorp.Tx, parent *ranger.Key) ranger.Key {
		GinkgoHelper()
		r := ranger.Range{
			Name:      "Range",
			TimeRange: telem.NewRangeSeconds(5, 10),
			Parent:    parent,
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		return r.Key
	}
	// createChain creates a root range and n nested descendants, and returns the root
	// and the deepest range.
	createChain := func(ctx context.Context, tx gorp.Tx, n int) (ranger.Key, ranger.Key) {
		GinkgoHelper()
		root := createRange(ctx, tx, nil)
		leaf := root
		for range n {
			leaf = createRange(ctx, tx, &leaf)
		}
		return root, leaf
	}

	Describe("Set", func() {
		It("Should set an alias for a channel on a range", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
		})

		It("Should move an alias that another channel holds on the range", func(
			ctx SpecContext,
		) {
			a, b := createChannel(ctx), createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := createRange(ctx, tx, nil)
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, r, a.Key(), "x")).To(Succeed())
			Expect(w.Set(ctx, r, b.Key(), "x")).To(Succeed())
			reader := aliasSvc.NewReader(tx)
			Expect(reader.Resolve(ctx, r, "x")).To(Equal(b.Key()))
			Expect(reader.Retrieve(ctx, r, a.Key())).Error().
				To(MatchError(query.ErrNotFound))
		})

		It("Should keep an alias that a channel holds on another range", func(
			ctx SpecContext,
		) {
			a, b := createChannel(ctx), createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r1, r2 := createRange(ctx, tx, nil), createRange(ctx, tx, nil)
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, r1, a.Key(), "x")).To(Succeed())
			Expect(w.Set(ctx, r2, b.Key(), "x")).To(Succeed())
			Expect(aliasSvc.NewReader(tx).Resolve(ctx, r1, "x")).To(Equal(a.Key()))
		})
	})

	Describe("Dispatch", func() {
		type frame = actions.Scoped[string, alias.Action]
		var (
			r   ranger.Range
			ch  channel.Channel
			key string
		)
		write := func(ctx context.Context, fn func(alias.Writer) error) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				return fn(aliasSvc.NewWriter(tx).WithDispatchKey("dk"))
			})
		}
		BeforeEach(func(ctx SpecContext) {
			ch = createChannel(ctx)
			r = ranger.Range{Name: "Range", TimeRange: telem.NewRangeSeconds(5, 10)}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rangerSvc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			Expect(write(ctx, func(w alias.Writer) error {
				return w.Set(ctx, r.Key, ch.Key(), "Before")
			})).To(Succeed())
			key = alias.Alias{Range: r.Key, Channel: ch.Key()}.GorpKey()
		})
		set := func(path string, value any) alias.Action {
			return alias.NewSetAction(
				alias.SetPayload{Path: []string{path}, Value: value},
			)
		}
		apply := func(ctx context.Context, key string, acts ...alias.Action) error {
			return write(ctx, func(w alias.Writer) error {
				return w.Apply(ctx, key, acts...)
			})
		}
		retrieve := func(ctx context.Context) string {
			GinkgoHelper()
			return MustSucceed(aliasSvc.NewReader(db).Retrieve(ctx, r.Key, ch.Key()))
		}

		It("Should store the edited alias", func(ctx SpecContext) {
			Expect(apply(ctx, key, set("alias", "After"))).To(Succeed())
			Expect(retrieve(ctx)).To(Equal("After"))
		})

		It("Should refuse a path outside the editable fields", func(ctx SpecContext) {
			err := apply(ctx, key, set("channel", 1))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).
				To(MatchError(ContainSubstring(`unknown object member name "channel"`)))
			Expect(retrieve(ctx)).To(Equal("Before"))
		})

		It("Should keep the alias of another channel through each step of an edit",
			func(ctx SpecContext) {
				other := createChannel(ctx)
				Expect(write(ctx, func(w alias.Writer) error {
					return w.Set(ctx, r.Key, other.Key(), "temp")
				})).To(Succeed())
				for _, text := range []string{"", "t", "te", "tem", "temp"} {
					Expect(apply(ctx, key, set("alias", text))).To(Succeed())
				}
				reader := aliasSvc.NewReader(db)
				Expect(reader.Retrieve(ctx, r.Key, other.Key())).To(Equal("temp"))
				Expect(reader.Retrieve(ctx, r.Key, ch.Key())).To(Equal("temp"))
				Expect(reader.Resolve(ctx, r.Key, "temp")).Error().
					To(MatchError(alias.ErrAmbiguous))
			},
		)

		It("Should refuse an alias that does not exist", func(ctx SpecContext) {
			other := alias.Alias{Range: r.Key, Channel: createChannel(ctx).Key()}
			Expect(apply(ctx, other.GorpKey(), set("alias", "After"))).
				To(MatchError(query.ErrNotFound))
		})

		It("Should broadcast a set and an applied batch", func(ctx SpecContext) {
			rec := &Recorder[string, alias.Action]{}
			DeferCleanup(aliasSvc.Actions().OnChange(rec.Record))
			other := createChannel(ctx)
			Expect(write(ctx, func(w alias.Writer) error {
				return w.Set(ctx, r.Key, other.Key(), "Created")
			})).To(Succeed())
			Expect(apply(ctx, key, set("alias", "After"))).To(Succeed())
			frames := rec.Snapshot()
			created := alias.Alias{Range: r.Key, Channel: other.Key(), Alias: "Created"}
			Expect(frames).To(Equal([]frame{
				{
					Key:         created.GorpKey(),
					DispatchKey: "dk",
					Seq:         frames[0].Seq,
					Actions: []alias.Action{
						alias.NewCreateAction(alias.CreatePayload{Alias: created}),
					},
				},
				{
					Key:         key,
					DispatchKey: "dk",
					Seq:         frames[1].Seq,
					Actions:     []alias.Action{set("alias", "After")},
				},
			}))
		})
	})

	Describe("Retrieve", func() {
		It("Should get an alias for a channel on a range", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			a := MustSucceed(aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()))
			Expect(a).To(Equal("Alias"))
		})

		It("Should return an error if an alias can't be found", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(
				aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()),
			).Error().
				To(MatchError(query.ErrNotFound))
		})

		It(
			"Should fallback to the parent range if the alias is not found",
			func(ctx SpecContext) {
				ch := createChannel(ctx)
				tx := DeferClose(db.OpenTx())
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
				).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
					Parent: &parent.Key,
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				a := MustSucceed(aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()))
				Expect(a).To(Equal("Alias"))
			},
		)
	})

	It("Should fall back to the parent range through a snapshot", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		tx := DeferClose(db.OpenTx())
		parent := ranger.Range{
			Name: "Parent",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
		Expect(
			aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
		).To(Succeed())
		r := ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(7 * telem.Second),
				End:   telem.TimeStamp(9 * telem.Second),
			},
			Parent: &parent.Key,
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		var a string
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			a, err = aliasSvc.NewReader(snap).Retrieve(ctx, r.Key, ch.Key())
			return err
		})).To(Succeed())
		Expect(a).To(Equal("Alias"))
	})

	It("Should resolve, list, and search aliases through a snapshot", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx)
		r := ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := rangerSvc.NewWriter(tx).Create(ctx, &r); err != nil {
				return err
			}
			return aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "snapshot alias")
		})).To(Succeed())
		var (
			resolved channel.Key
			listed   map[channel.Key]string
			found    []channel.Key
		)
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			reader := aliasSvc.NewReader(snap)
			resolved, err = reader.Resolve(ctx, r.Key, "snapshot alias")
			if err != nil {
				return err
			}
			if listed, err = reader.List(ctx, r.Key); err != nil {
				return err
			}
			found, err = reader.Search(ctx, r.Key, "snapshot", visible)
			return err
		})).To(Succeed())
		Expect(resolved).To(Equal(ch.Key()))
		Expect(listed).To(Equal(map[channel.Key]string{ch.Key(): "snapshot alias"}))
		Expect(found).To(Equal([]channel.Key{ch.Key()}))
	})

	Describe("Delete", func() {
		It("Should delete an alias for a channel on a range", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Delete(ctx, r.Key, ch.Key())).To(Succeed())
			Expect(
				aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()),
			).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Resolve", func() {
		It("Should resolve an alias for a channel on a range", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			resolved := MustSucceed(aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "Alias"))
			Expect(resolved).To(Equal(ch.Key()))
		})

		It(
			"Should return an error if an alias can't be resolved",
			func(ctx SpecContext) {
				ch := createChannel(ctx)
				tx := DeferClose(db.OpenTx())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
				).To(Succeed())
				Expect(
					aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "not_an_alias"),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)

		It(
			"Should fallback to the parent range if the alias is not found",
			func(ctx SpecContext) {
				ch := createChannel(ctx)
				tx := DeferClose(db.OpenTx())
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
				).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
					Parent: &parent.Key,
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				resolved := MustSucceed(
					aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "Alias"),
				)
				Expect(resolved).To(Equal(ch.Key()))
			},
		)

		It(
			"Should return an error if the alias can't be resolved on both the child range and its parent",
			func(ctx SpecContext) {
				ch := createChannel(ctx)
				tx := DeferClose(db.OpenTx())
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
				).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
					Parent: &parent.Key,
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				Expect(
					aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "not_an_alias"),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("Holders", func() {
		It("Should return every channel that holds the alias", func(ctx SpecContext) {
			a, b := createChannel(ctx), createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := createRange(ctx, tx, nil)
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, r, a.Key(), "x")).To(Succeed())
			Expect(w.Set(ctx, r, b.Key(), "y")).To(Succeed())
			Expect(w.Apply(
				ctx,
				alias.Alias{Range: r, Channel: b.Key()}.GorpKey(),
				alias.NewSetAction(
					alias.SetPayload{Path: []string{"alias"}, Value: "x"},
				),
			)).To(Succeed())
			Expect(aliasSvc.NewReader(tx).Holders(ctx, r, "x")).
				To(ConsistOf(a.Key(), b.Key()))
		})

		It("Should return no channels for an alias no range has", func(
			ctx SpecContext,
		) {
			tx := DeferClose(db.OpenTx())
			r := createRange(ctx, tx, nil)
			Expect(aliasSvc.NewReader(tx).Holders(ctx, r, "absent")).To(BeEmpty())
		})
	})

	Describe("Resolve exactly", func() {
		DescribeTable("Should not match an alias as a pattern", func(
			ctx SpecContext,
			term string,
		) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := createRange(ctx, tx, nil)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r, ch.Key(), "p10")).To(Succeed())
			Expect(aliasSvc.NewReader(tx).Resolve(ctx, r, term)).Error().
				To(MatchError(query.ErrNotFound))
		},
			Entry("prefix", "p1"),
			Entry("wildcard", "p.0"),
			Entry("anchored", "^p"),
		)
	})

	Describe("Ancestor limit", func() {
		It("Should resolve an alias on a range 64 ancestors up", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			root, leaf := createChain(ctx, tx, 64)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, root, ch.Key(), "x")).To(Succeed())
			Expect(aliasSvc.NewReader(tx).Resolve(ctx, leaf, "x")).To(Equal(ch.Key()))
		})

		It("Should return an error past 64 ancestors", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			root, leaf := createChain(ctx, tx, 65)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, root, ch.Key(), "x")).To(Succeed())
			reader := aliasSvc.NewReader(tx)
			Expect(reader.Resolve(ctx, leaf, "x")).Error().
				To(MatchError(alias.ErrAncestorLimit))
			Expect(reader.Retrieve(ctx, leaf, ch.Key())).Error().
				To(MatchError(alias.ErrAncestorLimit))
			Expect(reader.List(ctx, leaf)).Error().
				To(MatchError(alias.ErrAncestorLimit))
		})
	})

	Describe("Search", func() {
		var (
			r, other      ranger.Range
			inlet, outlet channel.Channel
			tx            gorp.Tx
		)
		BeforeEach(func(ctx SpecContext) {
			inlet, outlet = createChannel(ctx), createChannel(ctx)
			tx = DeferClose(db.OpenTx())
			r = ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			other = r
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &other)).To(Succeed())
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, r.Key, inlet.Key(), "turbopump inlet")).To(Succeed())
			Expect(w.Set(ctx, r.Key, outlet.Key(), "turbopump")).To(Succeed())
			Expect(w.Set(ctx, other.Key, inlet.Key(), "valve")).To(Succeed())
		})

		It("Should return the channels whose alias on the range matches the term",
			func(ctx SpecContext) {
				Expect(aliasSvc.NewReader(tx).Search(ctx, r.Key, "turbopump", visible)).
					To(Equal([]channel.Key{outlet.Key(), inlet.Key()}))
				Expect(aliasSvc.NewReader(tx).Search(ctx, r.Key, "valve", visible)).
					To(BeEmpty())
			})

		It("Should skip the aliases of channels that visible rejects",
			func(ctx SpecContext) {
				Expect(aliasSvc.NewReader(tx).Search(
					ctx,
					r.Key,
					"turbopump",
					func(ch channel.Key) bool { return ch != outlet.Key() },
				)).To(Equal([]channel.Key{inlet.Key()}))
			})

		It("Should find aliases through the resource search", func(ctx SpecContext) {
			q := search.Query[resource.ID]{Term: "turbopump inlet", Limit: 10}
			Expect(MustSucceed(aliasSvc.Resources().Search(ctx, tx, q)).Keys).
				To(Equal([]resource.ID{alias.ResourceID(r.Key, inlet.Key())}))
		})

		It(
			"Should drop a deleted alias from the resource search",
			func(ctx SpecContext) {
				id := alias.ResourceID(r.Key, inlet.Key())
				q := search.Query[resource.ID]{Term: "turbopump inlet", Limit: 10}
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(MustSucceed(aliasSvc.Resources().Search(ctx, db, q)).Keys).
					To(ContainElement(id))
				del := DeferClose(db.OpenTx())
				Expect(aliasSvc.NewWriter(del).Delete(ctx, r.Key, inlet.Key())).
					To(Succeed())
				Expect(del.Commit(ctx)).To(Succeed())
				Expect(MustSucceed(aliasSvc.Resources().Search(ctx, db, q)).Keys).
					ToNot(ContainElement(id))
				Expect(aliasSvc.Resources().Retrieve(ctx, db, []resource.ID{id})).
					To(BeEmpty())
			},
		)
	})

	Describe("SearchChannels", func() {
		var (
			r               ranger.Range
			term            string
			byName, byAlias channel.Channel
		)
		BeforeEach(func(ctx SpecContext) {
			r = ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rangerSvc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			term = fmt.Sprintf("probe%09d", rand.IntN(999999999))
			byName = channel.Channel{
				DataType: telem.Float32T,
				Name:     term,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &byName)
			})).To(Succeed())
			byAlias = createChannel(ctx)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return aliasSvc.NewWriter(tx).
					Set(ctx, r.Key, byAlias.Key(), term+" inlet")
			})).
				To(Succeed())
		})

		search := func(ctx SpecContext, q channel.Retrieve, limit int) []channel.Key {
			GinkgoHelper()
			chs := MustSucceed(aliasSvc.NewReader(db).SearchChannels(
				ctx, q, r.Key, term, visible, limit,
			))
			return lo.Map(chs, func(ch channel.Channel, _ int) channel.Key {
				return ch.Key()
			})
		}

		both := func() channel.Retrieve {
			return channelSvc.NewRetrieve().
				Where(channel.MatchKeys(byName.Key(), byAlias.Key()))
		}

		It("Should rank the channels whose alias matches the term first", func(
			ctx SpecContext,
		) {
			Expect(search(ctx, both(), 0)).
				To(Equal([]channel.Key{byAlias.Key(), byName.Key()}))
		})

		It("Should search the channels through a snapshot", func(ctx SpecContext) {
			var chs []channel.Channel
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
				chs, err = aliasSvc.NewReader(snap).
					SearchChannels(ctx, both(), r.Key, term, visible, 0)
				return err
			})).To(Succeed())
			Expect(lo.Map(chs, func(ch channel.Channel, _ int) channel.Key {
				return ch.Key()
			})).To(Equal([]channel.Key{byAlias.Key(), byName.Key()}))
		})

		It("Should limit the merged channels", func(ctx SpecContext) {
			Expect(search(ctx, both(), 1)).To(Equal([]channel.Key{byAlias.Key()}))
			Expect(search(ctx, both(), 2)).
				To(Equal([]channel.Key{byAlias.Key(), byName.Key()}))
		})

		It("Should list a channel that matches by alias and by name once", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return aliasSvc.NewWriter(tx).
					Set(ctx, r.Key, byName.Key(), term+" outlet")
			})).To(Succeed())
			second := channel.Channel{
				DataType: telem.Float32T,
				Name:     term + "_second",
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &second)
			})).To(Succeed())
			q := func() channel.Retrieve {
				return channelSvc.NewRetrieve().
					Where(channel.MatchKeys(byName.Key(), second.Key()))
			}
			Expect(search(ctx, q(), 0)).
				To(Equal([]channel.Key{byName.Key(), second.Key()}))
			Expect(search(ctx, q(), 2)).
				To(Equal([]channel.Key{byName.Key(), second.Key()}))
		})

		It("Should leave out an alias match that the query rejects", func(
			ctx SpecContext,
		) {
			q := channelSvc.NewRetrieve().Where(channel.MatchKeys(byName.Key()))
			Expect(search(ctx, q, 0)).To(Equal([]channel.Key{byName.Key()}))
		})

		It("Should skip the alias of a deleted channel", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, byAlias.Key(), false)
			})).
				To(Succeed())
			Expect(search(ctx, channelSvc.NewRetrieve(), 0)).
				To(Equal([]channel.Key{byName.Key()}))
		})
	})

	Describe("List", func() {
		It("Should list the aliases on a range", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			aliases := MustSucceed(aliasSvc.NewReader(tx).List(ctx, r.Key))
			Expect(aliases).To(HaveKeyWithValue(ch.Key(), "Alias"))
		})

		It("Should list the alias of the range over the alias of its parent", func(
			ctx SpecContext,
		) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			parent := createRange(ctx, tx, nil)
			child := createRange(ctx, tx, &parent)
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, parent, ch.Key(), "parent")).To(Succeed())
			Expect(w.Set(ctx, child, ch.Key(), "child")).To(Succeed())
			Expect(aliasSvc.NewReader(tx).List(ctx, child)).
				To(Equal(map[channel.Key]string{ch.Key(): "child"}))
		})

		It("Should list only the aliases of the range", func(ctx SpecContext) {
			listed, other := createChannel(ctx), createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			rng, sibling := createRange(ctx, tx, nil), createRange(ctx, tx, nil)
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, rng, listed.Key(), "listed")).To(Succeed())
			Expect(w.Set(ctx, sibling, other.Key(), "other")).To(Succeed())
			Expect(aliasSvc.NewReader(tx).List(ctx, rng)).
				To(Equal(map[channel.Key]string{listed.Key(): "listed"}))
		})

		It("Should list the aliases on a range and its parent", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			parent := ranger.Range{
				Name: "RetrieveParent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
			).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
				Parent: &parent.Key,
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			aliases := MustSucceed(aliasSvc.NewReader(tx).List(ctx, r.Key))
			Expect(aliases).To(HaveKeyWithValue(ch.Key(), "Alias"))
		})
	})

	Context("Range delete", func() {
		It("Should delete the aliases of a deleted range", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{Name: "Range", TimeRange: telem.NewRangeSeconds(5, 10)}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).
				To(Succeed())
			Expect(rangerSvc.NewWriter(tx).Delete(ctx, r.Key)).To(Succeed())
			Expect(aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key())).Error().
				To(MatchError(query.ErrNotFound))
		})
		It(
			"Should reject an alias on a range that does not exist",
			func(ctx SpecContext) {
				ch := createChannel(ctx)
				tx := DeferClose(db.OpenTx())
				missing := uuid.New()
				err := aliasSvc.NewWriter(tx).Set(ctx, missing, ch.Key(), "Alias")
				Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
				Expect(err).To(MatchError(
					"range: range " + missing.String() + " does not exist: validation error",
				))
			},
		)
	})

	Describe("Resources", func() {
		It("Should name each alias by its alias text", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name:      "Burn",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Tank pressure")).
				To(Succeed())
			id := alias.ResourceID(r.Key, ch.Key())
			Expect(aliasSvc.Resources().Retrieve(ctx, tx, []resource.ID{id})).
				To(Equal([]resource.Resource{{ID: id, Name: "Tank pressure"}}))
		})

		It(
			"Should return a validation error for a malformed key",
			func(ctx SpecContext) {
				Expect(aliasSvc.Resources().Retrieve(
					ctx,
					db,
					[]resource.ID{{Type: resource.TypeRangeAlias, Key: "pump"}},
				)).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(`invalid range alias key "pump"`)),
				))
			},
		)

		It("Should leave out an ID that names no alias", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name:      "Burn",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Tank pressure")).
				To(Succeed())
			id := alias.ResourceID(r.Key, ch.Key())
			Expect(aliasSvc.Resources().Retrieve(
				ctx,
				tx,
				[]resource.ID{alias.ResourceID(uuid.New(), ch.Key()), id},
			)).To(Equal([]resource.Resource{{ID: id, Name: "Tank pressure"}}))
		})

		It("Should list each alias by its alias text", func(ctx SpecContext) {
			ch := createChannel(ctx)
			tx := DeferClose(db.OpenTx())
			r := ranger.Range{
				Name:      "Burn",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Listed alias")).
				To(Succeed())
			id := alias.ResourceID(r.Key, ch.Key())
			Expect(aliasSvc.Resources().List(ctx, tx, 10000, nil)).
				To(ContainElement(resource.Resource{ID: id, Name: "Listed alias"}))
		})
	})
})

// visible accepts every channel.
func visible(channel.Key) bool { return true }
