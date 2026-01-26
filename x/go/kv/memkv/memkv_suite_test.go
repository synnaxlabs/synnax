package memkv_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMemKV(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mem KV Suite")
}
