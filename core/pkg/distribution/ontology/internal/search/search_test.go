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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/internal/resource"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/internal/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SearchTerm", func() {
	Describe("SearchTerm", func() {
		var (
			idx *search.Index
			ctx context.Context
		)
		BeforeEach(func() {
			idx = MustSucceed(search.New())
			ctx = context.Background()
			idx.Register(ctx, "test")
		})
		DescribeTable("SearchTerm Searching",
			func(res resource.Resource, term string) {
				Expect(idx.Index([]resource.Resource{res})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				})).To(HaveLen(1))
			},
			Entry("Exact Match", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "test"),
			Entry("Word in Multi-Word SearchTerm", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "October 28 Gooster",
			}, "Gooster"),
			Entry("Near match to term", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "October 27 Gooster",
			}, "Gooster"),
			Entry("Underscores in term", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "gse_ai_15",
			}, "ai_15"),
			Entry("All Caps", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "BBTPC",
			}, "BTTPC"),
			Entry("Upper and lowercase", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "BBTPC",
			}, "bttpc"),
			Entry("Close Match in Multi-Word SearchTerm", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "BBTPC Sim",
			}, "BTTPC"),
			Entry("Partial Match Beginning", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "Channel",
			}, "ch"),
			Entry("Scream Case with Underscore Exact", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "DAQ_PT",
			}, "DAQ_PT"),
			Entry("Scream Case with Underscore Partial", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "DAQ_PT_1",
			}, "DAQ_PT"),
		)
		DescribeTable("SearchTerm Prioritization",
			func(resources []resource.Resource, term string, first resource.ID) {
				Expect(idx.Index(resources)).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res).To(Not(BeEmpty()))
				Expect(res[0].Key).To(Equal(first.Key))
			},
			Entry("Exact Match First", []resource.Resource{
				{
					ID:   resource.ID{Type: "test", Key: "1"},
					Name: "test",
				},
				{
					ID:   resource.ID{Type: "test", Key: "2"},
					Name: "test2",
				},
			}, "test", resource.ID{Type: "test", Key: "1"}),
			Entry("Exact Match Multiple Words", []resource.Resource{
				{
					ID:   resource.ID{Type: "test", Key: "3"},
					Name: "October 30 Gooster",
				},
				{
					ID:   resource.ID{Type: "test", Key: "4"},
					Name: "October 31 Gooster",
				},
				{
					ID:   resource.ID{Type: "test", Key: "1"},
					Name: "October 28 Gooster",
				},
				{
					ID:   resource.ID{Type: "test", Key: "2"},
					Name: "October 29 Gooster",
				},
			}, "October 28 Gooster", resource.ID{Type: "test", Key: "1"}),
			Entry("Multi-word with shared prefix", []resource.Resource{
				{
					ID:   resource.ID{Type: "test", Key: "1"},
					Name: "View A",
				},
				{
					ID:   resource.ID{Type: "test", Key: "2"},
					Name: "View B",
				},
				{
					ID:   resource.ID{Type: "test", Key: "3"},
					Name: "View C",
				},
			}, "View A", resource.ID{Type: "test", Key: "1"}),
		)
		DescribeTable("No Results",
			func(res resource.Resource, term string) {
				Expect(idx.Index([]resource.Resource{res})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				})).To(HaveLen(0))
			},
			Entry("No Match", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "nope"),
			Entry("Partial No Match", resource.Resource{
				ID:   resource.ID{Type: "test", Key: "1"},
				Name: "Channel",
			}, "nn"),
		)
		Describe("Disjunction Fallback", func() {
			It("Should fall back to a disjunction search if the conjunction search finds no results", func() {
				Expect(idx.Index([]resource.Resource{
					{
						ID:   resource.ID{Type: "test", Key: "1"},
						Name: "My Blob",
					},
				})).To(Succeed())
				Expect(idx.Search(ctx, search.Request{
					Type: "test",
					Term: "My Blog",
				})).To(Not(BeEmpty()))
			})
			It("Should not fall back to a disjunction search if the conjunction search finds results", func() {
				Expect(idx.Index([]resource.Resource{
					{
						ID:   resource.ID{Type: "test", Key: "1"},
						Name: "gse_ai_12",
					},
					{
						ID:   resource.ID{Type: "test", Key: "2"},
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
})
