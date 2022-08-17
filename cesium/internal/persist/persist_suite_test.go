package persist_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPersist(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Persist Suite")
}
