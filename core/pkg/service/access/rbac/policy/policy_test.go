// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Policy", func() {
	var p policy.Policy
	BeforeEach(func() {
		p = policy.Policy{
			Name:   "test-policy",
			Key:    uuid.New(),
			Effect: policy.EffectAllow,
			Constraint: constraint.Constraint{
				IDs:     []ontology.ID{{Type: "channel", Key: "ch1"}},
				Actions: set.New(access.ActionRetrieve),
			},
		}
	})
	Describe("GorpKey", func() {
		It("Should return the key of the policy", func() {
			Expect(p.GorpKey()).To(Equal(p.Key))
		})
	})
	Describe("SetOptions", func() {
		It("Should return the options of the policy", func() {
			Expect(p.SetOptions()).To(BeNil())
		})
	})
	Describe("OntologyID", func() {
		It("Should return the ontology ID of the policy", func() {
			Expect(p.OntologyID()).To(Equal(ontology.ID{
				Type: policy.OntologyType,
				Key:  p.Key.String(),
			}))
		})
	})
})
