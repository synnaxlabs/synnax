// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bundle_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project/bundle"
)

var _ = Describe("Config", func() {
	full := func() bundle.Config {
		return bundle.Config{
			DB:       db,
			Resource: resourceSvc,
			Project:  projectSvc,
			Group:    groupSvc,
			Panel:    panelSvc,
			ImEx:     imexSvc,
		}
	}

	It("Should accept a config with every field set", func() {
		Expect(full().Validate()).To(Succeed())
	})

	DescribeTable(
		"Should require each field",
		func(clear func(*bundle.Config), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry("db", func(c *bundle.Config) { c.DB = nil }, "db: must be non-nil"),
		Entry(
			"resource",
			func(c *bundle.Config) { c.Resource = nil },
			"resource: must be non-nil",
		),
		Entry(
			"project",
			func(c *bundle.Config) { c.Project = nil },
			"project: must be non-nil",
		),
		Entry(
			"group",
			func(c *bundle.Config) { c.Group = nil },
			"group: must be non-nil",
		),
		Entry(
			"panel",
			func(c *bundle.Config) { c.Panel = nil },
			"panel: must be non-nil",
		),
		Entry("imex", func(c *bundle.Config) { c.ImEx = nil }, "imex: must be non-nil"),
	)

	It("Should take each field set on the override", func() {
		Expect(bundle.Config{}.Override(full())).To(Equal(full()))
	})

	It("Should keep each field the override leaves unset", func() {
		Expect(full().Override(bundle.Config{})).To(Equal(full()))
	})
})
