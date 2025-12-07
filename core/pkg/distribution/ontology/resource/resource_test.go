// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Resource", func() {
	Describe("Type", func() {
		Describe("String", func() {
			It("Should return the string representation of the type", func() {
				Expect(resource.Type("abc").String()).To(Equal("abc"))
			})
		})
	})

	Describe("ID", func() {
		Describe("Validation", func() {
			It("Should return an error if the resource ID does not have a key", func() {
				id := resource.ID{Type: "foo"}
				Expect(id.Validate()).To(HaveOccurredAs(validate.Error))
			})
			It("Should return an error if the resource ID does not have a type", func() {
				id := resource.ID{Key: "foo"}
				Expect(id.Validate()).To(HaveOccurredAs(validate.Error))
			})
			It("Should return nil if the resource ID is valid", func() {
				id := resource.ID{Type: "foo", Key: "bar"}
				Expect(id.Validate()).To(Succeed())
			})
		})

		Describe("IsType", func() {
			It("Should return true if the type of the key is empty", func() {
				Expect(resource.ID{Type: "foo"}.IsType()).To(BeTrue())
			})
			It("Should return false if the type of the key is not empty", func() {
				Expect(resource.ID{Type: "Bar", Key: "foo"}.IsType()).To(BeFalse())
			})
		})

		Describe("IsZero", func() {
			It("Should return true if both the type and key are empty", func() {
				Expect(resource.ID{}.IsZero()).To(BeTrue())
			})
			It("Should return false when the type is not empty", func() {
				Expect(resource.ID{Type: "cat"}.IsZero()).To(BeFalse())
			})
			It("Should return false when the key is not empty", func() {
				Expect(resource.ID{Key: "cat"}.IsZero()).To(BeFalse())
			})
		})

		Describe("String", func() {
			It("Should return the string representation to the ID", func() {
				Expect(resource.ID{Key: "dog", Type: "cat"}.String()).
					To(Equal("cat:dog"))
			})
		})

		Describe("Parse", func() {
			Context("Single", func() {
				It("Should parse an ID from a string", func() {
					id := MustSucceed(resource.ParseID("foo:bar"))
					Expect(id.Type).To(Equal(resource.Type("foo")))
					Expect(id.Key).To(Equal("bar"))
				})

				It("Should return an error if the ID has an invalid structure", func() {
					Expect(resource.ParseID("foo")).Error().To(HaveOccurredAs(validate.Error))
				})

				It("Should return an error if the ID is an empty string", func() {
					Expect(resource.ParseID("")).Error().To(HaveOccurredAs(validate.Error))
				})

				It("Should return an error if the ID has an empty type (leading colon)", func() {
					Expect(resource.ParseID(":bar")).Error().To(HaveOccurredAs(validate.Error))
				})

				It("Should return an error if the ID has an empty type with colons in key", func() {
					Expect(resource.ParseID(":word1:word2")).Error().To(HaveOccurredAs(validate.Error))
				})

				It("Should return an error if the ID has an empty type and key starts with colon", func() {
					Expect(resource.ParseID("::word1")).Error().To(HaveOccurredAs(validate.Error))
				})

				It("Should parse an ID with empty key (trailing colon)", func() {
					id := MustSucceed(resource.ParseID("foo:"))
					Expect(id.Type).To(Equal(resource.Type("foo")))
					Expect(id.Key).To(Equal(""))
				})

				It("Should ignore subsequence semi-colors in the type", func() {
					id := MustSucceed(resource.ParseID("foo:bar:baz"))
					Expect(id.Type).To(Equal(resource.Type("foo")))
					Expect(id.Key).To(Equal("bar:baz"))
				})
			})

			Context("Multiple", func() {
				Describe("ParseIDs", func() {
					It("Should parse a list of IDs from a list of strings", func() {
						Expect(resource.ParseIDs([]string{"foo:bar", "foo:baz"})).
							To(ConsistOf(
								resource.ID{Type: "foo", Key: "bar"},
								resource.ID{Type: "foo", Key: "baz"},
							))
					})
					It("Should return an error if any of the IDs have an invalid structure", func() {
						Expect(resource.ParseIDs([]string{"foo:bar", "foo"})).Error().
							To(HaveOccurredAs(validate.Error))
					})
					It("Should return an empty slice when given an empty slice", func() {
						ids, err := resource.ParseIDs([]string{})
						Expect(err).NotTo(HaveOccurred())
						Expect(ids).To(BeEmpty())
					})
				})
			})
		})
	})

	Describe("Resource", func() {
		r := resource.New(
			zyn.Object(nil),
			resource.ID{Type: "cat", Key: "dog"},
			"cat",
			map[string]any{},
		)
		It("Should correctly construct the resource", func() {
			Expect(r.ID.Type).To(Equal(resource.Type("cat")))
			Expect(r.ID.Key).To(Equal("dog"))
			Expect(r.Name).To(Equal("cat"))
			Expect(r.Data).To(BeEmpty())
		})
		Describe("BleveType", func() {
			It("Should return the type of the resource ID for classification within bleve", func() {
				Expect(r.BleveType()).To(Equal("cat"))
			})
		})
		Describe("GorpKey", func() {
			It("Should return the ID as the gorp key of the resource", func() {
				Expect(r.GorpKey()).To(Equal(r.ID))
			})
		})
		Describe("SetOptions", func() {
			It("Should return an empty slice", func() {
				Expect(r.SetOptions()).To(BeEmpty())
			})
		})
		Describe("Parse", func() {
			It("Should parse a resource from its schema", func() {
				type myStruct struct{ Cat string }
				schema := zyn.Object(map[string]zyn.Schema{"cat": zyn.String()})
				r := resource.New(
					schema,
					resource.ID{Type: "cat", Key: "dog"},
					"cat",
					map[string]any{"cat": "milo"},
				)
				var v myStruct
				Expect(r.Parse(&v)).To(Succeed())
				Expect(v.Cat).To(Equal("milo"))
			})
		})
	})

	Describe("IDs", func() {
		It("Should extract IDs from a slice of resources", func() {
			resources := []resource.Resource{
				resource.New(
					zyn.Object(nil),
					resource.ID{Type: "cat", Key: "dog1"},
					"cat1",
					map[string]any{},
				),
				resource.New(
					zyn.Object(nil),
					resource.ID{Type: "cat", Key: "dog2"},
					"cat2",
					map[string]any{},
				),
			}
			ids := resource.IDs(resources)
			Expect(ids).To(HaveLen(2))
			Expect(ids[0]).To(Equal(resource.ID{Type: "cat", Key: "dog1"}))
			Expect(ids[1]).To(Equal(resource.ID{Type: "cat", Key: "dog2"}))
		})

		It("Should return an empty slice for empty input", func() {
			ids := resource.IDs([]resource.Resource{})
			Expect(ids).To(BeEmpty())
		})
	})

	Describe("IDsToString", func() {
		It("Should convert IDs to strings", func() {
			ids := []resource.ID{
				{Type: "cat", Key: "dog1"},
				{Type: "cat", Key: "dog2"},
			}
			strings := resource.IDsToString(ids)
			Expect(strings).To(HaveLen(2))
			Expect(strings[0]).To(Equal("cat:dog1"))
			Expect(strings[1]).To(Equal("cat:dog2"))
		})

		It("Should return an empty slice for empty input", func() {
			strings := resource.IDsToString([]resource.ID{})
			Expect(strings).To(BeEmpty())
		})
	})
})
