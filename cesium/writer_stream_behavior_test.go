package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
)

var _ = Describe("Stream Writer Behavior", func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Sequence Numbering", func() {
		Specify("It should increment the sequence number correctly", func() {

		})
	})
})
