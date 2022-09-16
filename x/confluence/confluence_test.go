package confluence_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Confluence", func() {

	Describe("EmptyFlow", func() {

		It("Should do nothing", func() {
			ctx, cancel := signal.TODO()
			defer cancel()
			Expect(func() {
				EmptyFlow{}.Flow(ctx)
			}).ToNot(Panic())
		})

	})

})
