// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package override_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/override"
)

type myInterface interface {
	DoSomething()
}

type myInterfacePointerImpl struct{}

func (m *myInterfacePointerImpl) DoSomething() {}

type myInterfaceValueImpl struct{}

func (m myInterfaceValueImpl) DoSomething() {}

var _ = Describe("Override", func() {
	Describe("Numeric", func() {
		It("Should return the override value if it is not zero", func() {
			v := override.Numeric(0, 1)
			Expect(v).To(Equal(1))
		})
		It("Should return the value if the override value is zero", func() {
			v := override.Numeric(1, 0)
			Expect(v).To(Equal(1))
		})
	})
	Describe("String", func() {
		It("Should return the override value if it is not empty", func() {
			v := override.String("", "foo")
			Expect(v).To(Equal("foo"))
		})
		It("Should return the value if the override value is empty", func() {
			v := override.String("foo", "")
			Expect(v).To(Equal("foo"))
		})
	})
	Describe("Nil", func() {
		Context("Interface", func() {
			Context("Pointer", func() {
				It("Should return the override value if it is not nil", func() {
					v := override.Nil[myInterface](nil, &myInterfacePointerImpl{})
					Expect(v).ToNot(BeNil())
				})
			})
			Context("Value", func() {
				It("Should return the override value if it is not nil", func() {
					v := override.Nil[myInterface](nil, myInterfaceValueImpl{})
					Expect(v).ToNot(BeNil())
				})
			})
			Context("Nil", func() {
				It("Should return the value if the override value is nil", func() {
					v := override.Nil[myInterface](myInterfaceValueImpl{}, nil)
					Expect(v).ToNot(BeNil())
				})
			})
		})
		Context("Pointer", func() {
			It("Should return the override value if it is not nil", func() {
				v := override.Nil[*myInterfacePointerImpl](nil, &myInterfacePointerImpl{})
				Expect(v).ToNot(BeNil())
			})
			It("Should return the value if the override value is nil", func() {
				v := override.Nil[*myInterfacePointerImpl](&myInterfacePointerImpl{}, nil)
				Expect(v).ToNot(BeNil())
			})
		})
	})
})
