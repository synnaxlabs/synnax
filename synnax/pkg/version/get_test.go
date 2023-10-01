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
			returned_version := version.Get()
			clean_version := strings.TrimSpace(strings.ReplaceAll(returned_version, "\n", ""))
			Expect(strings.Count(clean_version, ".")).To(Equal(2))
		})
	})
})
