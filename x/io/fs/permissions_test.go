package fs_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs"
	"os"
)

var _ = Describe("Permissions", func() {
	Describe("CheckSufficientPermissions", func() {
		DescribeTable("should return the correct value",
			func(actual, expected os.FileMode, output bool) {
				Expect(fs.CheckSufficientPermissions(actual, expected)).To(Equal(output))
			},
			Entry("0755 700", os.FileMode(0755), fs.OS_USER_RWX, true),
			Entry("600 700", os.FileMode(600), fs.OS_USER_RWX, false),
		)

	})
})
