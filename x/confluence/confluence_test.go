package confluence_test

import (
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Confluence", func() {

	Describe("EmptyFlow", func() {

		It("Should do nothing", func() {
			ctx, cancel := signal.TODO()
			defer cancel()
			Expect(func() {
				confluence.EmptyFlow{}.Flow(ctx)
			}).ToNot(Panic())
		})

	})

})
