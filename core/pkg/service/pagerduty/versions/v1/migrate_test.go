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
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	status "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateStatusKeys", func() {
	It(
		"Should point each alert at the migrated key of its status",
		func(ctx SpecContext) {
			migrated, _ := status.MigrateKey("task:12")
			cfg := MustSucceed(v1.MigrateStatusKeys(ctx, v1.TaskConfig{
				Alerts: []v1.Alert{
					{Key: "a", Status: "task:12"},
					{Key: "b", Status: "pressure-alarm"},
				},
			}))
			Expect(cfg.Alerts[0].Status).To(Equal(migrated))
			Expect(cfg.Alerts[1].Status).To(Equal("pressure-alarm"))
		},
	)
})
