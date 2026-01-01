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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("SearchTerm", func() {
	Describe("SearchTerm", func() {
		var idx *search.Index
		BeforeEach(func() {
			idx = MustSucceed(search.New())
			idx.Register(ctx, "test", zyn.Object(nil))
		})
		DescribeTable("SearchTerm Searching",
			func(resource ontology.Resource, term string) {
				Expect(idx.Index([]ontology.Resource{resource})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res).To(HaveLen(1))
			},
			Entry("Exact Match", ontology.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "test"),
			Entry("Word in Multi-Word SearchTerm", ontology.Resource{
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
			Entry("Close Match in Multi-Word SearchTerm", ontology.Resource{
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
		DescribeTable("SearchTerm Prioritization",
			func(resources []ontology.Resource, term string, first ontology.ID) {
				Expect(idx.Index(resources)).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(len(res)).To(BeNumerically(">", 0))
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
		)
		DescribeTable("No Results",
			func(resource ontology.Resource, term string) {
				Expect(idx.Index([]ontology.Resource{resource})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res).To(HaveLen(0))
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
		Describe("Disjunction Fallback", func() {
			It("Should fall back to a disjunction search if the conjunction search finds no results", func() {
				Expect(idx.Index([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "test", Key: "1"},
						Name: "My Blob",
					},
				})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: "My Blog",
				}))
				Expect(len(res)).To(BeNumerically(">", 0))
			})
			It("Should not fall back to a disjunction search if the conjunction search finds results", func() {
				Expect(idx.Index([]ontology.Resource{
					{
						ID:   ontology.ID{Type: "test", Key: "1"},
						Name: "gse_ai_12",
					},
					{
						ID:   ontology.ID{Type: "test", Key: "2"},
						Name: "gse_doa_1",
					},
				})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: "gse_ai_12",
				}))
				Expect(len(res)).To(Equal(1))
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
})
