package stat_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ctx = context.Background()

func TestStat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stat Suite")
}