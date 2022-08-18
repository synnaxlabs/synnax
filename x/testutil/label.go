package testutil

import "github.com/onsi/ginkgo/v2"

var (
	Integration = ginkgo.Label("integration")
	Performance = ginkgo.Label("performance")
	Slow        = ginkgo.Label("slow")
)
