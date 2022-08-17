package kfs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKfs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kfs Suite")
}
