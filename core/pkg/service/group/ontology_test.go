// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
)

var _ = Describe("Ontology", func() {
	Describe("OntologyIDsFromGroups", func() {
		It("Should map a slice of groups to ontology IDs", func() {
			a := group.Group{Key: uuid.New(), Name: "a"}
			b := group.Group{Key: uuid.New(), Name: "b"}
			Expect(group.OntologyIDsFromGroups([]group.Group{a, b})).
				To(ConsistOf(a.OntologyID(), b.OntologyID()))
		})
	})
})
