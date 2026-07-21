// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/types/v0"
)

// key composes rack key 65538 (node 1, local 2) in the high 32 bits and local
// task key 7 in the low 32.
const key = v0.Key(65538<<32 | 7)

var _ = Describe("Key", func() {
	Describe("Rack", func() {
		It("Should return the rack the task belongs to", func() {
			Expect(key.Rack()).To(Equal(rack.Key(65538)))
		})
	})

	Describe("LocalKey", func() {
		It("Should return the task's key within its rack", func() {
			Expect(key.LocalKey()).To(Equal(uint32(7)))
		})
	})

	Describe("IsValid", func() {
		DescribeTable("Should require both the rack and local components",
			func(k v0.Key, valid bool) {
				Expect(k.IsValid()).To(Equal(valid))
			},
			Entry("both components set", key, true),
			Entry("zero key", v0.Key(0), false),
			Entry("missing local key", v0.Key(65538<<32), false),
			Entry("missing rack key", v0.Key(7), false),
		)
	})

	Describe("OntologyID", func() {
		It("Should return the task ontology identifier", func() {
			Expect(key.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeTask,
				Key:  key.String(),
			}))
		})
	})

	Describe("String", func() {
		It("Should format the key as its decimal value", func() {
			Expect(v0.Key(7).String()).To(Equal("7"))
		})
	})
})

var _ = Describe("Task", func() {
	Describe("GorpKey", func() {
		It("Should return the task's key", func() {
			Expect(v0.Task{Key: key}.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Task{}.SetOptions()).To(BeNil())
		})
	})
})
