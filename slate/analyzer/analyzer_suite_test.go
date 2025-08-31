package analyzer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAnalyzer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Analyzer Suite")
}
