// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Object", func() {
	Describe("Basic Parsing", func() {
		Specify("empty object schema", func() {
			type TestStruct struct{}

			schema := zyn.Object(nil)

			data := map[string]any{}

			var dest TestStruct
			Expect(schema.Parse(data, &dest)).To(Succeed())
		})

		Specify("valid object", func() {
			type TestStruct struct {
				Name  string
				Age   int
				Score float64
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})

			data := map[string]any{
				"Name":  "John",
				"Age":   42,
				"Score": 95.5,
			}

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

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
				"Address": zyn.Object(map[string]zyn.Z{
					"Street": zyn.String(),
					"City":   zyn.String(),
				}),
			})

			data := map[string]any{
				"Name": "John",
				"Address": map[string]any{
					"Street": "123 Main St",
					"City":   "Boston",
				},
			}

			var dest Person
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.Name).To(Equal("John"))
			Expect(dest.Address.Street).To(Equal("123 Main St"))
			Expect(dest.Address.City).To(Equal("Boston"))
		})
	})

	Describe("Invalid Inputs", func() {
		Specify("non-map data", func() {
			type TestStruct struct {
				Name string
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
			})

			var dest TestStruct
			Expect(schema.Parse("not a map", &dest)).To(MatchError(validate.FieldError{Message: "invalid type: expected map[string]any"}))
		})

		Specify("nil pointer", func() {
			type TestStruct struct {
				Name string
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
			})

			var dest *TestStruct
			Expect(schema.Parse(map[string]any{"Name": "John"}, dest)).To(HaveOccurredAs(zyn.InvalidDestError("object")))
		})

		Specify("non-pointer destination", func() {
			type TestStruct struct {
				Name string
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
			})

			var dest TestStruct
			Expect(schema.Parse(map[string]any{"Name": "John"}, dest)).To(HaveOccurredAs(zyn.InvalidDestError("object")))
		})

		Specify("missing required field", func() {
			type TestStruct struct {
				Name string
				Age  int
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
				"Age":  zyn.Number(),
			})

			var dest TestStruct
			Expect(schema.Parse(map[string]any{"Name": "John"}, &dest)).To(MatchError(validate.FieldError{
				Field:   "Age",
				Message: "missing required field",
			}))
		})
	})

	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			type TestStruct struct {
				Name  string
				Email *string
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name":  zyn.String(),
				"Email": zyn.String().Optional(),
			})

			data := map[string]any{
				"Name": "John",
			}

			var dest TestStruct
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest.Name).To(Equal("John"))
			Expect(dest.Email).To(BeNil())
		})

		Specify("required field with nil value", func() {
			type TestStruct struct {
				Name string
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
			})

			var dest TestStruct
			Expect(schema.Parse(nil, &dest)).To(MatchError(validate.FieldError{Message: "value is required but was nil"}))
		})
	})

	Describe("Dump", func() {
		Specify("basic object", func() {
			type TestStruct struct {
				Name  string
				Age   int
				Score float64
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})

			data := TestStruct{
				Name:  "John",
				Age:   42,
				Score: 95.5,
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(map[string]any{
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

			schema := zyn.Object(map[string]zyn.Z{
				"Name": zyn.String(),
				"Address": zyn.Object(map[string]zyn.Z{
					"Street": zyn.String(),
					"City":   zyn.String(),
				}),
			})

			data := Person{
				Name: "John",
				Address: Address{
					Street: "123 Main St",
					City:   "Boston",
				},
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(map[string]any{
				"name": "John",
				"address": map[string]any{
					"street": "123 Main St",
					"city":   "Boston",
				},
			}))
		})

		Specify("optional fields", func() {
			type TestStruct struct {
				Name  string
				Email *string
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name":  zyn.String(),
				"Email": zyn.String().Optional(),
			})

			data := TestStruct{
				Name:  "John",
				Email: nil,
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(map[string]any{
				"name": "John",
			}))
		})

		Specify("already dumped map[string]any", func() {
			type TestStruct struct {
				Name  string
				Age   int
				Score float64
			}

			schema := zyn.Object(map[string]zyn.Z{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})

			// First dump
			data := TestStruct{
				Name:  "John",
				Age:   42,
				Score: 95.5,
			}

			firstDump, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(firstDump).To(Equal(map[string]any{
				"name":  "John",
				"age":   int64(42),
				"score": 95.5,
			}))

			// Dump again with the already dumped data
			secondDump, err := schema.Dump(firstDump)
			Expect(err).ToNot(HaveOccurred())
			Expect(secondDump).To(Equal(map[string]any{
				"name":  "John",
				"age":   int64(42),
				"score": 95.5,
			}))
		})

		Specify("invalid map[string]any", func() {
			schema := zyn.Object(map[string]zyn.Z{
				"Name":  zyn.String(),
				"Age":   zyn.Number(),
				"Score": zyn.Number(),
			})

			// Missing required field
			data := map[string]any{
				"name": "John",
				// Missing Age
				"score": 95.5,
			}

			_, err := schema.Dump(data)
			Expect(err).To(MatchError(validate.FieldError{Message: "missing required field: Age"}))
		})

		Describe("Invalid Inputs", func() {
			Specify("nil value", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
				})

				_, err := schema.Dump(nil)
				Expect(err).To(MatchError(validate.FieldError{Message: "value is required but was nil"}))
			})

			Specify("nil pointer", func() {
				type TestStruct struct {
					Name string
				}

				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
				})

				var data *TestStruct
				_, err := schema.Dump(data)
				Expect(err).To(MatchError(validate.FieldError{Message: "value is required but was nil"}))
			})

			Specify("optional nil value", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
				}).Optional()

				result, err := schema.Dump(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})

			Specify("optional nil pointer", func() {
				type TestStruct struct {
					Name string
				}

				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
				}).Optional()

				var data *TestStruct
				result, err := schema.Dump(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})

			Specify("non-struct value", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
				})

				_, err := schema.Dump("not a struct")
				Expect(err).To(MatchError(validate.FieldError{Message: "invalid type: expected struct or map[string]any"}))
			})

			Specify("missing required field", func() {
				type TestStruct struct {
					Name string
				}

				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
					"Age":  zyn.Number(),
				})

				data := TestStruct{
					Name: "John",
				}

				_, err := schema.Dump(data)
				Expect(err).To(MatchError(validate.FieldError{Message: "missing required field: Age"}))
			})
		})

		Describe("Map Input", func() {
			Specify("valid map with snake case keys", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name":  zyn.String(),
					"Age":   zyn.Number(),
					"Score": zyn.Number(),
				})

				data := map[string]any{
					"name":  "John",
					"age":   42,
					"score": 95.5,
				}

				result, err := schema.Dump(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(map[string]any{
					"name":  "John",
					"age":   int64(42),
					"score": 95.5,
				}))
			})

			Specify("valid map with mixed case keys", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name":  zyn.String(),
					"Age":   zyn.Number(),
					"Score": zyn.Number(),
				})

				data := map[string]any{
					"Name":  "John",
					"age":   42,
					"Score": 95.5,
				}

				result, err := schema.Dump(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(map[string]any{
					"name":  "John",
					"age":   int64(42),
					"score": 95.5,
				}))
			})

			Specify("nested object map", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
					"Address": zyn.Object(map[string]zyn.Z{
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

				result, err := schema.Dump(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(map[string]any{
					"name": "John",
					"address": map[string]any{
						"street": "123 Main St",
						"city":   "Boston",
					},
				}))
			})

			Specify("optional fields in map", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name":  zyn.String(),
					"Email": zyn.String().Optional(),
				})

				data := map[string]any{
					"name": "John",
				}

				result, err := schema.Dump(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(map[string]any{
					"name": "John",
				}))
			})

			Specify("nil optional field in map", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name":  zyn.String(),
					"Email": zyn.String().Optional(),
				})

				data := map[string]any{
					"name":  "John",
					"email": nil,
				}

				result, err := schema.Dump(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(map[string]any{
					"name": "John",
				}))
			})

			Specify("invalid field type in map", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
					"Age":  zyn.Number(),
				})

				data := map[string]any{
					"name": "John",
					"age":  "not a number",
				}

				_, err := schema.Dump(data)
				Expect(err).To(MatchError(validate.FieldError{Message: "invalid field value for Age: :invalid type: expected number or convertible to number"}))
			})

			Specify("missing required field in map", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
					"Age":  zyn.Number(),
				})

				data := map[string]any{
					"name": "John",
				}

				_, err := schema.Dump(data)
				Expect(err).To(MatchError(validate.FieldError{Message: "missing required field: Age"}))
			})

			Specify("invalid nested object in map", func() {
				schema := zyn.Object(map[string]zyn.Z{
					"Name": zyn.String(),
					"Address": zyn.Object(map[string]zyn.Z{
						"Street": zyn.String(),
						"City":   zyn.String(),
					}),
				})

				data := map[string]any{
					"name":    "John",
					"address": "not an object",
				}

				_, err := schema.Dump(data)
				Expect(err).To(MatchError(validate.FieldError{Message: "invalid field value for Address: :invalid type: expected struct or map[string]any"}))
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

			schema := zyn.Object(map[string]zyn.Z{
				"FirstName": zyn.String(),
				"LastName":  zyn.String(),
				"Age":       zyn.Number(),
				"Score":     zyn.Number(),
			})

			data := TestStruct{
				FirstName: "John",
				LastName:  "Doe",
				Age:       42,
				Score:     95.5,
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(map[string]any{
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

			schema := zyn.Object(map[string]zyn.Z{
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

			schema := zyn.Object(map[string]zyn.Z{
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

			schema := zyn.Object(map[string]zyn.Z{
				"FirstName": zyn.String(),
				"LastName":  zyn.String(),
				"Address": zyn.Object(map[string]zyn.Z{
					"StreetName": zyn.String(),
					"CityName":   zyn.String(),
				}),
			})

			data := Person{
				FirstName: "John",
				LastName:  "Doe",
				Address: Address{
					StreetName: "123 Main St",
					CityName:   "Boston",
				},
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
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

			schema := zyn.Object(map[string]zyn.Z{
				"first_name": zyn.String(),
				"last_name":  zyn.String(),
				"age":        zyn.Number(),
				"score":      zyn.Number(),
			})

			data := TestStruct{
				FirstName: "John",
				LastName:  "Doe",
				Age:       42,
				Score:     95.5,
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
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

			schema := zyn.Object(map[string]zyn.Z{
				"FirstName": zyn.String(),
				"last_name": zyn.String(),
				"Age":       zyn.Number(),
				"score":     zyn.Number(),
			})

			data := TestStruct{
				FirstName: "John",
				LastName:  "Doe",
				Age:       42,
				Score:     95.5,
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
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

			schema := zyn.Object(map[string]zyn.Z{
				"first_name": zyn.String(),
				"last_name":  zyn.String(),
				"address": zyn.Object(map[string]zyn.Z{
					"street_name": zyn.String(),
					"city_name":   zyn.String(),
				}),
			})

			data := Person{
				FirstName: "John",
				LastName:  "Doe",
				Address: Address{
					StreetName: "123 Main St",
					CityName:   "Boston",
				},
			}

			result, err := schema.Dump(data)
			Expect(err).ToNot(HaveOccurred())
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
				type MyStruct struct {
					Value uuid.UUID
				}
				var schema = zyn.Object(map[string]zyn.Z{
					"value": zyn.UUID(),
				})
				value := uuid.New()
				data := map[string]any{
					"value": value.String(),
				}
				var res MyStruct
				Expect(schema.Parse(data, &res)).To(Succeed())
				Expect(res).To(Equal(MyStruct{Value: value}))
			})
		})
	})
})
