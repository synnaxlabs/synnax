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
	"sync"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
)

var errCommit = errors.New("commit failed")

// uncommittableDB is a kv.DB whose transactions fail to commit.
type uncommittableDB struct{ kv.DB }

func (d uncommittableDB) OpenTx() kv.Tx { return uncommittableTx{Tx: d.DB.OpenTx()} }

type uncommittableTx struct{ kv.Tx }

func (uncommittableTx) Commit(context.Context, ...any) error { return errCommit }

// cancelingDB is a kv.DB whose transactions call cancel once they commit.
type cancelingDB struct {
	kv.DB
	cancel context.CancelFunc
}

func (d cancelingDB) OpenTx() kv.Tx { return cancelingTx{Tx: d.DB.OpenTx(), db: d} }

type cancelingTx struct {
	kv.Tx
	db cancelingDB
}

func (t cancelingTx) Commit(ctx context.Context, opts ...any) error {
	err := t.Tx.Commit(ctx, opts...)
	t.db.cancel()
	return err
}

// gatedDB is a kv.DB that, once gated, holds the next commit until release closes.
type gatedDB struct {
	kv.DB
	gated   atomic.Bool
	held    chan struct{}
	release chan struct{}
}

func (d *gatedDB) OpenTx() kv.Tx { return gatedTx{Tx: d.DB.OpenTx(), db: d} }

type gatedTx struct {
	kv.Tx
	db *gatedDB
}

func (t gatedTx) Commit(ctx context.Context, opts ...any) error {
	if t.db.gated.CompareAndSwap(true, false) {
		close(t.db.held)
		<-t.db.release
	}
	return t.Tx.Commit(ctx, opts...)
}

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

	openOn := func(statusDB *gorp.DB, run func(context.Context) error) *driver.Runner {
		GinkgoHelper()
		r := MustSucceed(driver.NewRunner(driver.RunnerConfig{
			DB:     statusDB,
			Status: statusSvc,
			Task:   t,
			Open: func(context.Context) (func(context.Context) error, error) {
				opened++
				return run, nil
			},
		}))
		DeferCleanup(func() { Expect(r.Stop(false)).To(Succeed()) })
		return r
	}
	open := func(run func(context.Context) error) *driver.Runner {
		GinkgoHelper()
		return openOn(db, run)
	}

	retrieve := func(ctx context.Context) task.Status {
		GinkgoHelper()
		return retrieveStatus[task.StatusDetails](ctx, t.OntologyID().String())
	}

	Describe("NewRunner", func() {
		It("Should reject a configuration with no open", func() {
			Expect(driver.NewRunner(driver.RunnerConfig{
				DB:     db,
				Status: statusSvc,
				Task:   t,
			})).Error().To(MatchError(ContainSubstring("open: must be non-nil")))
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
			Expect(stat.Message).To(Equal("Stopped task"))
		})

		It(
			"Should stop the run when the running status cannot be written",
			func(ctx SpecContext) {
				ended := make(chan struct{}, 2)
				unwritable := gorp.Wrap(uncommittableDB{DB: db.KV()})
				r := openOn(unwritable, func(ctx context.Context) error {
					defer func() { ended <- struct{}{} }()
					return waitRun(ctx)
				})
				Expect(r.Start(ctx, "cmd-1")).To(MatchError(errCommit))
				Eventually(ended).Should(Receive())
				Expect(r.Start(ctx, "cmd-2")).To(MatchError(errCommit))
				Expect(opened).To(Equal(2))
			},
		)

		It("Should answer the start with the error of a failed open",
			func(ctx SpecContext) {
				r := MustSucceed(driver.NewRunner(driver.RunnerConfig{
					DB:     db,
					Status: statusSvc,
					Task:   t,
					Open: func(context.Context) (func(context.Context) error, error) {
						return nil, errors.New("port is in use")
					},
				}))
				Expect(r.Start(ctx, "cmd-1")).To(MatchError("port is in use"))
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to start task"))
				Expect(stat.Description).To(Equal("port is in use"))
				Expect(stat.Details.Cmd).To(Equal("cmd-1"))
				Expect(stat.Details.Running).To(BeFalse())
			},
		)

		It("Should return no error from Exec for a start it answered",
			func(ctx SpecContext) {
				r := MustSucceed(driver.NewRunner(driver.RunnerConfig{
					DB:     db,
					Status: statusSvc,
					Task:   t,
					Open: func(context.Context) (func(context.Context) error, error) {
						return nil, errors.New("port is in use")
					},
				}))
				Expect(r.Exec(ctx, task.Command{Type: "start", Key: "cmd-1"})).
					To(Succeed())
				stat := retrieve(ctx)
				Expect(stat.Message).To(Equal("Failed to start task"))
				Expect(stat.Details.Cmd).To(Equal("cmd-1"))
			},
		)

		It("Should return no error from Exec for a start answered before ctx ended",
			func(ctx SpecContext) {
				cmdCtx, cancel := context.WithCancel(ctx)
				r := MustSucceed(driver.NewRunner(driver.RunnerConfig{
					DB:     gorp.Wrap(cancelingDB{DB: db.KV(), cancel: cancel}),
					Status: statusSvc,
					Task:   t,
					Open: func(context.Context) (func(context.Context) error, error) {
						return nil, errors.New("port is in use")
					},
				}))
				Expect(r.Exec(cmdCtx, task.Command{Type: "start", Key: "cmd-1"})).
					To(Succeed())
				Expect(cmdCtx.Err()).To(MatchError(context.Canceled))
				stat := retrieve(ctx)
				Expect(stat.Message).To(Equal("Failed to start task"))
				Expect(stat.Details.Cmd).To(Equal("cmd-1"))
			},
		)

		DescribeTable(
			"Should return the error from Exec for a command it could not answer",
			func(ctx SpecContext, cmdType string) {
				unwritable := gorp.Wrap(uncommittableDB{DB: db.KV()})
				r := openOn(unwritable, waitRun)
				Expect(r.Exec(ctx, task.Command{Type: cmdType, Key: "cmd-1"})).
					To(MatchError(errCommit))
			},
			Entry("start", "start"),
			Entry("stop", "stop"),
		)

		It("Should report a run that ends on its own", func(ctx SpecContext) {
			r := open(func(context.Context) error { return errors.New("boom") })
			Expect(r.Start(ctx, "cmd-1")).To(Succeed())
			Eventually(func(g Gomega) {
				stat := retrieve(ctx)
				g.Expect(stat.Variant).To(Equal(status.VariantError))
				g.Expect(stat.Message).To(Equal("Failed to run task"))
				g.Expect(stat.Description).To(Equal("boom"))
				g.Expect(stat.Details.Running).To(BeFalse())
			}).Should(Succeed())
		})

		It("Should report a run that panics, and open a new run on the next start",
			func(ctx SpecContext) {
				// Only the first run panics, so the stop at cleanup ends a live run.
				var runs atomic.Int32
				r := open(func(ctx context.Context) error {
					if runs.Add(1) == 1 {
						panic("decoder broke")
					}
					return waitRun(ctx)
				})
				Expect(r.Start(ctx, "cmd-1")).To(Succeed())
				Eventually(func(g Gomega) {
					stat := retrieve(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("Failed to run task"))
					g.Expect(stat.Description).To(Equal("task panicked: decoder broke"))
					g.Expect(stat.Details.Running).To(BeFalse())
				}).Should(Succeed())
				Expect(r.Start(ctx, "cmd-2")).To(Succeed())
				Expect(opened).To(Equal(2))
			},
		)

		It("Should report a start as running after the status of a run that ended",
			func(ctx SpecContext) {
				gated := &gatedDB{
					DB:      db.KV(),
					held:    make(chan struct{}),
					release: make(chan struct{}),
				}
				fail := make(chan struct{})
				var runs atomic.Int32
				r := openOn(gorp.Wrap(gated), func(ctx context.Context) error {
					if runs.Add(1) > 1 {
						return waitRun(ctx)
					}
					select {
					case <-fail:
						return errors.New("boom")
					case <-ctx.Done():
						return nil
					}
				})
				// Cleanup runs in reverse, so a failed spec opens the gate before the
				// stop waits on the run that holds it.
				release := sync.OnceFunc(func() { close(gated.release) })
				DeferCleanup(release)
				Expect(r.Start(ctx, "cmd-1")).To(Succeed())
				gated.gated.Store(true)
				close(fail)
				Eventually(gated.held).Should(BeClosed())
				started := make(chan error, 1)
				go func() { started <- r.Start(ctx, "cmd-2") }()
				Consistently(started).ShouldNot(Receive())
				release()
				Eventually(started).Should(Receive(BeNil()))
				stat := retrieve(ctx)
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Cmd).To(Equal("cmd-2"))
			},
		)

		Describe("Given a run that fails while a repeat start writes its answer",
			func() {
				// failDuringAnswer holds the commit of the answer to a repeat start
				// until release runs, and ends the run during that hold.
				failDuringAnswer := func(ctx context.Context) (
					r *driver.Runner,
					started chan error,
					release func(),
				) {
					GinkgoHelper()
					gated := &gatedDB{
						DB:      db.KV(),
						held:    make(chan struct{}),
						release: make(chan struct{}),
					}
					fail, ended := make(chan struct{}), make(chan struct{})
					r = openOn(gorp.Wrap(gated), func(ctx context.Context) error {
						defer close(ended)
						select {
						case <-fail:
							return errors.New("boom")
						case <-ctx.Done():
							return nil
						}
					})
					// Cleanup runs in reverse, so a failed spec opens the gate before
					// the stop waits on the start that holds it.
					release = sync.OnceFunc(func() { close(gated.release) })
					DeferCleanup(release)
					Expect(r.Start(ctx, "cmd-1")).To(Succeed())
					gated.gated.Store(true)
					started = make(chan error, 1)
					go func() { started <- r.Start(ctx, "cmd-2") }()
					Eventually(gated.held).Should(BeClosed())
					close(fail)
					Eventually(ended).Should(BeClosed())
					return r, started, release
				}

				It("Should keep the error status of the run", func(ctx SpecContext) {
					_, started, release := failDuringAnswer(ctx)
					Consistently(started).ShouldNot(Receive())
					release()
					Eventually(started).Should(Receive(BeNil()))
					Eventually(func(g Gomega) {
						stat := retrieve(ctx)
						g.Expect(stat.Variant).To(Equal(status.VariantError))
						g.Expect(stat.Message).To(Equal("Failed to run task"))
						g.Expect(stat.Details.Running).To(BeFalse())
					}).Should(Succeed())
				})

				It("Should let a stop finish once the answer commits",
					func(ctx SpecContext) {
						r, started, release := failDuringAnswer(ctx)
						stopped := make(chan error, 1)
						go func() { stopped <- r.Stop(false) }()
						Consistently(stopped).ShouldNot(Receive())
						release()
						Eventually(started).Should(Receive(BeNil()))
						// A stop that passes the end of the run returns its error.
						result := Or(BeNil(), MatchError("boom"))
						Eventually(stopped).Should(Receive(result))
					},
				)
			},
		)
	})

	Describe("Report", func() {
		It("Should warn with the text of any error and restore the running status",
			func(ctx SpecContext) {
				r := open(waitRun)
				Expect(r.Start(ctx, "cmd-1")).To(Succeed())
				r.Report(ctx, "Failed to read data", errors.New("flaky"))
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Failed to read data"))
				Expect(stat.Description).To(Equal("flaky"))
				Expect(stat.Details.Running).To(BeTrue())
				r.Report(ctx, "Failed to read data", nil)
				stat = retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantSuccess))
				Expect(stat.Message).To(Equal("Started task"))
				Expect(stat.Description).To(BeEmpty())
			},
		)

		It("Should warn with the text and data of a Warning, and restore the data",
			func(ctx SpecContext) {
				r := open(waitRun)
				Expect(r.Start(ctx, "cmd-1")).To(Succeed())
				r.Report(ctx, "Failed to read data", driver.Warning{
					Message:     "Skipped 1 of 2 records",
					Description: "Record is not JSON",
					Data:        msgpack.EncodedJSON{"stats": "counts"},
				}.Degraded())
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Skipped 1 of 2 records"))
				Expect(stat.Description).To(Equal("Record is not JSON"))
				Expect(stat.Details.Data).To(HaveKeyWithValue("stats", "counts"))
				r.Report(ctx, "Failed to read data", nil)
				stat = retrieve(ctx)
				Expect(stat.Message).To(Equal("Started task"))
				Expect(stat.Details.Data).To(BeNil())
			},
		)
	})

	Describe("ReportError", func() {
		It("Should write an error status that Report does not clear",
			func(ctx SpecContext) {
				r := open(waitRun)
				Expect(r.Start(ctx, "cmd-1")).To(Succeed())
				r.ReportError(ctx, "Failed to send event", "outage")
				stat := retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to send event"))
				Expect(stat.Description).To(Equal("outage"))
				Expect(stat.Details.Running).To(BeTrue())
				Expect(stat.Details.Cmd).To(Equal(driver.NoCommand))
				r.Report(ctx, "Failed to read data", nil)
				stat = retrieve(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("Failed to send event"))
				Expect(stat.Description).To(Equal("outage"))
			},
		)
	})
})
