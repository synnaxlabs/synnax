package errutil_test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errutil"
)

var _ = Describe("Is", func() {
	DescribeTable("IsAny", func(err error, errs []error, expected bool) {
		Expect(errutil.IsAny(err, errs...)).To(BeEquivalentTo(expected))
	},
		Entry("Should return false if no errors are given", fmt.Errorf("test"), []error{}, false),
		Entry("Should return false if no errors are the same as the given error", fmt.Errorf("test"), []error{fmt.Errorf("test1"), fmt.Errorf("test2")}, false),
		Entry("Should return true if any of the errors are the same as the given error", fmt.Errorf("test"), []error{fmt.Errorf("test1"), fmt.Errorf("test")}, true),
	)
})
