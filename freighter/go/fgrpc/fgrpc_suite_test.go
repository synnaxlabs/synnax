package fgrpc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFGRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FGRPC Suite")
}
