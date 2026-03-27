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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Context", Ordered, func() {
	var (
		dist      mock.Node
		statusSvc *status.Service
	)

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc := MustSucceed(label.OpenService(
			ctx,
			label.ServiceConfig{
				DB:       dist.DB,
				Ontology: dist.Ontology,
				Group:    dist.Group,
			}),
		)
		statusSvc = MustSucceed(status.OpenService(
			ctx,
			status.ServiceConfig{
				Ontology: dist.Ontology,
				DB:       dist.DB,
				Group:    dist.Group,
				Label:    labelSvc,
			}),
		)

		DeferCleanup(func() {
			Expect(dist.Close()).To(Succeed())
		})
	})

	Describe("NewContext", func() {
		It("should create a context with the given status service", func() {
			driverCtx := driver.NewContext(ctx, statusSvc)
			Expect(driverCtx.Context).To(Equal(ctx))
		})
	})

	Describe("SetStatus", func() {
		It("should set a status", func() {
			driverCtx := driver.NewContext(ctx, statusSvc)
			stat := task.Status{
				Key:     "test-status-1",
				Variant: xstatus.VariantSuccess,
				Time:    telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())

			var statuses []status.Status[task.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[task.StatusDetails]]().
				WhereKeys("test-status-1").
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			Expect(statuses).To(HaveLen(1))
			Expect(statuses[0].Variant).To(Equal(xstatus.VariantSuccess))
		})

		It("should auto-fill time when zero", func() {
			driverCtx := driver.NewContext(ctx, statusSvc)
			beforeTime := telem.Now()
			stat := task.Status{
				Key:     "test-status-2",
				Variant: xstatus.VariantInfo,
				Time:    0,
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())
			afterTime := telem.Now()

			var statuses []status.Status[task.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[task.StatusDetails]]().
				WhereKeys("test-status-2").
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			Expect(statuses).To(HaveLen(1))
			Expect(statuses[0].Time).To(BeNumerically(">=", beforeTime))
			Expect(statuses[0].Time).To(BeNumerically("<=", afterTime))
		})

		It("should preserve provided time", func() {
			driverCtx := driver.NewContext(ctx, statusSvc)
			providedTime := telem.TimeStamp(1000000000)
			stat := task.Status{
				Key:     "test-status-3",
				Variant: xstatus.VariantWarning,
				Time:    providedTime,
			}
			Expect(driverCtx.SetStatus(stat)).To(Succeed())

			var statuses []status.Status[task.StatusDetails]
			Expect(gorp.NewRetrieve[string, status.Status[task.StatusDetails]]().
				WhereKeys("test-status-3").
				Entries(&statuses).
				Exec(ctx, dist.DB)).To(Succeed())
			Expect(statuses).To(HaveLen(1))
			Expect(statuses[0].Time).To(Equal(providedTime))
		})

		It("should fail with empty key", func() {
			driverCtx := driver.NewContext(ctx, statusSvc)
			stat := task.Status{
				Key:     "",
				Variant: xstatus.VariantSuccess,
				Time:    telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).To(MatchError(ContainSubstring("key")))
		})

		It("should fail with empty variant", func() {
			driverCtx := driver.NewContext(ctx, statusSvc)
			stat := task.Status{
				Key:     "test-status-invalid",
				Variant: "",
				Time:    telem.Now(),
			}
			Expect(driverCtx.SetStatus(stat)).To(MatchError(ContainSubstring("variant")))
		})
	})
})
