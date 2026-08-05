// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("KeyFromOntologyID", func() {
	It("Should return the key of a project ID", func() {
		key := uuid.New()
		Expect(project.KeyFromOntologyID(project.OntologyID(key))).To(Equal(key))
	})

	DescribeTable("Should reject an ID that is not a project",
		func(id ontology.ID, msg string) {
			Expect(project.KeyFromOntologyID(id)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("a group",
			ontology.ID{Type: ontology.ResourceTypeGroup, Key: uuid.NewString()},
			`must be a project, got "group"`),
		Entry("a zero ID", ontology.ID{}, `must be a project, got ""`),
		Entry("a project whose key is not a UUID",
			ontology.ID{Type: ontology.ResourceTypeProject, Key: "not-a-uuid"},
			`invalid project key "not-a-uuid"`),
		Entry("a project with an empty key",
			ontology.ID{Type: ontology.ResourceTypeProject},
			`invalid project key ""`),
	)
})
