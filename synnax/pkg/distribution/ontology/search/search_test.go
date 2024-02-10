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
				//idx.Register(ctx, schema.Schema{
				//	Type: "test",
				//})
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
