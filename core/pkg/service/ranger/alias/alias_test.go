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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Alias", Ordered, func() {
	var (
		db         *gorp.DB
		otg        *ontology.Ontology
		rangerSvc  *ranger.Service
		aliasSvc   *alias.Service
		channelSvc *channel.Service
		tx         gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Group:    groupSvc,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Group:    groupSvc,
			Label:    labelSvc,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostProvider: node.Cluster,
			Ontology:     otg,
			Group:        groupSvc,
			Status:       statusSvc,
		}))
		rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Group:    groupSvc,
			Label:    labelSvc,
		}))
		aliasSvc = MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
			DB:              node.DB,
			Ontology:        otg,
			Channel:         channelSvc,
			ParentRetriever: rangerSvc,
		}))
	})
	BeforeEach(func() {
		tx = DeferClose(db.OpenTx())
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
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
		return ch
	}

	Describe("Set", func() {
		It("Should set an alias for a channel on a range", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
		})
	})

	Describe("Retrieve", func() {
		It("Should get an alias for a channel on a range", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			a := MustSucceed(aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()))
			Expect(a).To(Equal("Alias"))
		})

		It("Should return an error if an alias can't be found", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()),
			).Error().
				To(MatchError(query.ErrNotFound))
		})

		It(
			"Should fallback to the parent range if the alias is not found",
			func(ctx SpecContext) {
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := createChannel(ctx)
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
				).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
					Parent: &parent,
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				a := MustSucceed(aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()))
				Expect(a).To(Equal("Alias"))
			},
		)
	})

	Describe("Delete", func() {
		It("Should delete an alias for a channel on a range", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
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
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			resolved := MustSucceed(aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "Alias"))
			Expect(resolved).To(Equal(ch.Key()))
		})

		It(
			"Should return an error if an alias can't be resolved",
			func(ctx SpecContext) {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := createChannel(ctx)
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
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := createChannel(ctx)
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
				).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
					Parent: &parent,
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
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := createChannel(ctx)
				Expect(
					aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
				).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
					Parent: &parent,
				}
				Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				Expect(
					aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "not_an_alias"),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("Search", func() {
		var (
			r, other      ranger.Range
			inlet, outlet channel.Channel
		)
		BeforeEach(func(ctx SpecContext) {
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
			inlet, outlet = createChannel(ctx), createChannel(ctx)
			w := aliasSvc.NewWriter(tx)
			Expect(w.Set(ctx, r.Key, inlet.Key(), "turbopump inlet")).To(Succeed())
			Expect(w.Set(ctx, r.Key, outlet.Key(), "turbopump")).To(Succeed())
			Expect(w.Set(ctx, other.Key, inlet.Key(), "valve")).To(Succeed())
		})

		It("Should return the channels whose alias on the range matches the term",
			func(ctx SpecContext) {
				Expect(aliasSvc.NewReader(tx).Search(ctx, r.Key, "turbopump")).
					To(Equal([]channel.Key{outlet.Key(), inlet.Key()}))
				Expect(aliasSvc.NewReader(tx).Search(ctx, r.Key, "valve")).To(BeEmpty())
			})

		It("Should find aliases through the ontology search", func(ctx SpecContext) {
			Expect(otg.Search(ctx, tx, "turbopump inlet")).To(Equal([]ontology.ID{
				alias.OntologyID(r.Key, inlet.Key()),
			}))
		})

		It(
			"Should drop a deleted alias from the ontology search",
			func(ctx SpecContext) {
				id := alias.OntologyID(r.Key, inlet.Key())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(otg.Search(ctx, nil, "turbopump inlet")).To(ContainElement(id))
				del := DeferClose(db.OpenTx())
				Expect(aliasSvc.NewWriter(del).Delete(ctx, r.Key, inlet.Key())).
					To(Succeed())
				Expect(del.Commit(ctx)).To(Succeed())
				Expect(otg.Search(ctx, nil, "turbopump inlet")).
					ToNot(ContainElement(id))
			},
		)
	})

	Describe("List", func() {
		It("Should list the aliases on a range", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			aliases := MustSucceed(aliasSvc.NewReader(tx).List(ctx, r.Key))
			Expect(aliases).To(HaveKeyWithValue(ch.Key(), "Alias"))
		})

		It("Should list the aliases on a range and its parent", func(ctx SpecContext) {
			parent := ranger.Range{
				Name: "RetrieveParent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias"),
			).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
				Parent: &parent,
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			aliases := MustSucceed(aliasSvc.NewReader(tx).List(ctx, r.Key))
			Expect(aliases).To(HaveKeyWithValue(ch.Key(), "Alias"))
		})
	})

	Context("Ontology", func() {
		It("Should find a created alias in the ontology", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(
				aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias"),
			).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(alias.OntologyID(r.Key, ch.Key())).
				Entry(&res).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(res).To(Equal(ontology.Resource{
				ID:   alias.OntologyID(r.Key, ch.Key()),
				Name: "Alias",
				Data: map[string]any{
					"range":   r.Key.String(),
					"channel": ch.Key().StorageKey(),
					"alias":   "Alias",
				},
			}))
		})
	})
})
