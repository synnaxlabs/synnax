// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search_test

import (
	"context"
	"io"
	"iter"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/change"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

type mockService struct {
	observe.Noop[iter.Seq[ontology.Change]]
	resourceType ontology.ResourceType
	fields       []string
}

func (m *mockService) Type() ontology.ResourceType { return m.resourceType }

func (m *mockService) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return func(func(ontology.Resource) bool) {}, xio.NopCloser, nil
}

func (m *mockService) SearchableFields() []string { return m.fields }

type observableMockService struct {
	observe.Observer[iter.Seq[ontology.Change]]
	resourceType ontology.ResourceType
	resources    []ontology.Resource
	fields       []string
}

func (m *observableMockService) Type() ontology.ResourceType { return m.resourceType }

func (m *observableMockService) OpenNexter(context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values(m.resources), xio.NopCloser, nil
}

func (m *observableMockService) SearchableFields() []string { return m.fields }

func newIndex(svcs ...*mockService) *search.Index {
	idx := MustSucceed(search.Open())
	for _, svc := range svcs {
		idx.RegisterService(svc)
	}
	Expect(idx.Initialize(context.Background())).To(Succeed())
	return idx
}

var _ = Describe("Search", func() {
	Describe("Search", func() {
		var idx *search.Index
		BeforeEach(func() {
			idx = newIndex(&mockService{resourceType: "test"})
		})
		DescribeTable("Searching",
			func(ctx SpecContext, res ontology.Resource, term string) {
				Expect(idx.IndexResources([]ontology.Resource{res})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				})).To(HaveLen(1))
			},
			Entry("Exact Match", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "test"),
			Entry("Word in Multi-Word Search", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "October 28 Gooster",
			}, "Gooster"),
			Entry("Near match to term", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "October 27 Gooster",
			}, "Gooster"),
			Entry("Underscores in term", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "gse_ai_15",
			}, "ai_15"),
			Entry("All Caps", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "BBTPC",
			}, "BTTPC"),
			Entry("Upper and lowercase", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "BBTPC",
			}, "bttpc"),
			Entry("Close Match in Multi-Word Search", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "BBTPC Sim",
			}, "BTTPC"),
			Entry("Partial Match Beginning", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "Channel",
			}, "ch"),
			Entry("Scream Case with Underscore Exact", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "DAQ_PT",
			}, "DAQ_PT"),
			Entry("Scream Case with Underscore Partial", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "DAQ_PT_1",
			}, "DAQ_PT"),
		)
		DescribeTable("Prioritization",
			func(ctx SpecContext, resources []ontology.Resource, term string, first ontology.ID) {
				Expect(idx.IndexResources(resources)).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res).To(Not(BeEmpty()))
				Expect(res[0].Key).To(Equal(first.Key))
			},
			Entry("Exact Match First", []ontology.Resource{
				{
					ID:   ontology.ID{Type: "test", Key: "1"},
					Name: "test",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "2"},
					Name: "test2",
				},
			}, "test", ontology.ID{Type: "test", Key: "1"}),
			Entry("Exact Match Multiple Words", []ontology.Resource{
				{
					ID:   ontology.ID{Type: "test", Key: "3"},
					Name: "October 30 Gooster",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "4"},
					Name: "October 31 Gooster",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "1"},
					Name: "October 28 Gooster",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "2"},
					Name: "October 29 Gooster",
				},
			}, "October 28 Gooster", ontology.ID{Type: "test", Key: "1"}),
			Entry("Multi-word with shared prefix", []ontology.Resource{
				{
					ID:   ontology.ID{Type: "test", Key: "1"},
					Name: "View A",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "2"},
					Name: "View B",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "3"},
					Name: "View C",
				},
			}, "View A", ontology.ID{Type: "test", Key: "1"}),
		)
		DescribeTable("No Results",
			func(ctx SpecContext, res ontology.Resource, term string) {
				Expect(idx.IndexResources([]ontology.Resource{res})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				})).To(BeEmpty())
			},
			Entry("No Match", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "nope"),
			Entry("Partial No Match", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "Channel",
			}, "nn"),
		)
		Describe("Multiple Fields", func() {
			It("Should match on extra searchable fields", func(ctx SpecContext) {
				idx = newIndex(&mockService{
					resourceType: "device",
					fields:       []string{"make", "model"},
				})
				Expect(idx.IndexResources([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "device", Key: "1"},
						Name: "My Device",
						Data: map[string]any{"make": "LabJack", "model": "T7"},
					},
				})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "device",
					Term: "LabJack",
				}))
				Expect(res).To(HaveLen(1))
				Expect(res[0].Key).To(Equal("1"))
			})
			It("Should match on name when extra fields are registered", func(ctx SpecContext) {
				idx = newIndex(&mockService{
					resourceType: "device",
					fields:       []string{"make", "model"},
				})
				Expect(idx.IndexResources([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "device", Key: "1"},
						Name: "My Device",
						Data: map[string]any{"make": "LabJack", "model": "T7"},
					},
				})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "device",
					Term: "My Device",
				}))
				Expect(res).To(HaveLen(1))
			})
			It("Should prioritize exact match across multiple types", func(ctx SpecContext) {
				idx = newIndex(
					&mockService{resourceType: "device", fields: []string{"make"}},
					&mockService{resourceType: "channel"},
				)
				Expect(idx.IndexResources([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "device", Key: "1"},
						Name: "Pressure Sensor",
						Data: map[string]any{"make": "NI"},
					},
					{
						ID:   ontology.ID{Type: "channel", Key: "2"},
						Name: "Pressure",
					},
				})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Term: "Pressure Sensor",
				}))
				Expect(res).ToNot(BeEmpty())
				Expect(res[0].Key).To(Equal("1"))
			})
			It("Should find results by searching extra fields across types", func(ctx SpecContext) {
				idx = newIndex(
					&mockService{resourceType: "device", fields: []string{"make"}},
					&mockService{resourceType: "channel"},
				)
				Expect(idx.IndexResources([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "device", Key: "1"},
						Name: "My Device",
						Data: map[string]any{"make": "LabJack"},
					},
					{
						ID:   ontology.ID{Type: "channel", Key: "2"},
						Name: "Temperature",
					},
				})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Term: "LabJack",
				}))
				Expect(res).To(HaveLen(1))
				Expect(res[0].Key).To(Equal("1"))
			})
		})
		Describe("Disjunction Fallback", func() {
			It("Should fall back to disjunction if conjunction finds no results", func(ctx SpecContext) {
				Expect(idx.IndexResources([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "test", Key: "1"},
						Name: "My Blob",
					},
				})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: "My Blog",
				})).To(Not(BeEmpty()))
			})
			It("Should not fall back to disjunction if conjunction finds results", func(ctx SpecContext) {
				Expect(idx.IndexResources([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "test", Key: "1"},
						Name: "gse_ai_12",
					},
					{
						ID:   ontology.ID{Type: "test", Key: "2"},
						Name: "gse_doa_1",
					},
				})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: "gse_ai_12",
				})).To(HaveLen(1))
			})
		})
	})
	DescribeTable("Custom Tokenizer",
		func(input string, expected []string) {
			tk := &search.SepTokenizer{}
			tok := tk.Tokenize([]byte(input))
			Expect(tok).To(HaveLen(len(expected)))
			for i, term := range expected {
				Expect(tok[i].Term).To(Equal([]byte(term)))
			}
		},
		Entry("Single Word", "test", []string{"test"}),
		Entry("Two Words", "test test", []string{"test", "test"}),
		Entry("Two Words with Underscore", "test_test", []string{"test", "test"}),
		Entry("Scream Case", "TEST", []string{"TEST"}),
		Entry("Scream Case with Underscore", "TEST_TEST", []string{"TEST", "TEST"}),
		Entry("Scream Case with Space", "TEST TEST", []string{"TEST", "TEST"}),
	)
	Describe("Initialize", func() {
		It("Should index existing resources from OpenNexter", func(ctx SpecContext) {
			svc := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "widget",
				resources: []ontology.Resource{
					{ID: ontology.ID{Type: "widget", Key: "1"}, Name: "Alpha Widget"},
					{ID: ontology.ID{Type: "widget", Key: "2"}, Name: "Beta Widget"},
					{ID: ontology.ID{Type: "widget", Key: "3"}, Name: "Gamma Gadget"},
				},
			}
			idx := MustSucceed(search.Open())
			idx.RegisterService(svc)
			Expect(idx.Initialize(ctx)).To(Succeed())
			res := MustSucceed(idx.Search(ctx, search.Request{
				Type: "widget",
				Term: "Alpha",
			}))
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal("1"))
		})

		It("Should make all startup resources searchable", func(ctx SpecContext) {
			svc := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "item",
				resources: []ontology.Resource{
					{ID: ontology.ID{Type: "item", Key: "a"}, Name: "Pressure Sensor"},
					{ID: ontology.ID{Type: "item", Key: "b"}, Name: "Temperature Probe"},
				},
			}
			idx := MustSucceed(search.Open())
			idx.RegisterService(svc)
			Expect(idx.Initialize(ctx)).To(Succeed())
			res := MustSucceed(idx.Search(ctx, search.Request{Term: "Temperature"}))
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal("b"))
		})
		It("Should index multiple services", func(ctx SpecContext) {
			svc1 := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "channel",
				resources: []ontology.Resource{
					{ID: ontology.ID{Type: "channel", Key: "ch1"}, Name: "gse_ai_1"},
				},
			}
			svc2 := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "device",
				resources: []ontology.Resource{
					{ID: ontology.ID{Type: "device", Key: "d1"}, Name: "LabJack T7"},
				},
			}
			idx := MustSucceed(search.Open())
			idx.RegisterService(svc1)
			idx.RegisterService(svc2)
			Expect(idx.Initialize(ctx)).To(Succeed())
			chRes := MustSucceed(idx.Search(ctx, search.Request{Term: "gse_ai"}))
			Expect(chRes).To(HaveLen(1))
			Expect(chRes[0].Key).To(Equal("ch1"))
			devRes := MustSucceed(idx.Search(ctx, search.Request{Term: "LabJack"}))
			Expect(devRes).To(HaveLen(1))
			Expect(devRes[0].Key).To(Equal("d1"))
		})
		It("Should index extra fields from FieldsProvider", func(ctx SpecContext) {
			svc := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "device",
				fields:       []string{"make", "model"},
				resources: []ontology.Resource{
					{
						ID:   ontology.ID{Type: "device", Key: "1"},
						Name: "My Device",
						Data: map[string]any{"make": "NI", "model": "cDAQ-9178"},
					},
				},
			}
			idx := MustSucceed(search.Open())
			idx.RegisterService(svc)
			Expect(idx.Initialize(ctx)).To(Succeed())
			res := MustSucceed(idx.Search(ctx, search.Request{Term: "cDAQ"}))
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal("1"))
		})
		It("Should apply live changes via OnChange", func(ctx SpecContext) {
			svc := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "task",
				resources:    []ontology.Resource{},
			}
			idx := MustSucceed(search.Open())
			idx.RegisterService(svc)
			Expect(idx.Initialize(ctx)).To(Succeed())
			Expect(idx.Search(ctx, search.Request{Term: "MyTask"})).To(BeEmpty())
			svc.Notify(ctx, slices.Values([]ontology.Change{
				{
					Variant: change.VariantSet,
					Key:     "task:t1",
					Value: ontology.Resource{
						ID:   ontology.ID{Type: "task", Key: "t1"},
						Name: "MyTask",
					},
				},
			}))
			res := MustSucceed(idx.Search(ctx, search.Request{Term: "MyTask"}))
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal("t1"))
		})
		It("Should remove resources via OnChange delete", func(ctx SpecContext) {
			svc := &observableMockService{
				Observer:     observe.New[iter.Seq[ontology.Change]](),
				resourceType: "task",
				resources: []ontology.Resource{
					{ID: ontology.ID{Type: "task", Key: "t1"}, Name: "DeleteMe"},
				},
			}
			idx := MustSucceed(search.Open())
			idx.RegisterService(svc)
			Expect(idx.Initialize(ctx)).To(Succeed())
			Expect(idx.Search(ctx, search.Request{Term: "DeleteMe"})).To(HaveLen(1))
			svc.Notify(ctx, slices.Values([]ontology.Change{
				{
					Variant: change.VariantDelete,
					Key:     "task:t1",
				},
			}))
			Expect(idx.Search(ctx, search.Request{Term: "DeleteMe"})).To(BeEmpty())
		})
	})
})
