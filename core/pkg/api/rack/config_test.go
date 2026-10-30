// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apirack.ServiceConfig {
		return apirack.ServiceConfig{
			Rack:   &rack.Service{},
			Device: &device.Service{},
			Task:   &task.Service{},
			Status: &status.Service{},
			RBAC:   &rbac.Service{},
			Auth:   &auth.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apirack.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"rack",
				func(c *apirack.ServiceConfig) { c.Rack = nil },
				"rack: must be non-nil",
			),
			Entry(
				"device",
				func(c *apirack.ServiceConfig) { c.Device = nil },
				"device: must be non-nil",
			),
			Entry(
				"task",
				func(c *apirack.ServiceConfig) { c.Task = nil },
				"task: must be non-nil",
			),
			Entry(
				"status",
				func(c *apirack.ServiceConfig) { c.Status = nil },
				"status: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apirack.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"auth",
				func(c *apirack.ServiceConfig) { c.Auth = nil },
				"auth: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apirack.ServiceConfig{}.Override(cfg)
			Expect(merged.Rack).To(BeIdenticalTo(cfg.Rack))
			Expect(merged.Device).To(BeIdenticalTo(cfg.Device))
			Expect(merged.Task).To(BeIdenticalTo(cfg.Task))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apirack.ServiceConfig{})
			Expect(merged.Rack).To(BeIdenticalTo(cfg.Rack))
			Expect(merged.Device).To(BeIdenticalTo(cfg.Device))
			Expect(merged.Task).To(BeIdenticalTo(cfg.Task))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
		})
	})
})
