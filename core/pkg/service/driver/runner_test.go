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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Runner", func() {
	var (
		t      task.Task
		opened int
		runErr error
	)
	waitRun := func(ctx context.Context) error {
		<-ctx.Done()
		return runErr
	}

	BeforeEach(func() {
		t = task.Task{Key: uuid.New(), Name: "runner-test", Type: "mock"}
		opened, runErr = 0, nil
	})

	open := func(run func(context.Context) error) *driver.Runner {
		GinkgoHelper()
		r := MustSucceed(driver.NewRunner(driver.RunnerConfig{
			Status: statusSvc,
			Task:   t,
			Open: func(context.Context) error {
				opened++
				return nil
			},
			Run: run,
		}))
		DeferCleanup(func() { Expect(r.Stop(false)).To(Succeed()) })
		return r
	}

	retrieve := func(ctx context.Context) task.Status {
		var stat task.Status
		_ = statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)
		return stat
	}

	Describe("NewRunner", func() {
		It("Should reject a configuration with no hooks", func() {
			Expect(driver.NewRunner(driver.RunnerConfig{
				Status: statusSvc, Task: t,
			})).Error().To(MatchError(ContainSubstring("open")))
		})
	})

	Describe("Start and Stop", func() {
		It("Should open once and answer both commands", func(ctx SpecContext) {
			r := open(waitRun)
			Expect(r.Start(ctx, "cmd-1")).To(Succeed())
			Expect(r.Start(ctx, "cmd-2")).To(Succeed())
			Expect(opened).To(Equal(1))
			stat := retrieve(ctx)
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
			Expect(stat.Details.Running).To(BeTrue())
			Expect(r.Exec(ctx, task.Command{Type: "stop", Key: "cmd-3"})).To(Succeed())
			stat = retrieve(ctx)
			Expect(stat.Details.Cmd).To(Equal("cmd-3"))
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Message).To(Equal("Task stopped successfully"))
		})

		It(
			"Should stop the run when the running status cannot be written",
			func(ctx SpecContext) {
				ended := make(chan struct{}, 2)
				r := open(func(ctx context.Context) error {
					defer func() { ended <- struct{}{} }()
					return waitRun(ctx)
				})
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				Expect(r.Start(cancelled, "cmd-1")).To(MatchError(context.Canceled))
				Eventually(ended).Should(Receive())
				Expect(r.Start(ctx, "cmd-2")).To(Succeed())
				Expect(opened).To(Equal(2))
			},
		)

		It("Should report a run that ends on its own", func(ctx SpecContext) {
			r := open(func(context.Context) error { return errors.New("boom") })
			Expect(r.Start(ctx, "cmd-1")).To(Succeed())
			Eventually(func(g Gomega) {
				stat := retrieve(ctx)
				g.Expect(stat.Variant).To(Equal(status.VariantError))
				g.Expect(stat.Message).To(Equal("boom"))
				g.Expect(stat.Details.Running).To(BeFalse())
			}).Should(Succeed())
		})
	})

	Describe("Report", func() {
		It("Should warn and restore the running status", func(ctx SpecContext) {
			r := open(waitRun)
			Expect(r.Start(ctx, "cmd-1")).To(Succeed())
			r.Report(ctx, errors.Wrap(driver.ErrTemporary, "flaky"))
			stat := retrieve(ctx)
			Expect(stat.Variant).To(Equal(status.VariantWarning))
			Expect(stat.Message).To(ContainSubstring("flaky"))
			Expect(stat.Details.Running).To(BeTrue())
			r.Report(ctx, nil)
			stat = retrieve(ctx)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Message).To(Equal("Task started successfully"))
		})
	})
})
