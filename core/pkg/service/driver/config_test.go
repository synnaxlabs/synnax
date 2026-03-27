// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Config", Ordered, func() {
	var (
		db           *gorp.DB
		rackService  *rack.Service
		taskService  *task.Service
		framerSvc    *framer.Service
		channelSvc   *channel.Service
		factory      driver.Factory
		hostProvider = mock.StaticHostKeyProvider(1)
	)

	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
		labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{DB: db, Ontology: otg, Group: g}))
		stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{Ontology: otg, DB: db, Group: g, Label: labelSvc}))
		rackService = MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       stat,
		}))
		taskService = MustSucceed(task.OpenService(
			ctx,
			task.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Rack:     rackService,
				Status:   stat,
			}),
		)
		factory = &mockFactory{name: "test"}
		ShouldNotLeakGoroutines()

		DeferCleanup(func() {
			Expect(db.Close()).To(Succeed())
		})
	})

	Describe("Validate", func() {
		It("should fail when DB is nil", func() {
			cfg := driver.Config{
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Rack is nil", func() {
			cfg := driver.Config{
				DB:        db,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Task is nil", func() {
			cfg := driver.Config{
				DB:        db,
				Rack:      rackService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Factories: []driver.Factory{factory},
				Host:      hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Factories is empty", func() {
			cfg := driver.Config{
				DB:      db,
				Rack:    rackService,
				Task:    taskService,
				Framer:  framerSvc,
				Channel: channelSvc,
				Host:    hostProvider,
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})

		It("should fail when Host is zero", func() {
			cfg := driver.Config{
				DB:        db,
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Factories: []driver.Factory{factory},
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})
	})
})
