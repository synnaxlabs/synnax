// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Resource", func() {
	Describe("Type", func() {
		Describe("String", func() {
			It("Should return the string representation of the type", func() {
				Expect(ontology.Type("abc").String()).To(Equal("abc"))
			})
		})
	})

	Describe("ID", func() {
		Describe("Validation", func() {
			It("Should return an error if the resource ID does not have a key", func() {
				id := core.ID{Type: "foo"}
				err := id.Validate()
				Expect(err).To(HaveOccurredAs(validate.Error))
			})
			It("Should return an error if the resource ID does not have a type", func() {
				id := ontology.ID{Key: "foo"}
				err := id.Validate()
				Expect(err).To(HaveOccurredAs(validate.Error))
			})
			It("Should return nil if the resource ID is valid", func() {
				id := core.ID{Type: "foo", Key: "bar"}
				err := id.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("IsType", func() {
			It("Should return true if the type of the key is empty", func() {
				Expect(ontology.ID{Type: "foo"}.IsType()).To(BeTrue())
			})
			It("Should return false if the type of the key is not empty", func() {
				Expect(ontology.ID{Type: "Bar", Key: "foo"}.IsType()).To(BeFalse())
			})
		})

		Describe("IsZero", func() {
			It("Should return true if both the type and key are empty", func() {
				Expect(ontology.ID{}.IsZero()).To(BeTrue())
			})
			It("Should return false when the type is not empty", func() {
				Expect(ontology.ID{Type: "cat"}.IsZero()).To(BeFalse())
			})
			It("Should return false when the key is not empty", func() {
				Expect(ontology.ID{Key: "cat"}.IsZero()).To(BeFalse())
			})
		})

		Describe("String", func() {
			It("Should return the string representation to the ID", func() {
				Expect(ontology.ID{Key: "dog", Type: "cat"}.String()).To(Equal("cat:dog"))
			})
		})

		Describe("Parse", func() {
			Context("Single", func() {
				It("Should parse an ID from a string", func() {
					id := MustSucceed(core.ParseID("foo:bar"))
					Expect(id.Type).To(Equal(core.Type("foo")))
					Expect(id.Key).To(Equal("bar"))
				})

				It("Should return an error if the ID has an invalid structure", func() {
					_, err := core.ParseID("foo")
					Expect(err).To(HaveOccurredAs(validate.Error))
				})

				It("Should return an error if the ID is an empty string", func() {
					_, err := core.ParseID("")
					Expect(err).To(HaveOccurredAs(validate.Error))
				})

				It("Should parse an ID with empty type (leading colon)", func() {
					id := MustSucceed(core.ParseID(":bar"))
					Expect(id.Type).To(Equal(core.Type("")))
					Expect(id.Key).To(Equal("bar"))
				})

				It("Should parse an ID with empty key (trailing colon)", func() {
					id := MustSucceed(core.ParseID("foo:"))
					Expect(id.Type).To(Equal(core.Type("foo")))
					Expect(id.Key).To(Equal(""))
				})

				It("Should ignore subsequence semi-colors in the type", func() {
					id := MustSucceed(core.ParseID("foo:bar:baz"))
					Expect(id.Type).To(Equal(core.Type("foo")))
					Expect(id.Key).To(Equal("bar:baz"))
				})
			})

			Context("Multiple", func() {
				Describe("ParseIDs", func() {
					It("Should parse a list of IDs from a list of strings", func() {
						ids, err := core.ParseIDs([]string{"foo:bar", "foo:baz"})
						Expect(err).NotTo(HaveOccurred())
						Expect(ids).To(ConsistOf(core.ID{Type: "foo", Key: "bar"}, core.ID{Type: "foo", Key: "baz"}))
					})
					It("Should return an error if any of the IDs have an invalid structure", func() {
						_, err := core.ParseIDs([]string{"foo:bar", "foo"})
						Expect(err).To(HaveOccurredAs(validate.Error))
					})
					It("Should return an empty slice when given an empty slice", func() {
						ids, err := core.ParseIDs([]string{})
						Expect(err).NotTo(HaveOccurred())
						Expect(ids).To(BeEmpty())
					})
				})
			})
		})
	})

	Describe("Resource", func() {
		r := ontology.NewResource(
			zyn.Object(nil),
			ontology.ID{Type: "cat", Key: "dog"},
			"cat",
			map[string]any{},
		)

		It("Should correctly construct the resource", func() {
			Expect(r.ID.Type).To(Equal(core.Type("cat")))
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
				type myStruct struct {
					Cat string
				}
				var schema = zyn.Object(map[string]zyn.Schema{
					"cat": zyn.String(),
				})
				r := ontology.NewResource(
					schema,
					ontology.ID{Type: "cat", Key: "dog"},
					"cat",
					map[string]any{"cat": "milo"},
				)
				var v myStruct
				Expect(r.Parse(&v)).To(Succeed())
				Expect(v.Cat).To(Equal("milo"))
			})
		})
	})
})
