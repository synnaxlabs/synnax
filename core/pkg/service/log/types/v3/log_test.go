// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/types/v3"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ = Describe("Log", func() {
	Describe("GorpKey", func() {
		It("Should return the log's key", func() {
			k := uuid.New()
			Expect(v3.Log{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v3.Log{}.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the log ontology identifier", func() {
			k := uuid.New()
			Expect(v3.Log{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeLog, Key: k.String(),
			}))
		})
	})
})
