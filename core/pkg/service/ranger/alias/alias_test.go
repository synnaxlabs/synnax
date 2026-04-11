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
	"time"

	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Alias", Ordered, func() {
	var (
		dist      mock.Node
		rangerSvc *ranger.Service
		aliasSvc  *alias.Service
		labelSvc  *label.Service
		tx        gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Search:   dist.Search,
		}))
		rangerSvc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Label:    labelSvc,
			Search:   dist.Search,
		}))
		aliasSvc = MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
			DB:              dist.DB,
			Ontology:        dist.Ontology,
			Channel:         svcchannel.Wrap(dist.Channel),
			ParentRetriever: rangerSvc,
			Search:          dist.Search,
		}))
		Expect(dist.Search.Initialize(ctx)).To(Succeed())
		DeferCleanup(func() { Expect(dist.Close()).To(Succeed()) })
	})
	BeforeEach(func() {
		tx = dist.DB.OpenTx()
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

	channelCount := 0
	createChannel := func(ctx context.Context) channel.Channel {
		channelCount++
		ch := channel.Channel{DataType: telem.Float32T, Name: fmt.Sprintf("test_%d", channelCount), Virtual: true}
		Expect(dist.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
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
			_, err := aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key())
			Expect(err).To(HaveOccurred())
		})

		It("Should fallback to the parent range if the alias is not found", func(ctx SpecContext) {
			parent := ranger.Range{
				Name: "Parent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			ch := createChannel(ctx)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias")).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
			a := MustSucceed(aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key()))
			Expect(a).To(Equal("Alias"))
		})
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Delete(ctx, r.Key, ch.Key())).To(Succeed())
			_, err := aliasSvc.NewReader(tx).Retrieve(ctx, r.Key, ch.Key())
			Expect(err).To(HaveOccurred())
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
			resolved := MustSucceed(aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "Alias"))
			Expect(resolved).To(Equal(ch.Key()))
		})

		It("Should return an error if an alias can't be resolved", func(ctx SpecContext) {
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := createChannel(ctx)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
			_, err := aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "not_an_alias")
			Expect(err).To(MatchError(query.ErrNotFound))
		})

		It("Should fallback to the parent range if the alias is not found", func(ctx SpecContext) {
			parent := ranger.Range{
				Name: "Parent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			ch := createChannel(ctx)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias")).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
			resolved := MustSucceed(aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "Alias"))
			Expect(resolved).To(Equal(ch.Key()))
		})

		It("Should return an error if the alias can't be resolved on both the child range and its parent", func(ctx SpecContext) {
			parent := ranger.Range{
				Name: "Parent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			ch := createChannel(ctx)
			Expect(aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias")).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
			_, err := aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "not_an_alias")
			Expect(err).To(HaveOccurred())
		})
	})

	Specify("Aliases should be searchable by the ontology", func(ctx SpecContext) {
		time.Sleep(10 * time.Millisecond)
		r := ranger.Range{
			Name: "Range",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(rangerSvc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		ch := createChannel(ctx)
		Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		Eventually(func(g Gomega) {
			g.Expect(aliasSvc.NewReader(nil).Search(ctx, r.Key, "Alias")).To(ContainElement(ch.Key()))
		}).Should(Succeed())
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, parent.Key, ch.Key(), "Alias")).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
			}
			Expect(rangerSvc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
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
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
			var res ontology.Resource
			Expect(dist.Ontology.NewRetrieve().
				WhereIDs(alias.OntologyID(r.Key, ch.Key())).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			var out alias.Alias
			Expect(res.Parse(&out)).To(Succeed())
			Expect(out.Channel).To(Equal(ch.Key()))
			Expect(out.Range).To(Equal(r.Key))
			Expect(out.Alias).To(Equal("Alias"))
		})
	})
})
