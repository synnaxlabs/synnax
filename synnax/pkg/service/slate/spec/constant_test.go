package spec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Constant", func() {
	DescribeTable("Validation", func(n spec.Node, expected error) {
		g := spec.Graph{Nodes: []spec.Node{n}}
		_, err := spec.Validate(ctx, spec.Config{}, g)
		if expected == nil {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurredAs(expected))
		}
	},
		Entry("Basic", spec.Node{
			Type: spec.ConstantType,
			Data: map[string]any{
				"data_type": "float32",
				"value":     12,
			},
		}, nil),
		Entry("Missing Data Type", spec.Node{
			Type: spec.ConstantType,
			Data: map[string]any{
				"value": "dog",
			},
		}, validate.FieldError{
			Field:   "data_type",
			Message: "invalid data type",
		}),
		Entry("Missing Value", spec.Node{
			Type: spec.ConstantType,
			Data: map[string]any{
				"data_type": "dog",
			},
		}, validate.FieldError{
			Field:   "value",
			Message: "invalid value",
		}),
	)
})
