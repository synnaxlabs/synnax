// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/types"
)

type customNamed struct{}

func (c customNamed) CustomTypeName() string { return "CustomName" }

type regularType struct{}
type yEndingType struct{}

type box struct{}
type class struct{}
type bus struct{}
type catch struct{}

type testInterface interface {
	DoSomething()
}

type namedType struct{}

type impl struct{}

func (i impl) DoSomething() {}

var _ = Describe("Name", func() {
	Context("Name", func() {
		It("Should return the type name for a regular struct", func() {
			Expect(types.Name[regularType]()).To(Equal("regularType"))
		})

		It("Should return the custom name for a type implementing CustomTypeName", func() {
			Expect(types.Name[customNamed]()).To(Equal("CustomName"))
		})

		It("Should work with built-in types", func() {
			Expect(types.Name[string]()).To(Equal("string"))
			Expect(types.Name[int]()).To(Equal("int"))
		})
	})

	Context("PluralName", func() {
		It("Should convert y endings to ies", func() {
			Expect(types.PluralName[yEndingType]()).To(Equal("yEndingTypes"))
		})

		It("Should add es to words ending in s, x, z", func() {
			Expect(types.PluralName[box]()).To(Equal("boxes"))
			Expect(types.PluralName[class]()).To(Equal("classes"))
			Expect(types.PluralName[bus]()).To(Equal("buses"))
		})

		It("Should add es to words ending in ch, sh", func() {
			Expect(types.PluralName[catch]()).To(Equal("catches"))
		})

		It("Should add s to regular words", func() {
			Expect(types.PluralName[regularType]()).To(Equal("regularTypes"))
		})

		It("Should work with custom named types", func() {
			Expect(types.PluralName[customNamed]()).To(Equal("CustomNames"))
		})

		It("Should work with built-in types", func() {
			Expect(types.PluralName[string]()).To(Equal("strings"))
			Expect(types.PluralName[int]()).To(Equal("ints"))
		})
	})

	Context("ValueName", func() {
		It("Should handle pointer types", func() {
			Expect(types.ValueName(reflect.ValueOf((*string)(nil)))).To(Equal("*string (nil)"))
			Expect(types.ValueName(reflect.ValueOf((*namedType)(nil)))).To(Equal("*types_test.namedType (nil)"))
			s := "hello"
			Expect(types.ValueName(reflect.ValueOf(&s))).To(Equal("*string"))
		})

		It("Should handle slice types", func() {
			Expect(types.ValueName(reflect.ValueOf([]string(nil)))).To(Equal("[]string (nil)"))
			Expect(types.ValueName(reflect.ValueOf([]int(nil)))).To(Equal("[]int (nil)"))
			Expect(types.ValueName(reflect.ValueOf([]string{}))).To(Equal("[]string"))
			Expect(types.ValueName(reflect.ValueOf([]int{}))).To(Equal("[]int"))
		})

		It("Should handle array types", func() {
			Expect(types.ValueName(reflect.ValueOf([3]string{}))).To(Equal("[3]string"))
			Expect(types.ValueName(reflect.ValueOf([5]int{}))).To(Equal("[5]int"))
		})

		It("Should handle map types", func() {
			Expect(types.ValueName(reflect.ValueOf(map[string]int(nil)))).To(Equal("map[string]int (nil)"))
			Expect(types.ValueName(reflect.ValueOf(map[int]string(nil)))).To(Equal("map[int]string (nil)"))
			Expect(types.ValueName(reflect.ValueOf(map[string]int{}))).To(Equal("map[string]int"))
			Expect(types.ValueName(reflect.ValueOf(map[int]string{}))).To(Equal("map[int]string"))
		})

		It("Should handle channel types", func() {
			Expect(types.ValueName(reflect.ValueOf((chan string)(nil)))).To(Equal("chan string (nil)"))
			Expect(types.ValueName(reflect.ValueOf((chan<- string)(nil)))).To(Equal("chan<- string (nil)"))
			Expect(types.ValueName(reflect.ValueOf((<-chan string)(nil)))).To(Equal("<-chan string (nil)"))
			ch := make(chan string)
			Expect(types.ValueName(reflect.ValueOf(ch))).To(Equal("chan string"))
		})

		It("Should handle function types", func() {
			Expect(types.ValueName(reflect.ValueOf((func())(nil)))).To(Equal("func (nil)"))
			Expect(types.ValueName(reflect.ValueOf(func() {}))).To(Equal("func"))
		})

		It("Should handle send directional channels", func() {
			v := make(chan<- string)
			Expect(types.ValueName(reflect.ValueOf(v))).To(Equal("chan<- string"))
		})

		It("Should handle receive directional channels", func() {
			v := make(<-chan string)
			Expect(types.ValueName(reflect.ValueOf(v))).To(Equal("<-chan string"))
		})

		It("Should handle interface types", func() {
			var i testInterface = impl{}
			Expect(types.ValueName(reflect.ValueOf(i))).To(Equal("types_test.impl"))
			Expect(types.ValueName(reflect.ValueOf((interface{})(nil)))).To(Equal("nil"))
		})

		It("Should handle named types", func() {
			Expect(types.ValueName(reflect.ValueOf(namedType{}))).To(Equal("types_test.namedType"))
		})

		It("Should handle nested complex types", func() {
			Expect(types.ValueName(reflect.ValueOf([]*string(nil)))).To(Equal("[]*string (nil)"))
			Expect(types.ValueName(reflect.ValueOf(map[string][]int(nil)))).To(Equal("map[string][]int (nil)"))
			Expect(types.ValueName(reflect.ValueOf([][]string{}))).To(Equal("[][]string"))
			Expect(types.ValueName(reflect.ValueOf(make(chan []string)))).To(Equal("chan []string"))
		})
	})

	Context("PackageName", func() {
		It("Should extract package name from custom types", func() {
			Expect(types.PackageName(reflect.TypeOf(namedType{}))).To(Equal("types_test"))
			Expect(types.PackageName(reflect.TypeOf(customNamed{}))).To(Equal("types_test"))
		})
	})
})
