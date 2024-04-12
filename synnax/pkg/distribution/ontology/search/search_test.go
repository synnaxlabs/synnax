// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Search", func() {
	Describe("Search", func() {
		var idx *search.Index
		BeforeEach(func() {
			idx = MustSucceed(search.New())
		})
		DescribeTable("Term Searching",
			func(resource schema.Resource, term string) {
				Expect(idx.Index([]schema.Resource{resource})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res).To(HaveLen(1))
			},
			Entry("Exact Match", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "test"),
			Entry("Word in Multi-Word Term", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "October 28 Gooster",
			}, "Gooster"),
			Entry("Near match to term", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "October 27 Gooster",
			}, "Gooster"),
			Entry("Underscores in term", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "gse_ai_15",
			}, "ai_15"),
			Entry("Captialization", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "BBTPC",
			}, "BTTPC"),
			Entry("Upper and lowercase", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "BBTPC",
			}, "bttpc"),
			Entry("Close Match in Multi-Word Term", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "BBTPC Sim",
			}, "BTTPC"),
			Entry("Partial Match Beginning", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "Channel",
			}, "ch"),
		)
		DescribeTable("Term Prioritization",
			func(resources []schema.Resource, term string, first ontology.ID) {
				Expect(idx.Index(resources)).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res[0].Key).To(Equal(first.Key))
			},
			Entry("Exact Match First", []schema.Resource{
				{
					ID:   ontology.ID{Type: "test", Key: "1"},
					Name: "test",
				},
				{
					ID:   ontology.ID{Type: "test", Key: "2"},
					Name: "test2",
				},
			}, "test", ontology.ID{Type: "test", Key: "1"}),
			Entry("Exact Match Multiple Words", []schema.Resource{
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
			func(resource schema.Resource, term string) {
				Expect(idx.Index([]schema.Resource{resource})).To(Succeed())
				res := MustSucceed(idx.Search(ctx, search.Request{
					Type: "test",
					Term: term,
				}))
				Expect(res).To(HaveLen(0))
			},
			Entry("No Match", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "test",
			}, "nope"),
			Entry("Multiple words no match", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "October 28 Gooster",
			}, "December Gooster"),
			Entry("Underscores no match", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "gse_ai_15",
			}, "ai_26"),
			Entry("Partial No Match", schema.Resource{
				ID:   ontology.ID{Type: "test", Key: "1"},
				Name: "Channel",
			}, "nn"),
		)
	})
})
