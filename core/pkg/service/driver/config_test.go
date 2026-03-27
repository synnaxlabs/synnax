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
	"github.com/synnaxlabs/synnax/pkg/service/driver"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		It("should fail when DB is nil", func() {
			cfg := driver.Config{
				Rack:      rackService,
				Task:      taskService,
				Framer:    framerSvc,
				Channel:   channelSvc,
				Factories: []driver.Factory{&mockFactory{name: "test"}},
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
				Factories: []driver.Factory{&mockFactory{name: "test"}},
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
				Factories: []driver.Factory{&mockFactory{name: "test"}},
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
				Factories: []driver.Factory{&mockFactory{name: "test"}},
			}
			Expect(cfg.Validate()).To(HaveOccurred())
		})
	})
})
