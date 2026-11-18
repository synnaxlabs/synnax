// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package file_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/file"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	var f *file.Factory
	BeforeEach(func() {
		f = MustSucceed(file.NewFactory(file.FactoryConfig{DB: db, Status: statusSvc}))
	})

	Describe("NewFactory", func() {
		It("Should require a DB and a status service", func() {
			Expect(file.NewFactory(file.FactoryConfig{})).Error().To(MatchError(And(
				ContainSubstring("db: must be non-nil"),
				ContainSubstring("status: must be non-nil"),
			)))
		})
	})

	Describe("Name", func() {
		It("Should return the file integration", func() {
			Expect(f.Name()).To(Equal("file"))
		})
	})

	Describe("InitialTasks", func() {
		It("Should return one file browser task", func() {
			initial := f.InitialTasks()
			Expect(initial).To(HaveLen(1))
			Expect(initial[0].Name).To(Equal("File Browser"))
			Expect(initial[0].Type).To(Equal("file_browser"))
		})
	})

	Describe("ConfigureTask", func() {
		It("Should not handle another task type", func(ctx SpecContext) {
			Expect(
				f.ConfigureTask(ctx, task.Task{Type: "sift_scan"}, driver.NoCommand),
			).
				Error().
				To(MatchError(driver.ErrTaskNotHandled))
		})

		It("Should report the task as running", func(ctx SpecContext) {
			b := configure(ctx, "")
			stat := taskStatus(ctx, b.stored)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Task configured"))
			Expect(stat.Details.Running).To(BeTrue())
		})
	})

	Describe("Exec", func() {
		It("Should not accept a command other than list", func(ctx SpecContext) {
			stat := configure(ctx, "").execType(ctx, "start", nil)
			Expect(stat.Variant).To(Equal(status.VariantError))
			Expect(stat.Message).To(Equal("Failed to start task"))
			Expect(stat.Description).
				To(Equal("The task does not accept the 'start' command."))
		})
	})
})

var _ = Describe("Service", func() {
	It("Should require a DB", func(ctx SpecContext) {
		Expect(file.OpenService(ctx, file.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})

	It("Should store the configs of file browser tasks", func() {
		stores := fileSvc.Stores()
		Expect(stores).To(HaveLen(1))
		Expect(stores[0].Type()).To(Equal("file_browser"))
	})
})
