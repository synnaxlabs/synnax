// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Map", func() {
	Describe("Basic Parsing", func() {
		Specify("string to string map", func() {
			schema := zyn.Map(zyn.String(), zyn.String())
			data := map[string]any{"a": "1", "b": "2"}
			var dest map[string]string
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal(map[string]string{"a": "1", "b": "2"}))
		})
		Specify("string to number map", func() {
			schema := zyn.Map(zyn.String(), zyn.Number())
			data := map[string]any{"x": 1.5, "y": 2.5}
			var dest map[string]float64
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal(map[string]float64{"x": 1.5, "y": 2.5}))
		})
		Specify("string to bool map", func() {
			schema := zyn.Map(zyn.String(), zyn.Bool())
			data := map[string]any{"enabled": true, "visible": false}
			var dest map[string]bool
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal(map[string]bool{"enabled": true, "visible": false}))
		})
		Specify("empty map", func() {
			schema := zyn.Map(zyn.String(), zyn.String())
			data := map[string]any{}
			var dest map[string]string
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(HaveLen(0))
		})
		Specify("nested map of objects", func() {
			type Config struct {
				Value string
			}
			schema := zyn.Map(zyn.String(), zyn.Object(map[string]zyn.Schema{
				"Value": zyn.String(),
			}))
			data := map[string]any{
				"a": map[string]any{"Value": "first"},
				"b": map[string]any{"Value": "second"},
			}
			var dest map[string]Config
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest["a"].Value).To(Equal("first"))
			Expect(dest["b"].Value).To(Equal("second"))
		})
	})
	Describe("Validate", func() {
		It("Should succeed for a valid map", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Validate(map[string]any{"k": "v"})).To(Succeed())
		})
		It("Should fail for a non-map", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Validate("not a map")).To(HaveOccurred())
		})
		It("Should succeed for optional nil", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Optional().Validate(nil)).To(Succeed())
		})
		It("Should fail for required nil", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Validate(nil)).
				To(MatchError(validate.ErrRequired))
		})
		It("Should include key in path when value validation fails", func() {
			Expect(zyn.Map(zyn.String(), zyn.Uint32()).Validate(
				map[string]any{"bad": "not a number"},
			)).To(MatchError(ContainSubstring("bad")))
		})
		It("Should include key in path when key validation fails", func() {
			Expect(zyn.Map(zyn.Uint32(), zyn.String()).Validate(
				map[string]any{"abc": "val"},
			)).To(MatchError(ContainSubstring("abc")))
		})
	})
	Describe("Invalid Inputs", func() {
		Specify("non-map destination", func() {
			var dest string
			Expect(zyn.Map(zyn.String(), zyn.String()).Parse(map[string]any{"a": "1"}, &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("nil pointer destination", func() {
			var dest *map[string]string
			Expect(zyn.Map(zyn.String(), zyn.String()).Parse(map[string]any{"a": "1"}, dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("non-pointer destination", func() {
			dest := map[string]string{}
			Expect(zyn.Map(zyn.String(), zyn.String()).Parse(map[string]any{"a": "1"}, dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("non-map data", func() {
			var dest map[string]string
			Expect(zyn.Map(zyn.String(), zyn.String()).Parse("not a map", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("value validation error includes key in path", func() {
			var dest map[string]uint32
			Expect(zyn.Map(zyn.String(), zyn.Uint32().Coerce()).Parse(
				map[string]any{"bad": "not a number"}, &dest,
			)).To(MatchError(ContainSubstring("bad")))
		})
		Specify("key parse error includes key in path", func() {
			var dest map[uint32]string
			Expect(zyn.Map(zyn.Uint32().Coerce(), zyn.String()).Parse(
				map[string]any{"abc": "val"}, &dest,
			)).To(MatchError(ContainSubstring("abc")))
		})
	})
	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest map[string]string
			Expect(zyn.Map(zyn.String(), zyn.String()).Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})
		Specify("required field with nil value", func() {
			var dest map[string]string
			Expect(zyn.Map(zyn.String(), zyn.String()).Parse(nil, &dest)).
				To(MatchError(validate.ErrRequired))
		})
	})
	Describe("Dump", func() {
		Specify("basic string map", func() {
			result := MustSucceed(zyn.Map(zyn.String(), zyn.String()).Dump(
				map[string]string{"a": "1", "b": "2"},
			))
			Expect(result).To(Equal(map[string]any{"a": "1", "b": "2"}))
		})
		Specify("number value map", func() {
			result := MustSucceed(zyn.Map(zyn.String(), zyn.Number()).Dump(
				map[string]int{"x": 1, "y": 2},
			))
			Expect(result).To(Equal(map[string]any{"x": int64(1), "y": int64(2)}))
		})
		Specify("empty map", func() {
			result := MustSucceed(zyn.Map(zyn.String(), zyn.String()).Dump(map[string]string{}))
			Expect(result).To(Equal(map[string]any{}))
		})
		Specify("map of objects", func() {
			type Config struct {
				Value string
			}
			schema := zyn.Map(zyn.String(), zyn.Object(map[string]zyn.Schema{
				"Value": zyn.String(),
			}))
			result := MustSucceed(schema.Dump(
				map[string]Config{"a": {Value: "first"}, "b": {Value: "second"}},
			))
			Expect(result).To(Equal(map[string]any{
				"a": map[string]any{"value": "first"},
				"b": map[string]any{"value": "second"},
			}))
		})
		Specify("nil required", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Dump(nil)).Error().
				To(MatchError(validate.ErrRequired))
		})
		Specify("nil pointer", func() {
			var m *map[string]string
			Expect(zyn.Map(zyn.String(), zyn.String()).Dump(m)).Error().
				To(MatchError(validate.ErrRequired))
		})
		Specify("optional nil value", func() {
			result := MustSucceed(zyn.Map(zyn.String(), zyn.String()).Optional().Dump(nil))
			Expect(result).To(BeNil())
		})
		Specify("optional nil pointer", func() {
			var m *map[string]string
			result := MustSucceed(zyn.Map(zyn.String(), zyn.String()).Optional().Dump(m))
			Expect(result).To(BeNil())
		})
		Specify("non-map value", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Dump("not a map")).Error().
				To(MatchError(ContainSubstring("expected map")))
		})
		Specify("non-nil pointer is dereferenced", func() {
			m := map[string]string{"a": "1"}
			result := MustSucceed(zyn.Map(zyn.String(), zyn.String()).Dump(&m))
			Expect(result).To(Equal(map[string]any{"a": "1"}))
		})
		Specify("value dump error includes key in path", func() {
			Expect(zyn.Map(zyn.String(), zyn.Uint32().Coerce()).Dump(
				map[string]int{"bad": -1},
			)).Error().To(MatchError(ContainSubstring("bad")))
		})
		Specify("key dump error includes key in path", func() {
			Expect(zyn.Map(zyn.Number(), zyn.String()).Dump(
				map[string]string{"abc": "val"},
			)).Error().To(MatchError(ContainSubstring("abc")))
		})
		Specify("dumped key must be a string", func() {
			Expect(zyn.Map(zyn.Number(), zyn.String()).Dump(
				map[int]string{1: "val"},
			)).Error().To(MatchError(ContainSubstring("must serialize to string")))
		})
		Specify("round-trip parse then dump", func() {
			schema := zyn.Map(zyn.String(), zyn.String())
			var dest map[string]string
			Expect(schema.Parse(map[string]any{"a": "1", "b": "2"}, &dest)).To(Succeed())
			result := MustSucceed(schema.Dump(dest))
			Expect(result).To(Equal(map[string]any{"a": "1", "b": "2"}))
		})
	})
	Describe("Shape", func() {
		It("Should return a MapShape", func() {
			shape := zyn.Map(zyn.String(), zyn.Number()).Shape()
			Expect(shape.DataType()).To(Equal(zyn.MapT))
			Expect(shape.Optional()).To(BeFalse())
			Expect(shape.Fields()).To(BeNil())
			mapShape, ok := shape.(zyn.MapShape)
			Expect(ok).To(BeTrue())
			Expect(mapShape.Key().DataType()).To(Equal(zyn.StringT))
			Expect(mapShape.Value().DataType()).To(Equal(zyn.NumberT))
		})
		It("Should reflect optional", func() {
			Expect(zyn.Map(zyn.String(), zyn.String()).Optional().Shape().Optional()).To(BeTrue())
		})
	})
})
