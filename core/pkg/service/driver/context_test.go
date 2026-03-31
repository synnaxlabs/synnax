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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Context", func() {
	Describe("NewContext", func() {
		It("should create a context with the given status service", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx, driver.WithStatusService(statusSvc))
			Expect(driverCtx.Context).To(Equal(ctx))
		})
	})

	Describe("Register", func() {
		It("should be a no-op when no register function is set", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx)
			driverCtx.Register(&mockTask{})
		})

		It("should call the register function with the task", func(ctx context.Context) {
			var registered driver.Task
			driverCtx := driver.NewContext(ctx, driver.WithRegister(func(t driver.Task) {
				registered = t
			}))
			mt := &mockTask{key: 42}
			driverCtx.Register(mt)
			Expect(registered).To(Equal(mt))
		})
	})

	Describe("SetStatus", func() {
		It("should return nil when status service is nil", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx)
			stat := task.Status{
				Key:     "test-nil-svc",
				Variant: xstatus.VariantSuccess,
				Time:    telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())
		})

		It("should set a status", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx, driver.WithStatusService(statusSvc))
			stat := task.Status{
				Key:     "test-status-1",
				Variant: xstatus.VariantSuccess,
				Time:    telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())

			var statuses []status.Status[task.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[task.StatusDetails]](nil).
				WhereKeys("test-status-1").
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			Expect(statuses).To(HaveLen(1))
			Expect(statuses[0].Variant).To(Equal(xstatus.VariantSuccess))
		})

		It("should auto-fill time when zero", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx, driver.WithStatusService(statusSvc))
			beforeTime := telem.Now()
			stat := task.Status{
				Key:     "test-status-2",
				Variant: xstatus.VariantInfo,
				Time:    0,
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())
			afterTime := telem.Now()

			var statuses []status.Status[task.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[task.StatusDetails]](nil).
				WhereKeys("test-status-2").
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			Expect(statuses).To(HaveLen(1))
			Expect(statuses[0].Time).To(BeNumerically(">=", beforeTime))
			Expect(statuses[0].Time).To(BeNumerically("<=", afterTime))
		})

		It("should preserve provided time", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx, driver.WithStatusService(statusSvc))
			providedTime := telem.TimeStamp(1000000000)
			stat := task.Status{
				Key:     "test-status-3",
				Variant: xstatus.VariantWarning,
				Time:    providedTime,
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())

			var statuses []status.Status[task.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[task.StatusDetails]](nil).
				WhereKeys("test-status-3").
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			Expect(statuses).To(HaveLen(1))
			Expect(statuses[0].Time).To(Equal(providedTime))
		})

		It("should fail with empty key", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx, driver.WithStatusService(statusSvc))
			stat := task.Status{
				Variant: xstatus.VariantSuccess,
				Time:    telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).To(MatchError(ContainSubstring("key")))
		})

		It("should fail with empty variant", func(ctx context.Context) {
			driverCtx := driver.NewContext(ctx, driver.WithStatusService(statusSvc))
			stat := task.Status{
				Key:  "test-status-invalid",
				Time: telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).
				To(MatchError(ContainSubstring("variant")))
		})
	})
})
