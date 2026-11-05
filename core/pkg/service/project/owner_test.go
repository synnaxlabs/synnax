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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
)

var _ = Describe("ValidateOwner", func() {
	It("Should accept a project alone", func() {
		Expect(project.ValidateOwner(new(uuid.New()), nil)).To(Succeed())
	})

	It("Should accept a range alone", func() {
		Expect(project.ValidateOwner(nil, new(ranger.Key(uuid.New())))).To(Succeed())
	})

	DescribeTable(
		"Should reject anything but exactly one owner",
		func(proj *project.Key, rng *ranger.Key) {
			Expect(project.ValidateOwner(proj, rng)).To(MatchError(ContainSubstring(
				"project: must belong to exactly one of a project and a range",
			)))
		},
		Entry("neither", nil, nil),
		Entry("both", new(uuid.New()), new(ranger.Key(uuid.New()))),
	)
})
