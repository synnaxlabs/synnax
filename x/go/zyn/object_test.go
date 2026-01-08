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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Object", func() {
	Describe("Basic Parsing", func() {
		Specify("empty object schema", func() {
			schema := zyn.Object(nil)
			data := map[string]any{}
			var dest struct{}
			Expect(schema.Parse(data, &dest)).To(Succeed())
		})
		Specify("valid object", func() {
			type TestStruct struct {
				Name  string
				Age   int
				Score float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})
			data := map[string]any{"Name": "John", "Age": 42, "Score": 95.5}
			var dest TestStruct
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.Name).To(Equal("John"))
			Expect(dest.Age).To(Equal(42))
			Expect(dest.Score).To(Equal(95.5))
		})
		Specify("nested object", func() {
			type Address struct {
				Street string
				City   string
			}
			type Person struct {
				Name    string
				Address Address
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name": zyn.String(),
				"Address": zyn.Object(map[string]zyn.Schema{
					"Street": zyn.String(),
					"City":   zyn.String(),
				}),
			})
			data := map[string]any{
				"Name":    "John",
				"Address": map[string]any{"Street": "123 Main St", "City": "Boston"},
			}
			var dest Person
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.Name).To(Equal("John"))
			Expect(dest.Address.Street).To(Equal("123 Main St"))
			Expect(dest.Address.City).To(Equal("Boston"))
		})
	})
	Describe("Validate", func() {
		It("Should return nil if the value is a valid object", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			Expect(schema.Validate(map[string]any{"Name": "John"})).To(Succeed())
		})
		It("Should return nil if the value is not a valid object", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			Expect(schema.Validate("not an object")).To(HaveOccurred())
		})
	})
	Describe("Invalid Inputs", func() {
		Specify("non-map data", func() {
			type TestStruct struct{ Name string }
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest TestStruct
			Expect(schema.Parse("not a map", &dest)).
				To(And(
					HaveOccurredAs(zyn.InvalidDestinationTypeError),
					MatchError(ContainSubstring("received string")),
				))
		})
		Specify("nil pointer", func() {
			type TestStruct struct{ Name string }
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest *TestStruct
			Expect(schema.Parse(map[string]any{"Name": "John"}, dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("non-pointer destination", func() {
			type TestStruct struct{ Name string }
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest TestStruct
			Expect(schema.Parse(map[string]any{"Name": "John"}, dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("missing required field", func() {
			type TestStruct struct {
				Name string
				Age  int
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name": zyn.String(),
				"Age":  zyn.Number(),
			})
			var dest TestStruct
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(MatchError(ContainSubstring("required")))
		})
		Specify("string destination", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest string
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("numeric destination", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest int
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("bool destination", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest bool
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("slice destination", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest []string
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("map destination", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest map[string]string
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
		Specify("channel destination", func() {
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest chan struct{}
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).
				To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
	})
	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			type TestStruct struct {
				Email *string
				Name  string
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name":  zyn.String(),
				"Email": zyn.String().Optional(),
			})
			data := map[string]any{"Name": "John"}
			var dest TestStruct
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.Name).To(Equal("John"))
			Expect(dest.Email).To(BeNil())
		})
		Specify("required field with nil value", func() {
			type TestStruct struct{ Name string }
			schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
			var dest TestStruct
			Expect(schema.Parse(nil, &dest)).
				To(MatchError(ContainSubstring("required")))
		})
	})
	Describe("Dump", func() {
		Specify("basic object", func() {
			type TestStruct struct {
				Name  string
				Age   int
				Score float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})
			data := TestStruct{Name: "John", Age: 42, Score: 95.5}
			Expect(schema.Dump(data)).To(Equal(map[string]any{
				"name":  "John",
				"age":   int64(42),
				"score": 95.5,
			}))
		})
		Specify("nested object", func() {
			type Address struct {
				Street string
				City   string
			}
			type Person struct {
				Name    string
				Address Address
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name": zyn.String(),
				"Address": zyn.Object(map[string]zyn.Schema{
					"Street": zyn.String(),
					"City":   zyn.String(),
				}),
			})
			data := Person{
				Name:    "John",
				Address: Address{Street: "123 Main St", City: "Boston"},
			}
			Expect(schema.Dump(data)).To(Equal(map[string]any{
				"name":    "John",
				"address": map[string]any{"street": "123 Main St", "city": "Boston"},
			}))
		})
		Specify("optional fields", func() {
			type TestStruct struct {
				Email *string
				Name  string
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name":  zyn.String(),
				"Email": zyn.String().Optional(),
			})
			data := TestStruct{Name: "John", Email: nil}
			Expect(schema.Dump(data)).To(Equal(map[string]any{"name": "John"}))
		})
		Specify("already dumped map[string]any", func() {
			type TestStruct struct {
				Name  string
				Age   int
				Score float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})
			// First dump
			data := TestStruct{Name: "John", Age: 42, Score: 95.5}
			firstDump := MustSucceed(schema.Dump(data))
			Expect(firstDump).To(Equal(map[string]any{
				"name":  "John",
				"age":   int64(42),
				"score": 95.5,
			}))
			Expect(schema.Dump(firstDump)).To(Equal(map[string]any{
				"name":  "John",
				"age":   int64(42),
				"score": 95.5,
			}))
		})
		Specify("invalid map[string]any", func() {
			schema := zyn.Object(map[string]zyn.Schema{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})
			// Missing required field age
			data := map[string]any{"name": "John", "score": 95.5}
			Expect(schema.Dump(data)).Error().
				To(MatchError(ContainSubstring("required")))
		})
		Describe("Invalid Inputs", func() {
			Specify("nil value", func() {
				schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
				Expect(schema.Dump(nil)).Error().
					To(MatchError(ContainSubstring("required")))
			})
			Specify("nil pointer", func() {
				type TestStruct struct{ Name string }
				schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
				var data *TestStruct
				Expect(schema.Dump(data)).Error().
					To(MatchError(ContainSubstring("required")))
			})
			Specify("optional nil value", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
				}).Optional()
				Expect(schema.Dump(nil)).To(BeNil())
			})
			Specify("optional nil pointer", func() {
				type TestStruct struct{ Name string }
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
				}).Optional()
				var data *TestStruct
				Expect(schema.Dump(data)).To(BeNil())
			})
			Specify("non-struct value", func() {
				schema := zyn.Object(map[string]zyn.Schema{"Name": zyn.String()})
				Expect(schema.Dump("not a struct")).Error().
					To(MatchError(ContainSubstring(
						"expected struct or map[string]any",
					)))
			})
			Specify("missing required field", func() {
				type TestStruct struct{ Name string }
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
					"Age":  zyn.Number(),
				})
				data := TestStruct{Name: "John"}
				Expect(schema.Dump(data)).Error().
					To(MatchError(ContainSubstring("required")))
			})
		})
		Describe("Map Input", func() {
			Specify("valid map with snake case keys", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name":  zyn.String(),
					"Age":   zyn.Number(),
					"Score": zyn.Number(),
				})
				data := map[string]any{"name": "John", "age": 42, "score": 95.5}
				Expect(schema.Dump(data)).To(Equal(map[string]any{
					"name":  "John",
					"age":   int64(42),
					"score": 95.5,
				}))
			})
			Specify("valid map with mixed case keys", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name":  zyn.String(),
					"Age":   zyn.Number(),
					"Score": zyn.Number(),
				})
				data := map[string]any{"name": "John", "age": 42, "Score": 95.5}
				Expect(schema.Dump(data)).To(Equal(map[string]any{
					"name":  "John",
					"age":   int64(42),
					"score": 95.5,
				}))
			})
			Specify("nested object map", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
					"Address": zyn.Object(map[string]zyn.Schema{
						"Street": zyn.String(),
						"City":   zyn.String(),
					}),
				})
				data := map[string]any{
					"name": "John",
					"address": map[string]any{
						"street": "123 Main St",
						"city":   "Boston",
					},
				}
				Expect(schema.Dump(data)).To(Equal(map[string]any{
					"name": "John",
					"address": map[string]any{
						"street": "123 Main St",
						"city":   "Boston",
					},
				}))
			})
			Specify("optional fields in map", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name":  zyn.String(),
					"Email": zyn.String().Optional(),
				})
				data := map[string]any{"name": "John"}
				Expect(schema.Dump(data)).To(Equal(map[string]any{"name": "John"}))
			})
			Specify("nil optional field in map", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name":  zyn.String(),
					"Email": zyn.String().Optional(),
				})
				data := map[string]any{"name": "John", "email": nil}
				Expect(schema.Dump(data)).To(Equal(map[string]any{"name": "John"}))
			})
			Specify("invalid field type in map", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
					"Age":  zyn.Number(),
				})
				data := map[string]any{"name": "John", "age": "not a number"}
				Expect(schema.Dump(data)).Error().
					To(MatchError(ContainSubstring(
						"expected number or convertible to number",
					)))
			})
			Specify("missing required field in map", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
					"Age":  zyn.Number(),
				})
				data := map[string]any{"name": "John"}
				Expect(schema.Dump(data)).Error().
					To(MatchError(ContainSubstring("required")))
			})
			Specify("invalid nested object in map", func() {
				schema := zyn.Object(map[string]zyn.Schema{
					"Name": zyn.String(),
					"Address": zyn.Object(map[string]zyn.Schema{
						"Street": zyn.String(),
						"City":   zyn.String(),
					}),
				})
				data := map[string]any{"name": "John", "address": "not an object"}
				Expect(schema.Dump(data)).Error().
					To(MatchError(ContainSubstring(
						"expected struct or map[string]any",
					)))
			})
		})
	})
	Describe("Case Conversion", func() {
		Specify("dumps to snake case", func() {
			type TestStruct struct {
				FirstName string
				LastName  string
				Age       int
				Score     float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"FirstName": zyn.String(),
				"LastName":  zyn.String(),
				"Age":       zyn.Number(),
				"Score":     zyn.Number(),
			})
			data := TestStruct{FirstName: "John", LastName: "Doe", Age: 42, Score: 95.5}
			Expect(schema.Dump(data)).To(Equal(map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        int64(42),
				"score":      95.5,
			}))
		})
		Specify("parses from snake case", func() {
			type TestStruct struct {
				FirstName string
				LastName  string
				Age       int
				Score     float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"FirstName": zyn.String(),
				"LastName":  zyn.String(),
				"Age":       zyn.Number(),
				"Score":     zyn.Number(),
			})
			data := map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        42,
				"score":      95.5,
			}
			var dest TestStruct
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Age).To(Equal(42))
			Expect(dest.Score).To(Equal(95.5))
		})
		Specify("parses from mixed case", func() {
			type TestStruct struct {
				FirstName string
				LastName  string
				Age       int
				Score     float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"FirstName": zyn.String(),
				"LastName":  zyn.String(),
				"Age":       zyn.Number(),
				"Score":     zyn.Number(),
			})
			data := map[string]any{
				"FirstName": "John",
				"last_name": "Doe",
				"Age":       42,
				"score":     95.5,
			}
			var dest TestStruct
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Age).To(Equal(42))
			Expect(dest.Score).To(Equal(95.5))
		})
		Specify("nested object case conversion", func() {
			type Address struct {
				StreetName string
				CityName   string
			}
			type Person struct {
				FirstName string
				LastName  string
				Address   Address
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"FirstName": zyn.String(),
				"LastName":  zyn.String(),
				"Address": zyn.Object(map[string]zyn.Schema{
					"StreetName": zyn.String(),
					"CityName":   zyn.String(),
				}),
			})
			data := Person{
				FirstName: "John",
				LastName:  "Doe",
				Address:   Address{StreetName: "123 Main St", CityName: "Boston"},
			}
			result := MustSucceed(schema.Dump(data))
			Expect(result).To(Equal(map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"address": map[string]any{
					"street_name": "123 Main St",
					"city_name":   "Boston",
				},
			}))
			// Test parsing back
			var dest Person
			Expect(schema.Parse(result, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Address.StreetName).To(Equal("123 Main St"))
			Expect(dest.Address.CityName).To(Equal("Boston"))
		})
	})
	Describe("Schema Keys", func() {
		Specify("accepts snake case keys in schema", func() {
			type TestStruct struct {
				FirstName string
				LastName  string
				Age       int
				Score     float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"first_name": zyn.String(),
				"last_name":  zyn.String(),
				"age":        zyn.Number(),
				"score":      zyn.Number(),
			})
			data := TestStruct{FirstName: "John", LastName: "Doe", Age: 42, Score: 95.5}
			result := MustSucceed(schema.Dump(data))
			Expect(result).To(Equal(map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        int64(42),
				"score":      95.5,
			}))
			// Test parsing back
			var dest TestStruct
			Expect(schema.Parse(result, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Age).To(Equal(42))
			Expect(dest.Score).To(Equal(95.5))
		})
		Specify("accepts mixed case keys in schema", func() {
			type TestStruct struct {
				FirstName string
				LastName  string
				Age       int
				Score     float64
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"FirstName": zyn.String(),
				"last_name": zyn.String(),
				"Age":       zyn.Number(),
				"score":     zyn.Number(),
			})
			data := TestStruct{FirstName: "John", LastName: "Doe", Age: 42, Score: 95.5}
			result := MustSucceed(schema.Dump(data))
			Expect(result).To(Equal(map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        int64(42),
				"score":      95.5,
			}))
			// Test parsing back
			var dest TestStruct
			Expect(schema.Parse(result, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Age).To(Equal(42))
			Expect(dest.Score).To(Equal(95.5))
		})
		Specify("nested object with snake case keys", func() {
			type Address struct {
				StreetName string
				CityName   string
			}
			type Person struct {
				FirstName string
				LastName  string
				Address   Address
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"first_name": zyn.String(),
				"last_name":  zyn.String(),
				"address": zyn.Object(map[string]zyn.Schema{
					"street_name": zyn.String(),
					"city_name":   zyn.String(),
				}),
			})
			data := Person{
				FirstName: "John",
				LastName:  "Doe",
				Address:   Address{StreetName: "123 Main St", CityName: "Boston"},
			}
			result := MustSucceed(schema.Dump(data))
			Expect(result).To(Equal(map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"address": map[string]any{
					"street_name": "123 Main St",
					"city_name":   "Boston",
				},
			}))
			// Test parsing back
			var dest Person
			Expect(schema.Parse(result, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Address.StreetName).To(Equal("123 Main St"))
			Expect(dest.Address.CityName).To(Equal("Boston"))
		})
	})
	Describe("Regression", func() {
		Describe("UUID Object", func() {
			It("Should parse correctly", func() {
				type MyStruct struct{ Value uuid.UUID }
				var schema = zyn.Object(map[string]zyn.Schema{"value": zyn.UUID()})
				value := uuid.New()
				data := map[string]any{"value": value.String()}
				var res MyStruct
				Expect(schema.Parse(data, &res)).To(Succeed())
				Expect(res).To(Equal(MyStruct{Value: value}))
			})
		})
	})
	Describe("Field Method", func() {
		Specify("building schema incrementally with Field()", func() {
			type TestStruct struct {
				FirstName string
				LastName  string
				Age       int
			}
			schema := zyn.Object(nil).
				Field("FirstName", zyn.String()).
				Field("LastName", zyn.String()).
				Field("Age", zyn.Number())
			data := TestStruct{FirstName: "John", LastName: "Doe", Age: 42}
			result := MustSucceed(schema.Dump(data))
			Expect(result).To(Equal(map[string]any{
				"first_name": "John",
				"last_name":  "Doe",
				"age":        int64(42),
			}))
			var dest TestStruct
			Expect(schema.Parse(result, &dest)).To(Succeed())
			Expect(dest.FirstName).To(Equal("John"))
			Expect(dest.LastName).To(Equal("Doe"))
			Expect(dest.Age).To(Equal(42))
		})
		Specify("combining Object() with Field()", func() {
			type TestStruct struct {
				Name  string
				Email string
				Age   int
			}
			schema := zyn.Object(map[string]zyn.Schema{
				"Name": zyn.String(),
			}).Field("Email", zyn.String()).Field("Age", zyn.Number())
			data := TestStruct{Name: "John", Email: "john@example.com", Age: 30}
			result := MustSucceed(schema.Dump(data))
			Expect(result).To(Equal(map[string]any{
				"name":  "John",
				"email": "john@example.com",
				"age":   int64(30),
			}))
			var dest TestStruct
			Expect(schema.Parse(result, &dest)).To(Succeed())
			Expect(dest.Name).To(Equal("John"))
			Expect(dest.Email).To(Equal("john@example.com"))
			Expect(dest.Age).To(Equal(30))
		})
	})
	Describe("Nested Object Field Errors", func() {
		It("Should correctly append path segments", func() {
			schema := zyn.Object(map[string]zyn.Schema{
				"first": zyn.Object(map[string]zyn.Schema{
					"second": zyn.Object(map[string]zyn.Schema{
						"third": zyn.Object(map[string]zyn.Schema{
							"value": zyn.Uint64(),
						}),
					}),
				}),
			})
			data := map[string]any{
				"first": map[string]any{
					"second": map[string]any{"third": map[string]any{"value": 123.2}},
				},
			}
			type MyStruct struct {
				First struct {
					Second struct{ Third struct{ Value uint64 } }
				}
			}
			var v MyStruct
			Expect(schema.Parse(data, &v)).
				To(MatchError(ContainSubstring("first.second.third.value")))
		})
	})
})
