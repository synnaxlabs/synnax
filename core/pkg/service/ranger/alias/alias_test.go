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
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Alias", Ordered, func() {
	var (
		db        *gorp.DB
		rangerSvc *ranger.Service
		aliasSvc  *alias.Service
		otg       *ontology.Ontology
		tx        gorp.Tx
		closer    io.Closer
	)
	BeforeAll(func(ctx SpecContext) {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{
			DB: db,
		}))
		searchIdx := MustSucceed(search.Open())
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg, Search: searchIdx}))
		lab := MustSucceed(label.OpenService(ctx, label.ServiceConfig{DB: db, Ontology: otg, Group: g, Search: searchIdx}))
		rangerSvc = MustSucceed(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lab,
			Search:   searchIdx,
		}))
		aliasSvc = MustSucceed(alias.OpenService(ctx, alias.ServiceConfig{
			DB:              db,
			Ontology:        otg,
			ParentRetriever: rangerSvc,
			Search:          searchIdx,
		}))
		Expect(searchIdx.Initialize(ctx)).To(Succeed())
		closer = xio.MultiCloser{db, otg, g, rangerSvc, aliasSvc}
	})
	AfterAll(func() {
		Expect(closer.Close()).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
			_, err := aliasSvc.NewReader(tx).Resolve(ctx, r.Key, "not_an_alias")
			Expect(err).To(HaveOccurredAs(query.ErrNotFound))
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
		ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
		Expect(gorp.NewCreate[channel.Key, channel.Channel]().
			Entry(&ch).
			Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
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
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
			Expect(aliasSvc.NewWriter(tx).Set(ctx, r.Key, ch.Key(), "Alias")).To(Succeed())
			var res ontology.Resource
			Expect(otg.NewRetrieve().
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
