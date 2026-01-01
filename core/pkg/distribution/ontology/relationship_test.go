// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Relationship", func() {
	Describe("ParseRelationship", func() {
		It("Should parse a relationship from a string", func() {
			r, err := ontology.ParseRelationship([]byte("foo:qux->parent->bar:baz"))
			Expect(err).NotTo(HaveOccurred())
			Expect(r.From.Type).To(Equal(core.Type("foo")))
			Expect(r.From.Key).To(Equal("qux"))
			Expect(r.Type).To(Equal(ontology.ParentOf))
			Expect(r.To.Type).To(Equal(core.Type("bar")))
			Expect(r.To.Key).To(Equal("baz"))
		})
		It("Should return an error if the relationship has an invalid structure", func() {
			_, err := ontology.ParseRelationship([]byte("foo:qux-parent->bar"))
			Expect(err).To(HaveOccurredAs(validate.Error))
		})
	})

})
