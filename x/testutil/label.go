package testutil

import "github.com/onsi/ginkgo/v2"

const (
	Integration = "integration"
	Performance = "performance"
)

func LabelIntegration() ginkgo.Labels { return ginkgo.Label(Integration) }

func LabelPerformance() ginkgo.Labels { return ginkgo.Label(Performance) }
