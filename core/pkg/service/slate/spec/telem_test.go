package spec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

func singleNodeGraph(node Node) Graph {
	return Graph{
		Nodes: []Node{
			{
				Key:    "1",
				Type:   TelemSourceType,
				Config: map[string]interface{}{"channel": 0},
			},
		},
	}
}

var _ = Describe("Telem", func() {
	Describe("Source", func() {
		It("Should return a validation error if the channel field is not specified", func() {
			g := singleNodeGraph(Node{
				Key:    "1",
				Type:   TelemSourceType,
				Config: map[string]interface{}{},
			})
			_, err := Validate(ctx, Config{}, g)
			Expect(err).To(HaveOccurred())
			Expect(err).To(validate.ContainPath([]string{"1", "channel"}))
			Expect(err).To(MatchError(ContainSubstring("required")))
		})

		It("Should return a validation error if the channel field is zero", func() {
			g := singleNodeGraph(Node{
				Key:    "1",
				Type:   TelemSourceType,
				Config: map[string]interface{}{},
			})
			_, err := Validate(ctx, Config{}, g)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("required")))
		})
	})

})
