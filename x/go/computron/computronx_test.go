package computron_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/computronx"
	"github.com/synnaxlabs/x/telem"
)

func TestComputronX(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ComputronX Suite")
}

var _ = Describe("Lua ComputronX Operations", Ordered, func() {
	var (
		interpreter *computronx.Interpreter
	)

	BeforeEach(func() {
		var err error
		interpreter, err = computronx.New()
		Expect(err).ToNot(HaveOccurred())
	})

	DescribeTable("Basic Arithmetic",
		func(expression string, expected float64) {
			calc, err := interpreter.NewCalculation(expression)
			Expect(err).ToNot(HaveOccurred())

			result, err := calc.Run(nil)
			Expect(err).ToNot(HaveOccurred())

			value := telem.ValueAt[float64](result, 0)
			Expect(value).To(Equal(expected))
		},
		Entry("Addition", "2 + 3", 5.0),
		Entry("Subtraction", "5 - 3", 2.0),
		Entry("Multiplication", "4 * 3", 12.0),
		Entry("Division", "10 / 2", 5.0),
		Entry("Modulo", "7 % 3", 1.0),
		Entry("Complex", "(2 + 3) * 4", 20.0),
		Entry("Exponentiation", "2 ^ 3", 8.0),
		Entry("Parentheses Priority", "2 * (3 + 4)", 14.0),
		Entry("Floating-Point Division", "7 / 2", 3.5),
	)

	Describe("Variable Handling", func() {
		It("Should handle simple variable substitution", func() {
			calc, err := interpreter.NewCalculation("x + y")
			Expect(err).ToNot(HaveOccurred())

			vars := map[string]interface{}{
				"x": telem.NewSeriesV[float64](5.0),
				"y": telem.NewSeriesV[float64](3.0),
			}

			result, err := calc.Run(vars)
			Expect(err).ToNot(HaveOccurred())

			value := telem.ValueAt[float64](result, 0)
			Expect(value).To(Equal(8.0))
		})

		It("Should handle complex variable expressions", func() {
			calc, err := interpreter.NewCalculation("(x + y) * (z - x)")
			Expect(err).ToNot(HaveOccurred())

			vars := map[string]interface{}{
				"x": telem.NewSeriesV[float64](2.0),
				"y": telem.NewSeriesV[float64](3.0),
				"z": telem.NewSeriesV[float64](7.0),
			}

			result, err := calc.Run(vars)
			Expect(err).ToNot(HaveOccurred())

			value := telem.ValueAt[float64](result, 0)
			Expect(value).To(Equal(25.0))
		})
	})

	Describe("Error Handling", func() {
		It("Should handle malformed expressions", func() {
			_, err := interpreter.NewCalculation("2 + (3 *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid Lua syntax"))
		})

		It("Should handle syntax errors", func() {
			_, err := interpreter.NewCalculation("2 +")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid Lua syntax"))
		})
	})
})
