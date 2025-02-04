// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schema_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resource", func() {
	Describe("ID Validation", func() {
		It("Should return an error if the resource ID does not have a key", func() {
			id := schema.ID{Type: "foo"}
			err := id.Validate()
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
		It("Should return an error if the resource ID does not have a type", func() {
			id := ontology.ID{Key: "foo"}
			err := id.Validate()
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
		It("Should return nil if the resource ID is valid", func() {
			id := schema.ID{Type: "foo", Key: "bar"}
			err := id.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("ParseID", func() {
		It("Should parse an ID from a string", func() {
			id, err := schema.ParseID("foo:bar")
			Expect(err).NotTo(HaveOccurred())
			Expect(id.Type).To(Equal(schema.Type("foo")))
			Expect(id.Key).To(Equal("bar"))
		})
		It("Should return an error if the ID has an invalid structure", func() {
			_, err := schema.ParseID("foo")
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
	})
	Describe("ParseIDs", func() {
		It("Should parse a list of IDs from a list of strings", func() {
			ids, err := schema.ParseIDs([]string{"foo:bar", "foo:baz"})
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(ConsistOf(schema.ID{Type: "foo", Key: "bar"}, schema.ID{Type: "foo", Key: "baz"}))
		})
		It("Should return an error if any of the IDs have an invalid structure", func() {
			_, err := schema.ParseIDs([]string{"foo:bar", "foo"})
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
	})

})
