package confluence_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/confluence"
)

var _ = Describe("Stream", func() {
	var (
		addr address.Address = "addr"
	)
	Describe("internal Stream", func() {
		Describe("Address", func() {
			Context("Stream", func() {
				It("Should set the inlet address properly", func() {
					stream := NewStream[int](0)
					stream.SetInletAddress(addr)
					Expect(stream.InletAddress()).To(Equal(addr))
				})
				It("Should set the outlet address properly", func() {
					stream := NewStream[int](0)
					stream.SetOutletAddress(addr)
					Expect(stream.OutletAddress()).To(Equal(addr))
				})
			})

		})
		Describe("Outlet", func() {
			var (
				ch     = make(chan int, 1)
				outlet = NewOutlet(ch)
			)
			It("Should set the address properly", func() {
				outlet.SetOutletAddress(addr)
				Expect(outlet.OutletAddress()).To(Equal(addr))
			})
			It("Should return the correct channel when calling Outlet", func() {
				ch <- 1
				Expect(<-outlet.Outlet()).To(Equal(1))
			})
		})
		Describe("Inlet", func() {
			var (
				ch    = make(chan int, 1)
				inlet = NewInlet(ch)
			)
			It("Should set the address properly", func() {
				inlet.SetInletAddress(addr)
				Expect(inlet.InletAddress()).To(Equal(addr))
			})
			It("Should return the correct channel when calling Inlet", func() {
				inlet.Inlet() <- 1
				Expect(<-ch).To(Equal(1))
			})
			It("Should close the wrapped channel", func() {
				inlet.Close()
			})
		})
	})
})
