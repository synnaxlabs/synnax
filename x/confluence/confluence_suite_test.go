package confluence_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfluence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Confluence Suite")
}
