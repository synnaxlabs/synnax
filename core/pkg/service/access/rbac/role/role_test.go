// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
)

var _ = Describe("Role", func() {
	var r role.Role
	BeforeEach(func() {
		r = role.Role{
			Name:        "test-role",
			Description: "Test role",
			Key:         uuid.New(),
		}
	})
	Describe("GorpKey", func() {
		It("Should return the key of the role", func() {
			Expect(r.GorpKey()).To(Equal(r.Key))
		})
	})
	Describe("SetOptions", func() {
		It("Should return nil", func() {
			Expect(r.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the ontology ID of the role", func() {
			Expect(r.OntologyID()).To(Equal(ontology.ID{
				Type: role.OntologyType,
				Key:  r.Key.String(),
			}))
		})
	})
})
