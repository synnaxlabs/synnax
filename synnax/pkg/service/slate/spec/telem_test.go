package spec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/validate"
)

func singleNodeGraph(node spec.Node) spec.Graph {
	return spec.Graph{
		Nodes: []spec.Node{
			{
				Key:    "1",
				Type:   spec.TelemSourceType,
				Config: map[string]interface{}{"channel": 0},
			},
		},
	}
}

var _ = Describe("Telem", func() {
	Describe("Source", func() {
		It("Should return a validation error if the channel field is not specified", func() {
			g := singleNodeGraph(spec.Node{
				Key:    "1",
				Type:   spec.TelemSourceType,
				Config: map[string]interface{}{},
			})
			_, err := spec.Validate(ctx, spec.Config{}, g)
			Expect(err).To(HaveOccurred())
			Expect(err).To(validate.ContainPath([]string{"1", "channel"}))
			Expect(err).To(MatchError(ContainSubstring("required")))
		})

		It("Should return a validation error if the channel field is zero", func() {
			g := singleNodeGraph(spec.Node{
				Key:    "1",
				Type:   spec.TelemSourceType,
				Config: map[string]interface{}{},
			})
			_, err := spec.Validate(ctx, spec.Config{}, g)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("required")))
		})
	})

})
