// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
)

// key composes rack key 65538 (node 1, local 2) in the high 32 bits and local task key
// 7 in the low 32.
const key = v1.Key(65538<<32 | 7)

var _ = Describe("Task", func() {
	Describe("GorpKey", func() {
		It("Should return the task's key", func() {
			Expect(v1.Task{Key: key}.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should lease the task to its rack's node", func() {
			Expect(v1.Task{Key: key}.SetOptions()).To(Equal([]any{node.Key(1)}))
		})
	})

	Describe("Rack", func() {
		It("Should return the rack the task belongs to", func() {
			Expect(v1.Task{Key: key}.Rack()).To(Equal(rack.Key(65538)))
		})
	})

	Describe("String", func() {
		It("Should include the name and key when named", func() {
			t := v1.Task{Key: key, Name: "reader"}
			Expect(t.String()).To(Equal("[reader]<" + key.String() + ">"))
		})
		It("Should fall back to the key when unnamed", func() {
			Expect(v1.Task{Key: key}.String()).To(Equal(key.String()))
		})
	})
	Describe("OntologyID", func() {
		It("Should return the task ontology identifier", func() {
			Expect(v1.Task{Key: key}.OntologyID()).To(Equal(key.OntologyID()))
		})
	})
})
