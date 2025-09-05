package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("MapResolver", func() {
	Describe("Resolve", func() {
		It("Should resolve existing symbol", func() {
			resolver := symbol.MapResolver{
				"pi":    symbol.Symbol{Name: "pi", Kind: symbol.KindConfigParam, Type: types.F64{}},
				"count": symbol.Symbol{Name: "count", Kind: symbol.KindVariable, Type: types.I32{}},
			}
			sym, err := resolver.Resolve("pi")
			Expect(err).ToNot(HaveOccurred())
			Expect(sym.Name).To(Equal("pi"))
			Expect(sym.Kind).To(Equal(symbol.KindConfigParam))
			Expect(sym.Type).To(Equal(types.F64{}))
		})

		It("Should return error for non-existent symbol", func() {
			resolver := symbol.MapResolver{
				"x": symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32{}},
			}
			_, err := resolver.Resolve("y")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})

		It("Should work with empty resolver", func() {
			resolver := symbol.MapResolver{}
			_, err := resolver.Resolve("anything")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(query.NotFound))
		})
	})
})