// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	Describe("FactoryConfig", func() {
		Describe("Override", func() {
			It("Should override nil fields with provided values", func() {
				base := sift.FactoryConfig{}
				override := sift.FactoryConfig{
					Device:        deviceSvc,
					Framer:        framerSvc,
					Channel:       dist.Channel,
					Status:        statusSvc,
					Task:          taskSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				result := base.Override(override)
				Expect(result.Device).To(Equal(deviceSvc))
				Expect(result.Framer).To(Equal(framerSvc))
				Expect(result.Channel).To(Equal(dist.Channel))
				Expect(result.Status).To(Equal(statusSvc))
				Expect(result.Task).To(Equal(taskSvc))
				Expect(result.ClientFactory).ToNot(BeNil())
			})

			It("Should preserve existing non-nil fields", func() {
				base := sift.FactoryConfig{
					Device: deviceSvc,
				}
				override := sift.FactoryConfig{}
				result := base.Override(override)
				Expect(result.Device).To(Equal(deviceSvc))
			})
		})

		Describe("Validate", func() {
			It("Should return error when Device is nil", func() {
				cfg := sift.FactoryConfig{
					Framer:        framerSvc,
					Channel:       dist.Channel,
					Status:        statusSvc,
					Task:          taskSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("Should return error when Framer is nil", func() {
				cfg := sift.FactoryConfig{
					Device:        deviceSvc,
					Channel:       dist.Channel,
					Status:        statusSvc,
					Task:          taskSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("Should return error when Channel is nil", func() {
				cfg := sift.FactoryConfig{
					Device:        deviceSvc,
					Framer:        framerSvc,
					Status:        statusSvc,
					Task:          taskSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("Should return error when Status is nil", func() {
				cfg := sift.FactoryConfig{
					Device:        deviceSvc,
					Framer:        framerSvc,
					Channel:       dist.Channel,
					Task:          taskSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("Should return error when Task is nil", func() {
				cfg := sift.FactoryConfig{
					Device:        deviceSvc,
					Framer:        framerSvc,
					Channel:       dist.Channel,
					Status:        statusSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("Should return error when ClientFactory is nil", func() {
				cfg := sift.FactoryConfig{
					Device:  deviceSvc,
					Framer:  framerSvc,
					Channel: dist.Channel,
					Status:  statusSvc,
					Task:    taskSvc,
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})

			It("Should succeed when all required fields are provided", func() {
				cfg := sift.FactoryConfig{
					Device:        deviceSvc,
					Framer:        framerSvc,
					Channel:       dist.Channel,
					Status:        statusSvc,
					Task:          taskSvc,
					ClientFactory: MustSucceed(client.NewMockFactory()),
				}
				Expect(cfg.Validate()).To(Succeed())
			})
		})
	})

	Describe("OpenFactory", func() {
		It("Should create a factory with valid config", func() {
			factory, err := sift.OpenFactory(sift.FactoryConfig{
				Device:        deviceSvc,
				Framer:        framerSvc,
				Channel:       dist.Channel,
				Status:        statusSvc,
				Task:          taskSvc,
				ClientFactory: MustSucceed(client.NewMockFactory()),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(factory).ToNot(BeNil())
			Expect(factory.Name()).To(Equal("Sift"))
			Expect(factory.Close()).To(Succeed())
		})

		It("Should return error with invalid config", func() {
			_, err := sift.OpenFactory(sift.FactoryConfig{})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ConfigureTask", func() {
		var (
			ctx     context.Context
			factory *sift.Factory
		)

		BeforeEach(func() {
			ctx = context.Background()
			factory = MustSucceed(sift.OpenFactory(sift.FactoryConfig{
				Device:        deviceSvc,
				Framer:        framerSvc,
				Channel:       dist.Channel,
				Status:        statusSvc,
				Task:          taskSvc,
				ClientFactory: MustSucceed(client.NewMockFactory()),
			}))
		})

		AfterEach(func() {
			if factory != nil {
				Expect(factory.Close()).To(Succeed())
			}
		})

		It("Should return ErrTaskNotHandled for unknown task types", func() {
			_, err := factory.ConfigureTask(driver.Context{Context: ctx}, task.Task{
				Type: "unknown_type",
			})
			Expect(err).To(MatchError(driver.ErrTaskNotHandled))
		})

		It("Should return ErrTaskNotHandled for empty task type", func() {
			_, err := factory.ConfigureTask(driver.Context{Context: ctx}, task.Task{})
			Expect(err).To(MatchError(driver.ErrTaskNotHandled))
		})
	})
})
