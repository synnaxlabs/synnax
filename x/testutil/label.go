package testutil

import "github.com/onsi/ginkgo/v2"

var (
	Integration = ginkgo.Label("integration")
	Performance = ginkgo.Label("performance")
	Unit        = ginkgo.Label("unit")
	Slow        = ginkgo.Label("slow")
)
