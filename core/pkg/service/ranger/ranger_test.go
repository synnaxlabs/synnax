// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ranger", Ordered, func() {
	var (
		db     *gorp.DB
		svc    *ranger.Service
		ctx    context.Context
		w      ranger.Writer
		otg    *ontology.Ontology
		tx     gorp.Tx
		closer io.Closer
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		ctx = context.Background()
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{
			DB:           db,
			EnableSearch: new(true),
		}))
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
		lab := MustSucceed(label.OpenService(ctx, label.ServiceConfig{DB: db, Ontology: otg, Group: g}))
		svc = MustSucceed(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lab,
		}))
		Expect(otg.InitializeSearchIndex(ctx)).To(Succeed())
		closer = xio.MultiCloser{db, otg, g, svc}
	})
	AfterAll(func() {
		Expect(closer.Close()).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
		})
		It("should return an error if the time range is invalid", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(10 * telem.Second),
					End:   telem.TimeStamp(5 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).
				To(MatchError(
					ContainSubstring("time_range.start cannot be after time_range.end"),
				))
		})
		It("should create a range with start equal to end", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(5 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
		})
		It("Should not override the UUID if it is already set", func() {
			k := uuid.New()
			r := &ranger.Range{
				Key:  k,
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).To(Equal(k))
		})
		Context("Parent Management", func() {
			It("Should set a custom parent for the range", func() {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateWithParent(ctx, r, parent.OntologyID())).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
			})
			It("Should NOT re-set the custom parent when the range exists but no parent is provided", func() {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateWithParent(ctx, r, parent.OntologyID())).To(Succeed())
				Expect(w.Create(ctx, r)).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
			})
			It("Should change the custom parent when the range exists and a new parent is provided", func() {
				parent1 := ranger.Range{
					Name:      "Parent1",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent1)).To(Succeed())
				parent2 := ranger.Range{
					Name:      "Parent2",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent2)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateWithParent(ctx, r, parent1.OntologyID())).To(Succeed())
				Expect(w.CreateWithParent(ctx, r, parent2.OntologyID())).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent2.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
				var res2 ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent1.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res2).
					Exec(ctx, tx)).To(HaveOccurredAs(query.ErrNotFound))
			})
			It("Should create multiple ranges with the same parent", func() {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r1 := ranger.Range{
					Name:      "Range1",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				r2 := ranger.Range{
					Name:      "Range2",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateManyWithParent(ctx, &[]ranger.Range{r1, r2}, parent.OntologyID())).To(Succeed())
				var res []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(2))
			})
			Context("Parent Method", func() {
				It("Should get the parent of the range", func() {
					parent := ranger.Range{
						Name:      "Parent",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(w.Create(ctx, &parent)).To(Succeed())
					r := ranger.Range{
						Name:      "Range",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(w.CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
					p := MustSucceed(r.RetrieveParent(ctx))
					Expect(p.Key).To(Equal(parent.Key))
				})
				It("Should return an error if the range has no parent", func() {
					p := ranger.Range{
						Name:      "Parent",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(w.Create(ctx, &p)).To(Succeed())
					_, err := p.RetrieveParent(ctx)
					Expect(err).To(HaveOccurredAs(query.ErrNotFound))
				})
			})
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve a range by its key", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It("Should retrieve a range by its name", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereNames(r.Name).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It("Should retrieve any ranges that overlap a given time range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereOverlapsWith(telem.TimeRange{
				Start: telem.TimeStamp(7 * telem.Second),
				End:   telem.TimeStamp(9 * telem.Second),
			}).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
	})

	Describe("Delete", func() {
		It("Should delete a range by its key", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, r.Key)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should delete all child ranges when a range is deleted", func() {
			parent := ranger.Range{
				Name: "Parent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, parent.Key)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
	})

	Describe("KV", func() {
		It("Should be able to store key-value pairs in a range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(r.SetKV(ctx, "key", "value")).To(Succeed())
		})
		It("Should be able to retrieve key-value pairs from a range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(r.SetKV(ctx, "key", "value")).To(Succeed())
			value, err := r.Get(ctx, "key")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal("value"))
		})
		It("Should be able to delete key-value pairs from a range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(r.SetKV(ctx, "key", "value")).To(Succeed())
			Expect(r.DeleteKV(ctx, "key")).To(Succeed())
			_, err := r.Get(ctx, "key")
			Expect(err).To(HaveOccurred())
		})
		It("Should set many key-value pairs on the range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(r.SetManyKV(ctx, []ranger.KVPair{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			})).To(Succeed())
			value, err := r.Get(ctx, "key1")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal("value1"))
			value, err = r.Get(ctx, "key2")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal("value2"))
		})
		It("Should be able to list all key-value pairs in a range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(r.SetKV(ctx, "key1", "value1")).To(Succeed())
			Expect(r.SetKV(ctx, "key2", "value2")).To(Succeed())
			meta, err := r.ListKV(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta).To(Equal([]ranger.KVPair{
				{Range: r.Key, Key: "key1", Value: "value1"},
				{Range: r.Key, Key: "key2", Value: "value2"},
			}))
			Expect(r.DeleteKV(ctx, "key1")).To(Succeed())
			meta, err = r.ListKV(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta).To(Equal([]ranger.KVPair{
				{Range: r.Key, Key: "key2", Value: "value2"},
			}))
		})
	})

	Describe("Alias", func() {

		Describe("Set", func() {
			It("Should set an Alias for a channel on a range", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
			})
		})

		Describe("Get", func() {
			It("Should get an Alias for a channel on a range", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				alias, err := r.RetrieveAlias(ctx, ch.Key())
				Expect(err).ToNot(HaveOccurred())
				Expect(alias).To(Equal("Alias"))
			})

			It("Should return an error if an Alias can't be found", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				_, err := r.RetrieveAlias(ctx, ch.Key())
				Expect(err).To(HaveOccurred())
			})

			It("Should fallback to the parent range if the Alias is not found", func() {
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				parent = parent.UseTx(tx)
				Expect(parent.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
				alias, err := r.RetrieveAlias(ctx, ch.Key())
				Expect(err).ToNot(HaveOccurred())
				Expect(alias).To(Equal("Alias"))
			})
		})

		Describe("Delete", func() {
			It("Should delete an Alias for a channel on a range", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				Expect(r.DeleteAlias(ctx, ch.Key())).To(Succeed())
				_, err := r.RetrieveAlias(ctx, ch.Key())
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Resolve", func() {

			It("Should resolve an Alias for a channel on a range", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				alias, err := r.ResolveAlias(ctx, "Alias")
				Expect(err).ToNot(HaveOccurred())
				Expect(alias).To(Equal(ch.Key()))
			})

			It("Should return an error if an Alias can't be resolved", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				_, err := r.ResolveAlias(ctx, "not_an_alias")
				Expect(err).To(HaveOccurredAs(query.ErrNotFound))
			})

			It("Should fallback to the parent range if the Alias is not found", func() {
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				parent = parent.UseTx(tx)
				Expect(parent.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
				alias, err := r.ResolveAlias(ctx, "Alias")
				Expect(err).ToNot(HaveOccurred())
				Expect(alias).To(Equal(ch.Key()))
			})

			It("Should return an error if the Alias can't be resolved on both the child range and its parent", func() {
				parent := ranger.Range{
					Name: "Parent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				parent = parent.UseTx(tx)
				Expect(parent.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
				_, err := r.ResolveAlias(ctx, "not_an_alias")
				Expect(err).To(HaveOccurred())
			})
		})

		Specify("Aliases should be searchable by the ontology", func() {
			time.Sleep(10 * time.Millisecond)
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(gorp.NewCreate[channel.Key, channel.Channel]().
				Entry(&ch).
				Exec(ctx, tx)).To(Succeed())
			r = r.UseTx(tx)
			Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(r.SearchAliases(ctx, "Alias")).To(ContainElement(ch.Key()))
			}).Should(Succeed())
		})

		Describe("List", func() {

			It("Should list the aliases on a range", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				aliases, err := r.RetrieveAliases(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(aliases).To(HaveKeyWithValue(ch.Key(), "Alias"))
			})

			It("Should list the aliases on a range and its parent", func() {
				parent := ranger.Range{
					Name: "RetrieveParent",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				parent = parent.UseTx(tx)
				Expect(parent.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(7 * telem.Second),
						End:   telem.TimeStamp(9 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
				aliases, err := r.RetrieveAliases(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(aliases).To(HaveKeyWithValue(ch.Key(), "Alias"))
			})

		})

		Context("Ontology", func() {
			It("Should find a created Alias in the ontology", func() {
				r := ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				ch := channel.Channel{Leaseholder: 1, LocalKey: 1}
				Expect(gorp.NewCreate[channel.Key, channel.Channel]().
					Entry(&ch).
					Exec(ctx, tx)).To(Succeed())
				r = r.UseTx(tx)
				Expect(r.SetAlias(ctx, ch.Key(), "Alias")).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(ranger.AliasOntologyID(r.Key, ch.Key())).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				var out ranger.Alias
				Expect(res.Parse(&out)).To(Succeed())
				Expect(out.Channel).To(Equal(ch.Key()))
				Expect(out.Range).To(Equal(r.Key))
				Expect(out.Alias).To(Equal("Alias"))
			})
		})
	})
})
