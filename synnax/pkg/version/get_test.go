package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/version"
	"strings"
)

var _ = Describe("Get", func() {
	Describe("Get", func() {
		It("Should return the version name", func() {
			v := version.Prod()
			cv := strings.TrimSpace(strings.ReplaceAll(v, "\n", ""))
			Expect(strings.Count(cv, ".")).To(Equal(2))
		})
	})
})
