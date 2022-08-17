package fmock_test

import (
	"github.com/arya-analytics/freighter/fmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network", func() {
	var net *fmock.Network[int, int]
	BeforeEach(func() {
		net = fmock.NewNetwork[int, int]()
	})
	Describe("RouteUnary", func() {
		It("Should correctly add a new unary route", func() {
			t1 := net.RouteUnary("localhost:0")
			Expect(t1.String()).To(Equal("mock.Unary{} at localhost:0"))
		})
		It("Should auto generate an address if an empty string is provided", func() {
			t1 := net.RouteUnary("")
			Expect(t1.String()).To(Equal("mock.Unary{} at localhost:0"))
		})
	})
	Describe("RouteStream", func() {
		It("Should correctly add a new StreamTransport route", func() {
			t1 := net.RouteStream("localhost:0", 1)
			Expect(t1.String()).To(Equal("mock.StreamTransport{} at localhost:0"))
		})
	})
})
