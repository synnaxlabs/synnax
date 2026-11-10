// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apiarc.ServiceConfig {
		return apiarc.ServiceConfig{
			Instrumentation: alamos.New("test"),
			DB:              &gorp.DB{},
			Arc:             &arc.Service{},
			Status:          &status.Service{},
			RBAC:            &rbac.Service{},
			Stopper:         &task.Stopper{},
			DispatchChannel: func(
				context.Context,
				apichannel.DispatchRequest,
			) (apichannel.DispatchResponse, error) {
				return apichannel.DispatchResponse{}, nil
			},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(clear func(*apiarc.ServiceConfig), msg string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"db",
				func(c *apiarc.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"arc",
				func(c *apiarc.ServiceConfig) { c.Arc = nil },
				"arc: must be non-nil",
			),
			Entry(
				"status",
				func(c *apiarc.ServiceConfig) { c.Status = nil },
				"status: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apiarc.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
			Entry(
				"dispatch_channel",
				func(c *apiarc.ServiceConfig) { c.DispatchChannel = nil },
				"dispatch_channel: must be non-nil",
			),
			Entry(
				"stopper",
				func(c *apiarc.ServiceConfig) { c.Stopper = nil },
				"stopper: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiarc.ServiceConfig{}.Override(cfg)
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.Arc).To(BeIdenticalTo(cfg.Arc))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.DispatchChannel).ToNot(BeNil())
			Expect(merged.Stopper).To(BeIdenticalTo(cfg.Stopper))
			Expect(merged.Meta).To(Equal(cfg.Meta))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiarc.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.Arc).To(BeIdenticalTo(cfg.Arc))
			Expect(merged.Status).To(BeIdenticalTo(cfg.Status))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
			Expect(merged.DispatchChannel).ToNot(BeNil())
			Expect(merged.Stopper).To(BeIdenticalTo(cfg.Stopper))
			Expect(merged.Meta).To(Equal(cfg.Meta))
		})
	})
})
